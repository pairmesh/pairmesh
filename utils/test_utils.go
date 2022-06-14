package utils

import (
	"net"
	"time"
)

const WAIT_TIME = 5

func WaitFor(f func() bool) bool {
	start := time.Now()
	for {
		if time.Since(start) > time.Second*WAIT_TIME {
			return false
		}
		if f() {
			return true
		}
	}
}

func WaitForServerUp(addr string) bool {
	var f = func() bool {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return false
		}
		conn.Close()
		return true
	}
	return WaitFor(f)
}
