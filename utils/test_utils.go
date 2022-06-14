package utils

import (
	"net"
	"time"
)

// Wait time threshold for WaitFor func
const WaitTime = 5

func waitFor(f func() bool) bool {
	start := time.Now()
	for {
		if time.Since(start) > time.Second*WaitTime {
			return false
		}
		if f() {
			return true
		}
	}
}

// WaitForServerUp waits for a given server to be up
func WaitForServerUp(addr string) bool {
	var f = func() bool {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return false
		}
		conn.Close()
		return true
	}
	return waitFor(f)
}
