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

	"github.com/pairmesh/pairmesh/cmd/pairmesh/device/runner"
	"github.com/pairmesh/pairmesh/cmd/pairmesh/device/tun"
)

var _ Device = &device{}

type device struct {
	tun.Device
}

// NewDevice constructs a new virtual network interface device.
func NewDevice() (Device, error) {
	dev, err := tun.NewTUN("pairmesh0")
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
		"ip",
		"addr",
		"add",
		fmt.Sprintf("%s/%d", address, 32),
		"dev",
		d.Name(),
	}
	err := runner.Run(setAddress)
	if err != nil {
		return err
	}

	// Up the virtual interface device
	upDevice := []string{
		"ip",
		"link",
		"set",
		"dev",
		d.Name(),
		"up",
	}

	return runner.Run(upDevice)
}

// Down implements the Device interface.
func (d device) Down() error {
	downDevice := []string{
		"ip",
		"link",
		"set",
		"dev",
		d.Name(),
		"down",
	}
	return runner.Run(downDevice)
}
