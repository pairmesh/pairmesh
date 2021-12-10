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
	"github.com/pairmesh/pairmesh/cmd/pairportal/db"
	"github.com/pairmesh/pairmesh/cmd/pairportal/db/models"
	"github.com/pairmesh/pairmesh/errcode"
	"github.com/pairmesh/pairmesh/internal/jwt"
	"github.com/pingcap/fn"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type (
	DeviceListItem struct {
		DeviceID   models.ID               `json:"device_id"`
		Name       string                  `json:"name"`
		OS         string                  `json:"os"`
		Version    string                  `json:"version"`
		Address    string                  `json:"address"`
		Status     models.DeviceStatusType `json:"status"`
		NetworkIDs []models.ID             `json:"network_ids"`
	}
	DeviceListResponse struct {
		Devices []DeviceListItem `json:"devices"`
	}
)

// DeviceList returns the device list associated to the user
func (s *server) DeviceList(ctx context.Context) (*DeviceListResponse, error) {
	userID := jwt.UserIDFromContext(ctx)
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
				Status:   models.DeviceStatusTypeOffline,
			}
			if d.LastSeen.After(time.Now().Add(-models.AssumeOnlineDuration)) {
				item.Status = models.DeviceStatusTypeOnline
			}
			var networkIDs []models.ID
			if err := tx.Raw(`select network_id from network_devices where user_id = ? and device_id = ?`, userID, d.ID).Scan(&networkIDs).Error; err == nil {
				item.NetworkIDs = networkIDs
			}
			res.Devices = append(res.Devices, item)
		}
		return nil
	})

	return res, err
}

type (
	AddDeviceToNetworksRequest struct {
		NetworkIDs []models.ID `json:"network_ids"`
	}

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

//AddDeviceToNetworks add device to networks
func (s *server) AddDeviceToNetworks(ctx context.Context, r *http.Request, req *AddDeviceToNetworksRequest) (*DeviceOperationResponse, error) {
	vars := Vars(mux.Vars(r))
	deviceID := vars.ModelID("device_id")
	if deviceID == 0 {
		return nil, errcode.ErrIllegalRequest
	}
	var res *DeviceOperationResponse
	err := db.Tx(func(tx *gorm.DB) error {
		userID := jwt.UserIDFromContext(ctx)
		for _, networkID := range req.NetworkIDs {
			var networkUser models.NetworkUser
			if err := models.NewNetworkUserQuerySet(tx).PreloadNetwork().NetworkIDEq(networkID).UserIDEq(userID).One(&networkUser); err != nil {
				return err
			}
			if networkUser.Network.Status == models.NetworkStatusTypeSuspend {
				return errors.New("can not add device to suspend network")
			}
			if networkUser.Role != models.RoleTypeOwner && networkUser.Role != models.RoleTypeAdmin {
				return errors.New("only network owner and admin add network device")
			}
			var device models.Device
			if err := models.NewDeviceQuerySet(tx).IDEq(deviceID).One(&device); err != nil {
				return err
			}
			if device.UserID != userID {
				return errors.New("only can add own device")
			}

			deviceInfo := &networkDeviceInfo{
				networkID: networkID,
				userID:    userID,
				deviceID:  deviceID,
				expiredAt: networkUser.ExpiredAt,
			}

			if err := s.createNetworkDevice(tx, deviceInfo); err != nil {
				return err
			}
		}

		res = &DeviceOperationResponse{
			Success: true,
		}

		return nil
	})

	return res, err
}

//DeviceDelete delete user device
func (s *server) DeviceDelete(ctx context.Context, r *http.Request) (*DeviceOperationResponse, error) {
	vars := Vars(mux.Vars(r))
	deviceID := vars.ModelID("device_id")
	if deviceID == 0 {
		return nil, errcode.ErrIllegalRequest
	}
	userID := jwt.UserIDFromContext(ctx)
	err := db.Tx(func(tx *gorm.DB) error {
		if err := models.NewNetworkDeviceQuerySet(tx).UserIDEq(userID).DeviceIDEq(deviceID).Delete(); err != nil {
			return err
		}

		if err := models.NewDeviceOnlineRecordQuerySet(tx).UserIDEq(userID).DeviceIDEq(deviceID).Delete(); err != nil {
			return err
		}

		return models.NewDeviceQuerySet(tx).
			UserIDEq(userID).
			IDEq(deviceID).
			Delete()
	})
	if err != nil {
		return nil, err
	}

	res := &DeviceOperationResponse{
		Success: true,
	}

	return res, nil
}

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

type (
	OnlineRecord struct {
		DeviceID models.ID `json:"device_id"`
		Name     string    `json:"name"`
		OS       string    `json:"os"`
		Version  string    `json:"version"`
		Address  string    `json:"address"`
		Time     int64     `json:"time"`
		Location string    `json:"location"`
	}

	DeviceOnlineRecordResponse struct {
		Records []OnlineRecord `json:"records"`
	}
)

// DeviceOnlineRecord returns the device online operation record
func (s *server) DeviceOnlineRecord(ctx context.Context, form *fn.Form) (*DeviceOnlineRecordResponse, error) {
	deviceId := form.Uint64("deviceId")
	res := &DeviceOnlineRecordResponse{}
	err := db.Tx(func(tx *gorm.DB) error {
		userID := jwt.UserIDFromContext(ctx)

		limit := 50
		var records []models.DeviceOnlineRecord
		query := models.NewDeviceOnlineRecordQuerySet(tx).PreloadDevice().UserIDEq(userID)
		if deviceId != 0 {
			limit = 10
			query = query.DeviceIDEq(models.ID(deviceId))
		}
		err := query.Limit(limit).OrderDescByCreatedAt().All(&records)
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		for _, record := range records {
			res.Records = append(res.Records, OnlineRecord{
				DeviceID: record.Device.ID,
				Name:     record.Device.Name,
				OS:       record.Device.OS,
				Version:  record.Device.Version,
				Address:  record.Device.Address,
				Time:     record.CreatedAt.UnixNano() / 1e6,
				Location: record.Location,
			})
		}
		return nil
	})
	return res, err
}

type (
	DeviceTraffic struct {
		DeviceID models.ID `json:"device_id"`
		Name     string    `json:"name"`
		Address  string    `json:"address"`
		RxBytes  uint64    `json:"rx_bytes"`
		TxBytes  uint64    `json:"tx_bytes"`
		Date     string    `json:"date"`
	}

	DeviceTrafficResponse struct {
		Traffics []DeviceTraffic `json:"traffics"`
	}
)
