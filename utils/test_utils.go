// Copyright 2022 PairMesh, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"net"
	"time"
)

// WaitTime is the time threshold for WaitFor func
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
