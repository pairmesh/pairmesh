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

package api

import (
	"time"

	"github.com/pairmesh/pairmesh/errcode"

	"github.com/pairmesh/pairmesh/portal/db"
	"github.com/pairmesh/pairmesh/portal/db/models"
	"github.com/pingcap/fn"
	"gorm.io/gorm"
)

type (
	networkUserInfo struct {
		networkID models.ID
		userID    models.ID
		role      models.RoleType
		expiredAt *time.Time
	}
	networkDeviceInfo struct {
		networkID models.ID
		userID    models.ID
		deviceID  models.ID
		expiredAt *time.Time
	}
)

func (s *server) createNetworkUser(tx *gorm.DB, info *networkUserInfo) error {
	var networkUser models.NetworkUser
	query := models.NewNetworkUserQuerySet(tx).NetworkIDEq(info.networkID).UserIDEq(info.userID)
	err := query.One(&networkUser)
	switch err {
	case nil:
		if networkUser.ExpiredAt != nil && networkUser.ExpiredAt.Before(time.Now()) {
			err = models.NewNetworkDeviceQuerySet(tx).NetworkIDEq(info.networkID).UserIDEq(info.userID).Delete()
			if err != nil && err != gorm.ErrRecordNotFound {
				return err
			}
		}
		return query.GetUpdater().SetRole(info.role).SetExpiredAt(info.expiredAt).Update()
	case gorm.ErrRecordNotFound:
		createUser := &models.NetworkUser{
			NetworkID: info.networkID,
			UserID:    info.userID,
			Role:      info.role,
			ExpiredAt: info.expiredAt,
		}
		return db.Create(createUser)
	default:
		return err
	}
}

func (s *server) createNetworkDevice(tx *gorm.DB, info *networkDeviceInfo) error {
	var networkDevice models.NetworkDevice
	query := models.NewNetworkDeviceQuerySet(tx).NetworkIDEq(info.networkID).UserIDEq(info.userID).DeviceIDEq(info.deviceID)
	err := query.One(&networkDevice)
	switch err {
	case nil:
		return query.GetUpdater().SetExpiredAt(info.expiredAt).Update()
	case gorm.ErrRecordNotFound:
		createDevice := &models.NetworkDevice{
			NetworkID: info.networkID,
			UserID:    info.userID,
			DeviceID:  info.deviceID,
			ExpiredAt: info.expiredAt,
		}
		return db.Create(createDevice)
	default:
		return err
	}
}

type VersionCheckResponse struct {
	NewVersion      bool   `json:"new_version"`
	NewVersionCode  string `json:"new_version_code"`
	DownloadAddress string `json:"download_address"`
}

func (s *server) VersionCheck(form *fn.Form) (*VersionCheckResponse, error) {
	version := form.Get("version")
	platform := form.Get("platform")
	if version == "" || platform == "" {
		return nil, errcode.ErrIllegalRequest
	}
	//TODO
	return nil, nil
}
