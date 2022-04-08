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
	"time"
)

type (
	// ID represents all Identity
	ID uint64

	// Base is the base columns of all tables
	Base struct {
		ID        ID        `gorm:"not null"`
		CreatedAt time.Time `gorm:"not null"`
	}

	// Updatable is the base columns of updatable tables
	Updatable struct {
		Base
		//a time pointer which will mean it can be a nil value
		UpdatedAt *time.Time
	}

	// Deletable is the base columns of deletable tables
	Deletable struct {
		Updatable
		DeletedAt *time.Time
	}
)

// gen:qs
type (
	// User represents the user table in database
	User struct {
		Deletable

		// with a PairMesh user, it must be  unique & not null
		Email     string `gorm:"type:varchar(64)"`
		Avatar    string `gorm:"type:varchar(512)"`
		Name      string `gorm:"type:varchar(64);not null"`
		Salt      string `gorm:"type:varchar(64);not null"`
		Hash      string `gorm:"type:varchar(64);not null"`
		SecretKey string `gorm:"type:varchar(64);not null"`
		Origin    string `gorm:"type:enum('pairmesh','github','wechat');default:'pairmesh'"`
	}

	// Device represents a virtual network device
	Device struct {
		Deletable

		UserID        ID        `gorm:"not null;index"`
		User          *User     `gorm:"foreignkey:UserID"`
		RelayServerID ID        `gorm:"not null"`
		Name          string    `gorm:"type:varchar(128)"`
		OS            string    `gorm:"type:varchar(32);not null"`
		Version       string    `gorm:"type:varchar(32);not null"`
		MachineID     string    `gorm:"type:varchar(128);not null"`
		LastSeen      time.Time `gorm:"not null"`
		Address       string    `gorm:"type:varchar(32);not null"`
	}

	// Network represents a network
	Network struct {
		Deletable

		// The network is owned by the creator of network.
		CreatedByID ID     `gorm:"not null"`
		CreatedBy   *User  `gorm:"foreignkey:CreatedByID"`
		Name        string `gorm:"type:varchar(64);not null"`
		Description string `gorm:"type:varchar(256);not null"`
	}

	// NetworkUser is used to associate users to networks
	NetworkUser struct {
		Deletable

		UserID    ID       `gorm:"not null;index"`
		User      *User    `gorm:"foreignkey:UserID"`
		NetworkID ID       `gorm:"not null"`
		Network   *Network `gorm:"foreignkey:NetworkID"`
		Role      RoleType `gorm:"type:enum('owner','admin', 'member');default:'member'"`
	}

	// Invitation represents the invitation request from the team admin/owner
	Invitation struct {
		Deletable

		NetworkID   ID       `gorm:"not null"`
		Network     *Network `gorm:"foreignkey:NetworkID"`
		InvitedByID ID       `gorm:"not null"`
		InvitedBy   *User    `gorm:"foreignkey:InvitedByID"`
		UserID      ID       `gorm:"not null"`
		User        *User    `gorm:"foreignkey:UserID"`
		Role        RoleType `gorm:"type:enum('admin', 'member');default:'member'"`
	}

	// RelayServer describes a relay server packet relay node running within a RelayRe.
	RelayServer struct {
		Deletable
		Name        string    `gorm:"type:varchar(128);not null;unique"`
		Region      string    `gorm:"type:varchar(32);not null"`
		Host        string    `gorm:"type:varchar(64);not null"`
		Port        int       `gorm:"not null;default:0"`
		STUNPort    int       `gorm:"not null;default:0"`
		PublicKey   string    `gorm:"type:varchar(64);not null"`
		StartedAt   time.Time `gorm:"not null"`
		KeepaliveAt time.Time `gorm:"not null"`
	}

	// GithubUser represents the github_user table in database
	GithubUser struct {
		Deletable

		UserID    ID     `gorm:"not null;unique"`
		User      *User  `gorm:"foreignkey:UserID"`
		GithubID  ID     `gorm:"not null;unique"`
		Login     string `gorm:"type:varchar(128);not null;unique"`
		AvatarURL string `gorm:"type:varchar(512)"`
		Location  string `gorm:"type:varchar(128)"`
	}

	// WechatUser represents the wechat_user table in database
	WechatUser struct {
		Deletable

		UserID     ID     `gorm:"not null;unique"`
		User       *User  `gorm:"foreignkey:UserID"`
		UnionID    string `gorm:"type:varchar(128);not null;unique"`
		Nickname   string `gorm:"type:varchar(64);not null;unique"`
		HeadImgURL string `gorm:"type:varchar(512)"`
		City       string `gorm:"type:varchar(128)"`
	}

	// AuthKey stores the pre-authentication keys
	AuthKey struct {
		Deletable

		UserID    ID        `gorm:"not null"`
		User      *User     `gorm:"foreignkey:UserID"`
		Type      KeyType   `gorm:"not null"`
		Role      KeyType   `gorm:"type:enum('one-off','reusable');default:'reusable'"`
		Key       string    `gorm:"not null;unique"`
		MachineID string    `gorm:"type:varchar(128);"`
		ExpiredAt time.Time `gorm:"not null"`
		Enabled   bool      `gorm:"not null;default:TRUE"`
	}
)
