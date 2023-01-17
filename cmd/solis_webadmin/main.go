package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"time"
)

var optListen string
var optTarget string
var optStation int
var optStaticDir string

func init() {
	flag.StringVar(&optListen, "listen", "127.0.0.1:8502", "Web listen address and port")
	flag.StringVar(&optStaticDir, "staticdir", "static", "Path to static HTML/JS content")
	flag.StringVar(&optTarget, "target", "127.0.0.1:502", "Modbus TCP target address and port")
	flag.IntVar(&optStation, "station", 1, "Modbus station ID")
	flag.Parse()
}

func main() {
	http.Handle("/", http.FileServer(http.Dir(optStaticDir)))
	http.HandleFunc("/modbus", ModbusHandler)
	log.Fatal(http.ListenAndServe(optListen, nil))
}

func HTTPError(w http.ResponseWriter, statusCode int, err error) {
	log.Printf("%d: %v", statusCode, err)
	w.WriteHeader(statusCode)
	w.Write([]byte(err.Error()))
}

func ModbusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		HTTPError(w, http.StatusBadRequest, fmt.Errorf("Wrong HTTP method"))
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		HTTPError(w, http.StatusInternalServerError, err)
		return
	}
	log.Printf("Modbus request: %02x\n", body)
	resp, err := ModbusExchange(optTarget, byte(optStation), body)
	if err != nil {
		HTTPError(w, http.StatusInternalServerError, err)
		return
	}
	log.Printf("Modbus response: %02x\n", resp)
	w.Write(resp)
}

func ModbusExchange(addr string, station byte, request []byte) ([]byte, error) {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	hdr := []byte{
		byte(rand.Intn(256)),
		byte(rand.Intn(256)),
		0,
		0,
		byte((len(request) + 1) >> 8),
		byte((len(request) + 1) & 0xff),
		station,
	}
	hdr = append(hdr, request...)
	n, err := conn.Write(hdr)
	if err != nil {
		return nil, err
	}
	if n < len(hdr) {
		return nil, fmt.Errorf("Undersized write: %d instead of %d", n, len(hdr))
	}
	hdr2 := make([]byte, 7, 7)
	n, err = io.ReadFull(conn, hdr2)
	if err != nil {
		return nil, err
	}
	if hdr2[0] != hdr[0] || hdr2[1] != hdr[1] {
		return nil, fmt.Errorf("Mismatched TXID in response")
	}
	if hdr2[2] != 0 || hdr2[3] != 0 {
		return nil, fmt.Errorf("Bad proto in response")
	}
	if hdr2[6] != hdr[6] {
		return nil, fmt.Errorf("Mismatched STA in response")
	}
	l := ((int(hdr2[4]) << 8) | int(hdr2[5])) - 1
	if l < 1 {
		return nil, fmt.Errorf("Invalid read length")
	}
	resp := make([]byte, l, l)
	n, err = io.ReadFull(conn, resp)
	if err != nil {
		return nil, err
	}
	if n < l {
		return nil, fmt.Errorf("Undersized read: %d instead of %d", n, l)
	}
	return resp, nil
}
