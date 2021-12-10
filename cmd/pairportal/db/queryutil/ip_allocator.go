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
	"fmt"
	"math/rand"

	"github.com/pairmesh/pairmesh/cmd/pairportal/db/models"

	"gorm.io/gorm"
	"inet.af/netaddr"
)

const maxRetry = 200

/*
Network:   100.64.0.0/10        01100100.01 000000.00000000.00000000
HostMin:   100.64.0.1           01100100.01 000000.00000000.00000001
HostMax:   100.127.255.254      01100100.01 111111.11111111.11111110
Broadcast: 100.127.255.255      01100100.01 111111.11111111.11111111
Hosts/Net: 4194302               Class A
*/

// NextIP retrieve the next available IP address
// TODO: maybe use algorithm like
// github.com/juanfont/headscale ?
func NextIP(tx *gorm.DB) (string, error) {
	n := func(n int) byte { return byte(rand.Intn(n)) }
	var nextIP string
	var device models.Device
	for i := 0; i < maxRetry; i++ {
		ip := netaddr.IPFrom4([4]byte{100, 64 + n(64), n(256), 2 + n(253)})
		if isReservedIP(ip) {
			continue
		}
		nextIP = ip.String()
		err := models.NewDeviceQuerySet(tx).AddressEq(nextIP).One(&device)
		if err == gorm.ErrRecordNotFound {
			return nextIP, nil
		}
	}
	return "", fmt.Errorf("found IP reach max retry count")
}

func isReservedIP(ip netaddr.IP) bool {
	ipRaw := ip.As4()

	// A.A.A.A
	// 100.100.100.100
	if ipRaw[0] == ipRaw[1] && ipRaw[1] == ipRaw[2] && ipRaw[2] == ipRaw[3] {
		return true
	}

	// A.B.C.D
	// 100.101.102.103
	if ipRaw[1] == ipRaw[0]+1 && ipRaw[2] == ipRaw[1]+1 && ipRaw[3] == ipRaw[2]+1 {
		return true
	}

	// D.C.B.A
	// 100.99.98.97
	if ipRaw[3] == ipRaw[2]-1 && ipRaw[2] == ipRaw[1]-1 && ipRaw[1] == ipRaw[0]-1 {
		return true
	}

	// A.x.x.x
	// 100.2.2.2
	if ipRaw[1] == ipRaw[2] && ipRaw[2] == ipRaw[3] {
		return true
	}

	// A.x.y.z
	// 100.2.3.4
	if ipRaw[2] == ipRaw[1]+1 && ipRaw[3] == ipRaw[2]+1 {
		return true
	}

	// A.z.y.x
	// 100.4.3.2
	if ipRaw[3] == ipRaw[2]-1 && ipRaw[2] == ipRaw[1]-1 {
		return true
	}

	return false

}
