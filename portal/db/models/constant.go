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

const MaxOwnDevice = 10

const AssumeOnlineDuration = time.Second * 120

type RoleType string

const (
	RoleTypeOwner  RoleType = "owner"
	RoleTypeAdmin  RoleType = "admin"
	RoleTypeMember RoleType = "member"
)

// String implements the fmt.Stringer interface
func (r RoleType) String() string {
	return string(r)
}

type KeyType string

const (
	KeyTypeOneOff    KeyType = "one-off"
	KeyTypeReusable  KeyType = "reusable"
	KeyTypeEphemeral KeyType = "ephemeral"
)

// String implements the fmt.Stringer interface
func (k KeyType) String() string {
	return string(k)
}

type DeviceStatusType string

const (
	DeviceStatusTypeOnline  DeviceStatusType = "online"
	DeviceStatusTypeOffline DeviceStatusType = "offline"
)

// String implements the fmt.Stringer interface
func (d DeviceStatusType) String() string {
	return string(d)
}
