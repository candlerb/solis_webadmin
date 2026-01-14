package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"time"
)

const (
	TIMED_CHARGE_RATE = 43141
	TIMER3            = 43163
)

type Request struct {
	state       bool
	current_end time.Time
}

var boostreq chan *Request

var optListen string
var optTarget string
var optStation uint
var optForceChargeRate uint
var optDefaultChargeRate uint

func init() {
	flag.StringVar(&optListen, "listen", ":8533", "Web listen address and port")
	flag.StringVar(&optTarget, "target", "127.0.0.1:502", "Modbus TCP target address and port")
	flag.UintVar(&optStation, "station", 1, "Modbus station ID")
	flag.UintVar(&optForceChargeRate, "boost-charge-rate", 700, "Boost charge rate (0.1A increments)")
	flag.UintVar(&optDefaultChargeRate, "default-charge-rate", 220, "Default charge rate (0.1A increments)")
	flag.Parse()
}

func main() {
	log.SetFlags(0) // Disable timestamps, systemd/journald adds its own
	boostreq = make(chan *Request)
	http.HandleFunc("/boost", BoostHandler)
	go boost_loop()
	log.Fatal(http.ListenAndServe(optListen, nil))
}

func HTTPError(w http.ResponseWriter, statusCode int, err error) {
	log.Printf("%d: %v", statusCode, err)
	w.WriteHeader(statusCode)
	w.Write([]byte(err.Error()))
}

func BoostHandler(w http.ResponseWriter, r *http.Request) {
	var nr Request

	state := r.FormValue("state")
	current_end := r.FormValue("current_end")
	log.Printf("Received update: state=%s current_end=%s", state, current_end)
	if state == "on" {
		nr.state = true
	} else if state == "off" {
		nr.state = false
	} else {
		HTTPError(w, http.StatusBadRequest, errors.New("Invalid request (state)"))
		return
	}
	// We do see state changes where state=off and current_end is unset.
	// In that case, leave current_end as the zero value.
	if current_end != "" && current_end != "None" {
		if ce, err := time.Parse(time.RFC3339, current_end); err == nil {
			nr.current_end = ce
		} else {
			HTTPError(w, http.StatusBadRequest, err)
			return
		}
	}
	boostreq <- &nr
}

