// Copyright 2021 PairMesh, Inc.
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

package queryutil

import (
	"testing"

	"inet.af/netaddr"
)

func TestIPReserved(t *testing.T) {
	ss := []netaddr.IP{
		netaddr.IPv4(100, 100, 100, 100),
		netaddr.IPv4(100, 101, 102, 103),
		netaddr.IPv4(100, 99, 98, 97),

		netaddr.IPv4(100, 99, 99, 99),
		netaddr.IPv4(100, 98, 97, 96),
		netaddr.IPv4(100, 96, 97, 98),
	}

	for _, v := range ss {
		if !isReservedIP(v) {
			t.Fatal("ip:" + v.String())
		}
	}

	ss = []netaddr.IP{

		netaddr.IPv4(100, 98, 96, 94),
		netaddr.IPv4(100, 94, 96, 98),
		netaddr.IPv4(100, 94, 97, 98),
	}
	for _, v := range ss {
		if isReservedIP(v) {
			t.Fatal("ip:" + v.String())
		}
	}
}
