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

package device

import (
	"github.com/pairmesh/pairmesh/node/device/runner"

	"github.com/pairmesh/pairmesh/node/device/tun"
)

var _ Device = &device{}

type device struct {
	tun.Device
}

// NewDevice constructs a new virtual network interface device.
func NewDevice() (Device, error) {
	dev, err := tun.NewTUN()
	if err != nil {
		return nil, err
	}
	return &device{Device: dev}, nil
}

// Router implements the Device interface
func (d device) Router() Router {
	return newRouter(d)
}

// Up implements the Device interface.
func (d device) Up(address string) error {
	// Set the IP address for the virtual interface and enable the device
	setAddressAndUp := []string{
		"ifconfig",
		d.Name(),
		"inet",
		address,
		address,
		"up",
	}
	return runner.Run(setAddressAndUp)
}

// Down implements the Device interface.
func (d device) Down() error {
	// Noting to do
	return nil
}
