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
	"gorm.io/gorm"
)

// PeerDevices returns all peer devices of the specified user identifier.
func PeerDevices(tx *gorm.DB, userID ID) ([]Device, error) {
	var devices []Device
	tx.Raw(`
SELECT *
FROM devices
WHERE user_id IN
      (SELECT user_id
       FROM network_users
       WHERE network_id IN
             (SELECT network_id FROM network_users WHERE user_id = ?))
`, userID).Scan(&devices)

	return devices, tx.Error
}

// NetworkStats gets network stats info from database
func NetworkStats(tx *gorm.DB, networkID ID) (userCount, deviceCount int64, err error) {
	userCount, err = NewNetworkUserQuerySet(tx).NetworkIDEq(networkID).Count()
	if err != nil {
		return
	}

	tx.Raw("SELECT COUNT(*) FROM devices WHERE user_id IN (SELECT user_id FROM network_users WHERE network_id=?)", networkID).Scan(&deviceCount)
	err = tx.Error
	return
}
