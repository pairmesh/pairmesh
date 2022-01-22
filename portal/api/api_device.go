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
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pairmesh/pairmesh/errcode"
	"github.com/pairmesh/pairmesh/pkg/jwt"
	"github.com/pairmesh/pairmesh/portal/db"
	"github.com/pairmesh/pairmesh/portal/db/models"
	"gorm.io/gorm"
)

type (
	DeviceListItem struct {
		DeviceID models.ID               `json:"device_id"`
		Name     string                  `json:"name"`
		OS       string                  `json:"os"`
		Version  string                  `json:"version"`
		Address  string                  `json:"address"`
		LastSeen time.Time               `json:"last_seen"`
		Status   models.DeviceStatusType `json:"status"`
	}
	DeviceListResponse struct {
		Devices []DeviceListItem `json:"devices"`
	}
)

func (s *server) deviceList(userID models.ID) (*DeviceListResponse, error) {
	var res *DeviceListResponse

	err := db.Tx(func(tx *gorm.DB) error {
		var devices []models.Device
		if err := models.NewDeviceQuerySet(tx).UserIDEq(userID).OrderDescByCreatedAt().All(&devices); err != nil {
			return err
		}
		res = &DeviceListResponse{}
		for _, d := range devices {
			item := DeviceListItem{
				DeviceID: d.ID,
				Name:     d.Name,
				OS:       d.OS,
				Version:  d.Version,
				Address:  d.Address,
				LastSeen: d.LastSeen,
			}
			if d.LastSeen.After(time.Now().Add(-models.AssumeOnlineDuration)) {
				item.Status = models.DeviceStatusTypeOnline
			} else {
				item.Status = models.DeviceStatusTypeOffline
			}
			res.Devices = append(res.Devices, item)
		}
		return nil
	})

	return res, err
}

// UserDeviceList returns the device list associated to the user
func (s *server) UserDeviceList(r *http.Request) (*DeviceListResponse, error) {
	vars := Vars(mux.Vars(r))
	memberID := vars.ModelID("user_id")
	if memberID == 0 {
		return nil, errcode.ErrIllegalRequest
	}

	return s.deviceList(memberID)
}

// DeviceList returns the device list associated to the user
func (s *server) DeviceList(ctx context.Context) (*DeviceListResponse, error) {
	userID := jwt.UserIDFromContext(ctx)
	return s.deviceList(userID)
}

type (
	DeviceDeleteRequest struct {
		DeviceID models.ID `json:"device_id"`
	}

	DeviceUpdateRequest struct {
		Name string `json:"name"`
	}

	DeviceOperationResponse struct {
		Success bool `json:"success"`
	}
)

//DeviceUpdate update user device
func (s *server) DeviceUpdate(ctx context.Context, r *http.Request, req *DeviceUpdateRequest) (*DeviceOperationResponse, error) {
	vars := Vars(mux.Vars(r))
	deviceID := vars.ModelID("device_id")
	if deviceID == 0 {
		return nil, errcode.ErrIllegalRequest
	}
	userID := jwt.UserIDFromContext(ctx)
	err := db.Tx(func(tx *gorm.DB) error {
		return models.NewDeviceQuerySet(tx).
			UserIDEq(userID).
			IDEq(deviceID).
			GetUpdater().SetName(req.Name).Update()
	})
	if err != nil {
		return nil, err
	}

	res := &DeviceOperationResponse{
		Success: true,
	}

	return res, nil
}
