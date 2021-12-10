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

const BandwidthForFree = 1 //unit: M

const MaxOwnDevice = 10

const MaxNetworkOwnByFree = 0
const MaxNetworkDeviceOwnByFree = 0
const MaxNetworkOwnByProfessional = 10

const AssumeOnlineDuration = time.Second * 120

const StatisticDataKeepDays = 7

type SubscriptionType string

const (
	SubscriptionTypeFree         SubscriptionType = "free"
	SubscriptionTypeProfessional SubscriptionType = "professional"
	SubscriptionTypeFlagship     SubscriptionType = "flagship"
)

// String implements the fmt.Stringer interface
func (s SubscriptionType) String() string {
	return string(s)
}

type SubscriptionDurationType string

const (
	SubscriptionDurationTypeQuarter  SubscriptionDurationType = "quarter"
	SubscriptionDurationTypeHalfYear SubscriptionDurationType = "half-year"
	SubscriptionDurationTypeOneYear  SubscriptionDurationType = "one-year"
)

// String implements the fmt.Stringer interface
func (s SubscriptionDurationType) String() string {
	return string(s)
}

type SubscriptionStatusType string

const (
	SubscriptionStatusTypeNormal   SubscriptionStatusType = "normal"
	SubscriptionStatusTypeExpire   SubscriptionStatusType = "expire"
	SubscriptionStatusTypeDeducted SubscriptionStatusType = "deducted"
)

// String implements the fmt.Stringer interface
func (s SubscriptionStatusType) String() string {
	return string(s)
}

type NetworkStatusType string

const (
	NetworkStatusTypeActive   NetworkStatusType = "active"
	NetworkStatusTypeInactive NetworkStatusType = "inactive"
	NetworkStatusTypeSuspend  NetworkStatusType = "suspend"
)

// String implements the fmt.Stringer interface
func (s NetworkStatusType) String() string {
	return string(s)
}

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

type InvitationStatusType string

const (
	InvitationStatusTypeAccept      InvitationStatusType = "accept"
	InvitationStatusTypeReject      InvitationStatusType = "reject"
	InvitationStatusTypeUnprocessed InvitationStatusType = "unprocessed"
	InvitationStatusTypeRevoke      InvitationStatusType = "revoke"
)

// String implements the fmt.Stringer interface
func (i InvitationStatusType) String() string {
	return string(i)
}

type OrderStatusType string

const (
	OrderStatusTypeUnpaid   OrderStatusType = "unpaid"
	OrderStatusTypeComplete OrderStatusType = "complete"
	OrderStatusTypeRevoke   OrderStatusType = "revoke"
)

// String implements the fmt.Stringer interface
func (o OrderStatusType) String() string {
	return string(o)
}

type PaymentPlatformType string

const (
	PaymentPlatformAlipay PaymentPlatformType = "alipay"
	PaymentPlatformWechat PaymentPlatformType = "wechat"
)

// String implements the fmt.Stringer interface
func (p PaymentPlatformType) String() string {
	return string(p)
}

type EmergencyLevelType uint

const (
	Emergency EmergencyLevelType = 1
	Warning   EmergencyLevelType = 2
	Recommend EmergencyLevelType = 3
	Prompt    EmergencyLevelType = 4
	Info      EmergencyLevelType = 5
)
