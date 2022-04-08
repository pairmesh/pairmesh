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

import "time"

// MaxOwnDevice is the max device numbers that one user could have
const MaxOwnDevice = 10

// AssumeOnlineDuration is the time that a device is assumed to be online
const AssumeOnlineDuration = time.Second * 120

// RoleType as string alias, represents the type of a role
type RoleType string

// RoleType represents type of a role
const (
	RoleTypeOwner  RoleType = "owner"
	RoleTypeAdmin  RoleType = "admin"
	RoleTypeMember RoleType = "member"
)

// String implements the fmt.Stringer interface
func (r RoleType) String() string {
	return string(r)
}

// KeyType as string alias, represents the type of the key
type KeyType string

// KeyType constants are values representing types of a key
const (
	KeyTypeOneOff    KeyType = "one-off"
	KeyTypeReusable  KeyType = "reusable"
	KeyTypeEphemeral KeyType = "ephemeral"
)

// String implements the fmt.Stringer interface
func (k KeyType) String() string {
	return string(k)
}

// DeviceStatusType represents the status of a device
type DeviceStatusType string

// DeviceStatusType constants are values representing device status
const (
	DeviceStatusTypeOnline  DeviceStatusType = "online"
	DeviceStatusTypeOffline DeviceStatusType = "offline"
)

// String implements the fmt.Stringer interface
func (d DeviceStatusType) String() string {
	return string(d)
}