func boost_loop() {
	// Start by reading the current charge rate. This tests out modbus
	current_charge_rate, err := read_register(optTarget, byte(optStation), TIMED_CHARGE_RATE)
	if err != nil {
		log.Fatalf("Error reading timed charge rate (reg TIMED_CHARGE_RATE): %s", err)
	}
	log.Printf("Initial timed charge rate: %d", current_charge_rate)

	// Last update message received from HA (default is zero message, i.e. offpeak is "off")
	lu := &Request{}

	// If we have set a programme, remember when it ends and wakeup when it does
	var time_prog_end time.Time
	var chan_prog_end <-chan time.Time

	// Scheduled wakeups
	chan_23_30 := make_wakeup_chan(23, 30, 0)
	chan_05_30 := make_wakeup_chan(5, 30, 0)

	// Simple retry logic
	var retry <-chan time.Time

	set_programme := func(upto time.Time) error {
		var err error
		now := time.Now()
		// Work out the end time of this programme
		ept := upto
		// Stop at 23:30, when our regular nightly timer takes over.
		// Actually: stop at 23:00. There's little benefit in charging from
		// 23:00 to 23:30; if the battery empties then we'll be using grid
		// anyway. Stopping early reduces risk of charging when Octopus
		// decide to make this a peak slot despite dispatching; and if
		// the charge is *only* from 23:00 to 23:30, then we avoid
		// programming it entirely.
		if limit := today_clock_time(now, 23, 0, 0); ept.After(limit) {
			ept = limit
		}
		// Maximum 4 hours for safety
		if limit := now.Add(4 * time.Hour); ept.After(limit) {
			ept = limit
		}

		// Only apply the programme if it ends at least 5 minutes in the future
		if ept.After(now.Add(5 * time.Minute)) {
			if current_charge_rate != uint16(optForceChargeRate) {
				err = set_timed_charge_rate(uint16(optForceChargeRate))
				if err == nil {
					current_charge_rate = uint16(optForceChargeRate)
				}
			}
			err = set_timer3(now, ept)
			if err == nil {
				time_prog_end = ept
				chan_prog_end = time.After(ept.Sub(now))
			}
		} else {
			log.Printf("Not setting programme: end time would be %s", ept)
		}
		return err
	}

	unset_programme := func() error {
		err := set_timer3(time.Time{}, time.Time{})
		if err == nil {
			time_prog_end = time.Time{}
			chan_prog_end = nil
		}
		return err
	}

	update := func() {
		// we are looking at lu (last update), which might have been received some time in the past
		now := time.Now()
		if lu.state == true && !lu.current_end.IsZero() && lu.current_end.After(now) {
			log.Printf("off_peak is active")
			// prog already configured, and ends in the future? Wait until it ends
			// before we think about changing it
			if !time_prog_end.IsZero() && time_prog_end.After(now) {
				log.Printf("Existing programme is active")
				return
			}
			// is it between 23:29 and 05:29? Ignore, deal with it at 05:30 wakeup
			if now.Hour() < 5 ||
				(now.Hour() == 5 && now.Minute() < 30) ||
				(now.Hour() == 23 && now.Minute() >= 29) {
				log.Printf("Already in off-peak period")
				return
			}
			// otherwise, start a programme
			if set_programme(lu.current_end) != nil {
				retry = time.After(5 * time.Minute)
			}
		} else {
			log.Printf("off_peak is inactive")
			// If time_prog_end is set and more than 5 minutes in the future, then
			// forcibly terminate the current programme, by zeroing it
			if !time_prog_end.IsZero() && time_prog_end.After(now.Add(5*time.Minute)) {
				log.Printf("Terminating programme early")
				if unset_programme() != nil {
					retry = time.After(5 * time.Minute)
				}
			}
		}
	}

	// Force inverter into known state, also validates modbus write is working
	err = unset_programme()
	if err != nil {
		log.Fatalf("Error zeroing programme: %s", err)
	}

	for {
		select {
		case r := <-boostreq:
			lu = r
			// lu.state=on: start a new programme, unless one is already running,
			// and as long as the new programme would run for at last 5 minutes.
			// lu.state=off: if a programme is already running then terminate it,
			// unless it would already end in the next 5 minutes.
			update()

		case <-chan_prog_end:
			log.Printf("Programme end wakeup")
			// If solis programme has ended but the current desired state is "on"
			// (and ends at least 5 mins in the future) then write a new programme.
			update()

		case <-retry:
			log.Printf("Retry")
			retry = nil
			update()

		case <-chan_23_30:
			log.Printf("23:30 wakeup")
			chan_23_30 = make_wakeup_chan(23, 30, 0)
			// Zero any programme we made, to make sure it doesn't trigger tomorrow
			if !time_prog_end.IsZero() {
				if unset_programme() != nil {
					// Retry
					chan_23_30 = time.After(10 * time.Minute)
				}
			}
			// Reset charge rate to default rate if required
			if current_charge_rate != uint16(optDefaultChargeRate) {
				if set_timed_charge_rate(uint16(optDefaultChargeRate)) == nil {
					current_charge_rate = uint16(optDefaultChargeRate)
				}
			}

		case <-chan_05_30:
			log.Printf("05:30 wakeup")
			chan_05_30 = make_wakeup_chan(5, 30, 0)
			// If the off-peak window has ended but the current desired state is "on"
			// (and ends at least 5 mins in the future) then start a new programme.
			update()
		}
	}
}

// Create a wakeup channel at the next given clock time in the future
func make_wakeup_chan(hh, mm, ss int) <-chan time.Time {
	now := time.Now()
	t := today_clock_time(now, hh, mm, ss)
	if !t.After(now) {
		t = t.AddDate(0, 0, 1)
	}
	return time.After(t.Sub(now))
}

func today_clock_time(now time.Time, hh, mm, ss int) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day(), hh, mm, ss, 0, time.Local)
}

func set_timer3(t_from, t_to time.Time) error {
	var from_hh, from_mm, to_hh, to_mm int
	if !t_from.IsZero() {
		from_hh = t_from.Hour()
		from_mm = t_from.Minute()
	}
	if !t_to.IsZero() {
		to_hh = t_to.Hour()
		to_mm = t_to.Minute()
	}
	log.Printf("Set programme from %02d:%02d to %02d:%02d", from_hh, from_mm, to_hh, to_mm)
	err := write_registers(optTarget, byte(optStation), TIMER3, []uint16{
		uint16(from_hh),
		uint16(from_mm),
		uint16(to_hh),
		uint16(to_mm),
		//0, 0, 0, 0,   // doesn't appear to be necessary set discharge timer
	})
	if err != nil {
		log.Printf("ERROR setting timer: %s", err)
	}
	return err
}

func set_timed_charge_rate(rate uint16) error {
	log.Printf("Set timed charge rate to %d", rate)
	err := write_register(optTarget, byte(optStation), TIMED_CHARGE_RATE, rate)
	if err != nil {
		log.Printf("ERROR setting charge rate: %s", err)
	}
	return err
}
