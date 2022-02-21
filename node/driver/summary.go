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

package driver

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/pairmesh/pairmesh/node/mesh"
)

const mockSummary = false

type Summary struct {
	Enabled bool          `json:"enabled"`
	Status  string        `json:"status"`
	Profile *Profile      `json:"profile"`
	Mesh    *mesh.Summary `json:"mesh"`
}

type Profile struct {
	UserID uint64 `json:"user_id"`
	Name   string `json:"name"`
	IPv4   string `json:"ipv4"`
}

// Equal returns true if the p equals rhs.
func (p *Profile) Equal(rhs *Profile) bool {
	// The same pointer or nil
	if p == rhs {
		return true
	}
	if p == nil && rhs != nil {
		return false
	}
	if p != nil && rhs == nil {
		return false
	}
	return p.UserID == rhs.UserID && p.Name == rhs.Name && p.IPv4 == rhs.IPv4
}

// Equal returns true if the s equals rhs.
func (s *Summary) Equal(rhs *Summary) bool {
	// The same pointer or nil
	if s == rhs {
		return true
	}
	if s == nil && rhs != nil {
		return false
	}
	if s != nil && rhs == nil {
		return false
	}
	return s.Enabled == rhs.Enabled && s.Profile.Equal(rhs.Profile) && s.Mesh.LastChangedAt == rhs.Mesh.LastChangedAt
}

// mockSummarize returns a mock summary for testing.
func (d *NodeDriver) mockSummarize() *Summary {
	networkStatus := []string{"connecting", "connected"}
	meshSummary := &mesh.Summary{
		LastChangedAt: time.Now(),
	}

	// Generate the mock device information randomly.
	deviceStatus := []mesh.State{mesh.StatePending, mesh.StateRelay, mesh.StateP2P}
	devCnt := 3
	for i := 0; i < devCnt; i++ {
		meshSummary.MyDevices = append(meshSummary.MyDevices, mesh.Device{
			Name:   fmt.Sprintf("mock-device-%d", i),
			IPv4:   fmt.Sprintf("10.0.12.%d", i),
			Status: deviceStatus[rand.Intn(len(deviceStatus))],
		})
	}

	// Generate the mock network information randomly.
	netCnt := 5
	for i := 0; i < netCnt; i++ {
		network := mesh.Network{
			ID:   uint64(i),
			Name: fmt.Sprintf("mock-network-%d", i),
		}
		devCnt := rand.Intn(50)
		for j := 0; j < devCnt; j++ {
			network.Devices = append(network.Devices, mesh.Device{
				Name:   fmt.Sprintf("mock-device-%d-%d", i, j),
				IPv4:   fmt.Sprintf("10.0.%d.%d", i, j),
				Status: deviceStatus[rand.Intn(len(deviceStatus))],
			})
		}
		meshSummary.Networks = append(meshSummary.Networks, network)
	}

	return &Summary{
		Enabled: d.enable.Load(),
		Status:  networkStatus[rand.Intn(len(networkStatus))],
		Profile: &Profile{
			UserID: uint64(d.userID),
			IPv4:   d.credential.address,
			Name:   d.name,
		},
		Mesh: meshSummary,
	}
}
