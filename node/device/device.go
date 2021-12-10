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

import "io"

// Device represents the abstraction of virtual network interface device.
// The Device interface has different implementation for cross platform.
// 1. Windows: use wintun-go project to manage layer-three virtual interface.
// 2. Linux: use /dev/tun file to create layer-three tun file descriptor.
// 3. macOS: use /dev/utun file to create layer-three tun file descriptor.
// 4. Android: use VPN service interface to create low-level resource.
// 5. iOS: use Network extension to create VPN service.
type Device interface {
	io.ReadWriteCloser

	// Name returns the name of the Peerly virtual network device.
	Name() string

	// Router returns the router of the current device.
	Router() Router

	// Up runs the device with the address of the virtual tunnel device.
	Up(ipv4 string) error

	// Down closed the virtual device
	Down() error
}
