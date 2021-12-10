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
	"fmt"

	"github.com/pairmesh/pairmesh/node/device/firewall"
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
	// Set the IP address for the virtual interface
	setAddress := []string{
		"netsh",
		"interface",
		"ip",
		"set",
		"address",
		fmt.Sprintf(`name="%s"`, d.Name()),
		fmt.Sprintf("addr=%s", address),
		"gateway=none",
	}
	err := runner.Run(setAddress)
	if err != nil {
		return err
	}

	upDevice := []string{
		"netsh",
		"interface",
		"set",
		"interface",
		fmt.Sprintf("\"%s\"", d.Name()),
		"enable",
	}

	// Async setup firewall rules when process startup.
	go firewall.Setup(address)

	return runner.Run(upDevice)
}

// Down implements the Device interface.
func (d device) Down() error {
	downDevice := []string{
		"netsh",
		"interface",
		"set",
		"interface",
		fmt.Sprintf("\"%s\"", d.Name()),
		"disable",
	}

	return runner.Run(downDevice)
}
