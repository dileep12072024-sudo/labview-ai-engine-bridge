package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

// The bridge VI listens once and keeps the connection open, so the LabVIEW
// diagram needs a single TCP Listen outside its loop. We mirror that here with
// one persistent connection, reconnecting only when it breaks.
var lv struct {
	mu   sync.Mutex
	conn net.Conn
	rd   *bufio.Reader
}

func sendToLabVIEW(payload string) (string, error) {
	lv.mu.Lock()
	defer lv.mu.Unlock()

	ack, err := roundTrip(payload)
	if err == nil {
		return ack, nil
	}
	// One silent retry: LabVIEW drops the socket whenever the bridge VI is
	// stopped and re-run, which happens constantly while editing.
	closeConn()
	return roundTrip(payload)
}

func roundTrip(payload string) (string, error) {
	if lv.conn == nil {
		c, err := net.DialTimeout("tcp", *labviewAddr, 2*time.Second)
		if err != nil {
			return "", err
		}
		lv.conn, lv.rd = c, bufio.NewReader(c)
	}
	lv.conn.SetDeadline(time.Now().Add(30 * time.Second))
	// CRLF on the wire so the bridge VI can use TCP Read in CRLF mode, which is
	// one node instead of a hand-rolled buffer loop.
	wire := strings.TrimRight(payload, "\r\n") + "\r\n"
	if _, err := lv.conn.Write([]byte(wire)); err != nil {
		return "", err
	}
	line, err := lv.rd.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "ERR") {
		return "", fmt.Errorf("%s", strings.TrimSpace(strings.TrimPrefix(line, "ERR")))
	}
	return line, nil
}

func closeConn() {
	if lv.conn != nil {
		lv.conn.Close()
		lv.conn, lv.rd = nil, nil
	}
}

// startMockDaemon stands in for the LabVIEW bridge VI so the whole chain can be
// exercised before any LabVIEW work is done. It logs each command and acks it.
func startMockDaemon(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("mock daemon: %v (something is already on %s)", err, addr)
		return
	}
	log.Printf("mock LabVIEW daemon listening on %s", addr)
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go mockSession(c)
		}
	}()
}

func mockSession(c net.Conn) {
	defer c.Close()
	objects := []string{}
	sc := bufio.NewScanner(c)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		log.Printf("[mock labview] %s", line)
		f := fields(line)
		ack := "OK"
		switch f["CMD"] {
		case "PING":
			ack = "OK mock daemon (LabVIEW is NOT actually running)"
		case "NEWVI":
			objects = objects[:0]
			ack = "OK new VI " + f["NAME"]
		case "NEW":
			objects = append(objects, f["ID"]+" ("+f["STY"]+")")
			ack = fmt.Sprintf("OK placed %s at %s,%s", f["ID"], f["X"], f["Y"])
		case "WIRE":
			ack = fmt.Sprintf("OK wired %s -> %s", f["SRC"], f["DST"])
		case "LIST":
			ack = "OK " + strings.Join(objects, ", ")
		case "SAVE":
			ack = "OK saved " + f["PATH"]
		case "RUN":
			ack = "OK running"
		}
		fmt.Fprintln(c, ack)
	}
}

// fields parses one command line the same way the bridge VI must: split on ';',
// then split each field on its FIRST ':' only.
func fields(line string) map[string]string {
	m := map[string]string{}
	for _, f := range strings.Split(line, ";") {
		if k, v, ok := strings.Cut(f, ":"); ok {
			m[k] = v
		}
	}
	return m
}
