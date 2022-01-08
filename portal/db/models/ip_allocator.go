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

package models

import (
	"fmt"
	"math/rand"

	"gorm.io/gorm"
	"inet.af/netaddr"
)

const maxRetry = 200

// NextIP retrieve the next available IP address
// TODO: maybe use algorithm like
func NextIP(tx *gorm.DB) (string, error) {
	n := func(n int) byte { return byte(rand.Intn(n)) }
	var nextIP string
	var device Device
	for i := 0; i < maxRetry; i++ {
		ip := netaddr.IPFrom4([4]byte{10, n(256), n(256), 2 + n(253)})
		nextIP = ip.String()
		err := NewDeviceQuerySet(tx).AddressEq(nextIP).One(&device)
		if err == gorm.ErrRecordNotFound {
			return nextIP, nil
		}
	}
	return "", fmt.Errorf("found IP reach max retry count")
}
