package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"time"
)

func read_register(addr string, station byte, register uint16) (uint16, error) {
	rr, err := read_registers(addr, station, register, 1)
	if err != nil {
		return 0, err
	}
	return rr[0], nil
}

func read_registers(addr string, station byte, register uint16, count int) ([]uint16, error) {
	b := make([]byte, 5)
	b[0] = byte(3) // function code
	binary.BigEndian.PutUint16(b[1:], register)
	binary.BigEndian.PutUint16(b[3:], uint16(count))
	r, err := ModbusExchange(addr, station, b)
	if err != nil {
		return nil, err
	}
	if int(r[1]) != count*2 {
		return nil, errors.New("Response length does not match count")
	}
	rr := make([]uint16, count)
	for i := range count {
		rr[i] = binary.BigEndian.Uint16(r[2+i*2:])
	}
	return rr, nil
}

func write_register(addr string, station byte, register uint16, value uint16) error {
	b := make([]byte, 5)
	b[0] = byte(6) // function code
	binary.BigEndian.PutUint16(b[1:], register)
	binary.BigEndian.PutUint16(b[3:], value)
	r, err := ModbusExchange(addr, station, b)
	if err != nil {
		return err
	}
	if binary.BigEndian.Uint16(r[1:]) != register {
		return errors.New("Response: unexpected register")
	}
	if binary.BigEndian.Uint16(r[3:]) != value {
		return errors.New("Response: unexpected length")
	}
	return nil
}

func write_registers(addr string, station byte, register uint16, values []uint16) error {
	b := make([]byte, 6+len(values)*2)
	b[0] = byte(16) // function code
	binary.BigEndian.PutUint16(b[1:], register)
	binary.BigEndian.PutUint16(b[3:], uint16(len(values)))
	b[5] = byte(len(values) * 2)
	for i := range len(values) {
		binary.BigEndian.PutUint16(b[6+i*2:], values[i])
	}
	r, err := ModbusExchange(addr, station, b)
	if err != nil {
		return err
	}
	if binary.BigEndian.Uint16(r[1:]) != register {
		return errors.New("Response: unexpected register")
	}
	if binary.BigEndian.Uint16(r[3:]) != uint16(len(values)) {
		return errors.New("Response: unexpected length")
	}
	return nil
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
