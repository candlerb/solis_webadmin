I honestly don't ever expect anyone to use this code!

This is a hack to integrate Home Assistant with my Solis inverter, so that
when Intelligent Octopus Go has an off-peak period outside of normal peak
times, the inverter is programmed to charge.

I didn't attempt to do this in HA directly because:
1. HA semantics are poorly documented and I was too scared
2. I wanted to use a compiled language rather than Python
3. I wanted to optimise it to minimise the number of modbus messages sent
   (to minimise risk of clashes)

I used HA to work out the peak/off-peak detection, as that's already handled
by the BottlecapDave integration, and I fought with Docker enough to make HA
run (running as standalone Python is no longer supported).  But in the end,
if I can't still understand HA enough, I might have to integrate directly
into the Octopus API.

## Logic

Every time the off_peak state changes, or the current_end time changes, an
update is sent over HTTP containing those two items of data.

I configure the boost programmes in the third timer slot. This leaves the
other two slots for regular timed activity, including the 23:30-05:30 charge.

The logic is:

* When we receive an update and the new state is "on", and there is
  currently no programme set or it has already ended, and the
  current time is *not* between 23:30 and 05:30:
    * calculate ept = min(current_end, now + 4 hours, next 23:30)
    * If ept is more than 5 minutes in the future:
        - programme a charge from now to ept (and remember programme end time)

* When we receive an update and the new state is "off", and there is
  currently a programme set which ends at least 5 minutes in the future:
    * zero the programme (i.e. set it to 00:00-00:00)
    * (This is to cancel a boost where the end time has been brought
      forward)

* When the programme end time comes along, and the most recent state is
  "on", and the current time is not between 23:30 and 05:30:
    * calculate ept = min(current_end, now + 4 hours, next 23:30)
    * If ept is more than 5 minutes away:
        - programme a charge from now to ept (and remember programme end time)
    * (i.e. if the end time has been made later, we extend the program by
       setting a new one)

* At 23:30 every day, if a programme is set:
    * zero the programme. This prevents the programme from repeating
      tomorrow.

* At 05:30 every day, if the most recent state is "on":
    * calculate ept = min(current_end, now + 4 hours, next 23:30)
    * If ept is more than 5 minutes away:
        - programme a charge from now to ept (and remember programme end time)
    * (in other words, deal with an off_peak period which started before
          05:30 and extends beyond 05:30)

I also like to charge the battery slowly from 23:30 to 05:30, so I set the
charge rate to high on the first boost programme, and set it back to low
rate at 23:30 if it has been changed.

With this logic, we might only programme the timer once for each charge
boost and once at 23:30 to cancel it (plus the charge rate changes).

## Installation

To make this work, you run solis_boost somewhere, and you point Home
Assistant to it like this, by enabling the
[RESTful command](https://www.home-assistant.io/integrations/rest_command/)
integration in `configuration.yaml' file:

```
rest_command:
  solis_boost:
    url: "http://solis.home.deploy2.net:8533/boost"
    method: post
    content_type: application/x-www-form-urlencoded
    payload: |
      {% set off_peak = states.binary_sensor.octopus_energy_electricity_XXXXXXXXXX_XXXXXXXXXXXXX_off_peak -%}
      {{ urlencode(dict(
        state=off_peak.state,
        current_end=off_peak.attributes.current_end.isoformat() if off_peak.attributes.current_end,
      )).decode('utf-8') }}
```

Then create an automation which triggers on the Octopus off_peak sensor state
or `current_end` attribute:

```
alias: Solis off-peak boost
description: ""
triggers:
  - trigger: state
    entity_id:
      - >-
        binary_sensor.octopus_energy_electricity_XXXXXXXXXX_XXXXXXXXXXXXX_off_peak
    from: null
    to: null
  - trigger: state
    entity_id:
      - >-
        binary_sensor.octopus_energy_electricity_XXXXXXXXXX_XXXXXXXXXXXXX_off_peak
    attribute: current_end
conditions: []
actions:
  - action: rest_command.solis_boost
    metadata: {}
    data: {}
mode: single
```

## Test

You can test using curl. Beware that "+" in a URL is converted to a space,
which breaks parsing, so you need to urlencode it first:

```
curl --data state=on --data-urlencode 'current_end=2026-01-12T16:46:00+00:00' 'solis:8533/boost'
```
