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

	"github.com/pairmesh/pairmesh/cmd/pairportal/db"
	"github.com/pairmesh/pairmesh/cmd/pairportal/db/models"
	"github.com/pairmesh/pairmesh/errcode"
	"github.com/pairmesh/pairmesh/internal/jwt"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type (
	NetworkRequest struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	NetworkItem struct {
		NetworkID         models.ID                `json:"network_id"`
		Name              string                   `json:"name"`
		Creator           string                   `json:"creator"`
		CreatorID         models.ID                `json:"creator_id"`
		Description       string                   `json:"description"`
		Status            models.NetworkStatusType `json:"status"`
		CreatedAt         int64                    `json:"created_at"`
		MemberCount       int64                    `json:"member_count"`
		DeviceCount       int64                    `json:"device_count"`
		DeviceOnlineCount int64                    `json:"device_online_count"`
		Role              models.RoleType          `json:"role"`
	}

	NetworkResponse struct {
		Network NetworkItem `json:"network"`
	}
)

//CreateNetwork create network
func (s *server) CreateNetwork(ctx context.Context, req *NetworkRequest) (*NetworkResponse, error) {
	var res *NetworkResponse
	err := db.Tx(func(tx *gorm.DB) error {
		userID := jwt.UserIDFromContext(ctx)

		network := &models.Network{
			Name:        req.Name,
			Description: req.Description,
			CreatedByID: userID,
		}
		if err := db.Create(network); err != nil {
			return err
		}

		userInfo := &networkUserInfo{
			networkID: network.ID,
			userID:    userID,
			role:      models.RoleTypeOwner,
		}
		if err := s.createNetworkUser(tx, userInfo); err != nil {
			return err
		}
		var creatorName string
		if err := tx.Raw(`select name from users where id = ?`, userID).Scan(&creatorName).Error; err != nil {
			return err
		}
		item := NetworkItem{
			NetworkID:   network.ID,
			Name:        network.Name,
			Creator:     creatorName,
			CreatorID:   network.CreatedByID,
			Description: network.Description,
			Status:      network.Status,
			CreatedAt:   network.CreatedAt.UnixNano() / 1e6,
			MemberCount: 1,
			Role:        models.RoleTypeOwner,
		}

		res = &NetworkResponse{
			Network: item,
		}

		return nil
	})

	return res, err
}

//UpdateNetwork update the network
func (s *server) UpdateNetwork(ctx context.Context, r *http.Request, req *NetworkRequest) (*NetworkResponse, error) {
	vars := Vars(mux.Vars(r))
	networkID := vars.ModelID("network_id")
	if networkID == 0 {
		return nil, errcode.ErrIllegalRequest
	}
	var res *NetworkResponse
	err := db.Tx(func(tx *gorm.DB) error {
		if err := s.networkUserOperationCheck(ctx, networkID, models.RoleTypeAdmin); err != nil {
			return err
		}

		if err := models.NewNetworkQuerySet(tx).IDEq(networkID).GetUpdater().SetName(req.Name).SetDescription(req.Description).Update(); err != nil {
			return err
		}

		item := NetworkItem{
			NetworkID:   networkID,
			Name:        req.Name,
			Description: req.Description,
		}
		res = &NetworkResponse{
			Network: item,
		}
		return nil
	})
	return res, err
}

type (
	networkDigitalInfo struct {
		memberCount       int64
		deviceCount       int64
		deviceOnlineCount int64
	}

	NetworkInfoResponse struct {
		Networks []NetworkItem `json:"networks"`
	}
)

func (s *server) networkDigitalInfo(networkID models.ID) *networkDigitalInfo {
	res := &networkDigitalInfo{}
	_ = db.Tx(func(tx *gorm.DB) error {
		var userIDs []models.ID
		err := tx.Raw(`select user_id from network_users where network_id = ? and (expired_at is null or expired_at > ?)`, networkID, time.Now()).Scan(&userIDs).Error
		if err != nil || len(userIDs) == 0 {
			return nil
		}
		res.memberCount = int64(len(userIDs))

		var deviceIDs []models.ID
		err = tx.Raw(`select device_id from network_devices where network_id = ? and user_id in (?)`, networkID, userIDs).Scan(&deviceIDs).Error
		if err == nil && len(deviceIDs) > 0 {
			deviceSet := models.NewDeviceQuerySet(tx).IDIn(deviceIDs...)
			deviceCount, err := deviceSet.Count()
			if err == nil {
				res.deviceCount = deviceCount
			}
			deviceOnlineCount, err := deviceSet.LastSeenGte(time.Now().Add(-models.AssumeOnlineDuration)).Count()
			if err == nil {
				res.deviceOnlineCount = deviceOnlineCount
			}
		}
		return nil
	})
	return res
}

// Network get network by network id
func (s *server) Network(ctx context.Context, r *http.Request) (*NetworkItem, error) {
	vars := Vars(mux.Vars(r))
	networkId := vars.ModelID("network_id")
	if networkId == 0 {
		return nil, errcode.ErrIllegalRequest
	}
	var res *NetworkItem
	err := db.Tx(func(tx *gorm.DB) error {
		userID := jwt.UserIDFromContext(ctx)

		var network models.Network
		if err := models.NewNetworkQuerySet(tx).PreloadCreatedBy().IDEq(networkId).One(&network); err != nil {
			return err
		}
		var role string
		if err := tx.Raw(`select role from network_users where network_id = ? and user_id = ?`, network.ID, userID).Scan(&role).Error; err != nil {
			return err
		}

		digitalInfo := s.networkDigitalInfo(network.ID)
		res = &NetworkItem{
			NetworkID:         network.ID,
			Name:              network.Name,
			Creator:           network.CreatedBy.Name,
			CreatorID:         network.CreatedByID,
			Description:       network.Description,
			Status:            network.Status,
			CreatedAt:         network.CreatedAt.UnixNano() / 1e6,
			MemberCount:       digitalInfo.memberCount,
			DeviceCount:       digitalInfo.deviceCount,
			DeviceOnlineCount: digitalInfo.deviceOnlineCount,
			Role:              models.RoleType(role),
		}

		return nil
	})
	return res, err
}

// NetworkList returns the networks which the user or the device have group in
func (s *server) NetworkList(ctx context.Context, form *fn.Form) (*NetworkInfoResponse, error) {
	deviceId := form.Get("deviceId")
	res := &NetworkInfoResponse{}
	err := db.Tx(func(tx *gorm.DB) error {
		userID := jwt.UserIDFromContext(ctx)

		var networkIDs []models.ID
		var err error
		if deviceId != "" {
			err = tx.Raw(`select network_id from network_devices where user_id = ? and device_id = ? AND (expired_at is null or expired_at > ?)`, userID, deviceId, time.Now()).Scan(&networkIDs).Error
		} else {
			err = tx.Raw(`select network_id from network_users where user_id = ? AND (expired_at is null or expired_at > ?)`, userID, time.Now()).Scan(&networkIDs).Error
		}
		if err != nil {
			return err
		}
		if len(networkIDs) == 0 {
			return nil
		}

		var networks []models.Network
		if err = models.NewNetworkQuerySet(tx).PreloadCreatedBy().IDIn(networkIDs...).OrderDescByCreatedAt().All(&networks); err != nil {
			return err
		}

		var role string
		for _, network := range networks {
			if err = tx.Raw(`select role from network_users where network_id = ? and user_id = ?`, network.ID, userID).Scan(&role).Error; err != nil {
				return err
			}

			digitalInfo := s.networkDigitalInfo(network.ID)
			item := NetworkItem{
				NetworkID:         network.ID,
				Name:              network.Name,
				Creator:           network.CreatedBy.Name,
				CreatorID:         network.CreatedByID,
				Description:       network.Description,
				Status:            network.Status,
				CreatedAt:         network.CreatedAt.UnixNano() / 1e6,
				MemberCount:       digitalInfo.memberCount,
				DeviceCount:       digitalInfo.deviceCount,
				DeviceOnlineCount: digitalInfo.deviceOnlineCount,
				Role:              models.RoleType(role),
			}

			res.Networks = append(res.Networks, item)
		}
		return nil
	})
	return res, err
}

// NetworkAddableForDevice returns the networks which the device can be add by network owner
func (s *server) NetworkAddableForDevice(ctx context.Context, r *http.Request) (*NetworkInfoResponse, error) {
	vars := Vars(mux.Vars(r))
	deviceID := vars.ModelID("device_id")
	if deviceID == 0 {
		return nil, errcode.ErrIllegalRequest
	}
	var res *NetworkInfoResponse
	err := db.Tx(func(tx *gorm.DB) error {
		userID := jwt.UserIDFromContext(ctx)
		var networkIDs []models.ID
		if err := tx.Raw(`select network_id from network_devices where user_id = ? and device_id = ?`, userID, deviceID).Scan(&networkIDs).Error; err != nil {
			return err
		}
		var networks []models.Network
		networkQuery := models.NewNetworkQuerySet(tx).PreloadCreatedBy().CreatedByIDEq(userID)
		if len(networkIDs) != 0 {
			networkQuery = networkQuery.IDNotIn(networkIDs...)
		}

		if err := networkQuery.StatusNe(models.NetworkStatusTypeSuspend).All(&networks); err != nil {
			return err
		}

		res = &NetworkInfoResponse{}
		for _, network := range networks {
			digitalInfo := s.networkDigitalInfo(network.ID)
			item := NetworkItem{
				NetworkID:         network.ID,
				Name:              network.Name,
				Creator:           network.CreatedBy.Name,
				CreatorID:         network.CreatedByID,
				Description:       network.Description,
				Status:            network.Status,
				CreatedAt:         network.CreatedAt.UnixNano() / 1e6,
				MemberCount:       digitalInfo.memberCount,
				DeviceCount:       digitalInfo.deviceCount,
				DeviceOnlineCount: digitalInfo.deviceOnlineCount,
				Role:              models.RoleTypeOwner,
			}

			res.Networks = append(res.Networks, item)
		}
		return nil
	})
	return res, err
}

type (
	NetworkMemberItem struct {
		NetworkID         models.ID                `json:"network_id"`
		UserID            models.ID                `json:"user_id"`
		Name              string                   `json:"name"`
		Avatar            string                   `json:"avatar"`
		Origin            string                   `json:"origin"`
		JoinTime          int64                    `json:"join_time"`
		Role              models.RoleType          `json:"role"`
		Status            models.NetworkStatusType `json:"status"`
		ExpiredAt         int64                    `json:"expire_at"`
		DeviceCount       int64                    `json:"device_count"`
		DeviceOnlineCount int64                    `json:"device_online_count"`
	}

	NetworkMemberResponse struct {
		Members []NetworkMemberItem `json:"members"`
	}
)

// NetworkMembers returns the members of the network
func (s *server) NetworkMembers(r *http.Request) (*NetworkMemberResponse, error) {
	vars := Vars(mux.Vars(r))
	networkID := vars.ModelID("network_id")
	if networkID == 0 {
		return nil, errcode.ErrIllegalRequest
	}
	var res *NetworkMemberResponse
	err := db.Tx(func(tx *gorm.DB) error {
		var networkUsers []models.NetworkUser
		if err := models.NewNetworkUserQuerySet(tx).PreloadUser().NetworkIDEq(networkID).All(&networkUsers); err != nil {
			return err
		}

		res = &NetworkMemberResponse{}
		now := time.Now()
		for _, user := range networkUsers {
			if user.ExpiredAt != nil && user.ExpiredAt.Before(now) {
				continue
			}
			item := NetworkMemberItem{
				NetworkID: networkID,
				UserID:    user.UserID,
				Name:      user.User.Name,
				Avatar:    user.User.Avatar,
				Origin:    user.User.Origin,
				JoinTime:  user.CreatedAt.UnixNano() / 1e6,
				Role:      user.Role,
				Status:    user.Status,
			}
			if user.ExpiredAt != nil {
				item.ExpiredAt = user.ExpiredAt.UnixNano() / 1e6
			}
			var deviceIDs []models.ID
			err := tx.Raw(`select device_id from network_devices where network_id = ? and user_id = ?`, networkID, user.UserID).Scan(&deviceIDs).Error
			if err != nil {
				continue
			}
			item.DeviceCount = int64(len(deviceIDs))
			if item.DeviceCount > 0 {
				deviceOnlineCount, err := models.NewDeviceQuerySet(tx).IDIn(deviceIDs...).LastSeenGte(time.Now().Add(-models.AssumeOnlineDuration)).Count()
				if err != nil {
					continue
				}
				item.DeviceOnlineCount = deviceOnlineCount
			}
			res.Members = append(res.Members, item)
		}
		return nil
	})
	return res, err
}

type (
	NetworkDeviceItem struct {
		NetworkID     models.ID                `json:"network_id"`
		DeviceID      models.ID                `json:"device_id"`
		Name          string                   `json:"name"`
		OS            string                   `json:"os"`
		Version       string                   `json:"version"`
		Address       string                   `json:"address"`
		DeviceStatus  models.DeviceStatusType  `json:"device_status"`
		NetworkStatus models.NetworkStatusType `json:"network_status"`
	}

	NetworkDevicesResponse struct {
		Devices []NetworkDeviceItem `json:"devices"`
	}
)

// NetworkDevices returns all the devices of the network
func (s *server) NetworkDevices(r *http.Request) (*NetworkDevicesResponse, error) {
	vars := Vars(mux.Vars(r))
	networkID := vars.ModelID("network_id")
	if networkID == 0 {
		return nil, errcode.ErrIllegalRequest
	}

	var networkDevices []models.NetworkDevice
	err := db.Tx(func(tx *gorm.DB) error {
		return models.NewNetworkDeviceQuerySet(tx).PreloadDevice().NetworkIDEq(networkID).All(&networkDevices)
	})
	if err != nil {
		return nil, err
	}

	res := &NetworkDevicesResponse{}
	for _, networkDevice := range networkDevices {
		item := NetworkDeviceItem{
			NetworkID:     networkDevice.NetworkID,
			DeviceID:      networkDevice.DeviceID,
			Name:          networkDevice.Device.Name,
			OS:            networkDevice.Device.OS,
			Version:       networkDevice.Device.Version,
			Address:       networkDevice.Device.Address,
			DeviceStatus:  models.DeviceStatusTypeOffline,
			NetworkStatus: networkDevice.Status,
		}
		if networkDevice.Device.LastSeen.After(time.Now().Add(-models.AssumeOnlineDuration)) {
			item.DeviceStatus = models.DeviceStatusTypeOnline
		}

		res.Devices = append(res.Devices, item)
	}

	return res, err
}

// NetworkMemberDevices returns the devices of the network member
func (s *server) NetworkMemberDevices(r *http.Request) (*NetworkDevicesResponse, error) {
	vars := Vars(mux.Vars(r))
	networkID := vars.ModelID("network_id")
	memberID := vars.ModelID("member_id")
	if networkID == 0 || memberID == 0 {
		return nil, errcode.ErrIllegalRequest
	}

	var networkDevices []models.NetworkDevice
	err := db.Tx(func(tx *gorm.DB) error {
		return models.NewNetworkDeviceQuerySet(tx).PreloadDevice().NetworkIDEq(networkID).UserIDEq(memberID).OrderDescByCreatedAt().All(&networkDevices)
	})
	if err != nil {
		return nil, err
	}

	res := &NetworkDevicesResponse{}
	for _, networkDevice := range networkDevices {
		item := NetworkDeviceItem{
			NetworkID:     networkDevice.NetworkID,
			DeviceID:      networkDevice.DeviceID,
			Name:          networkDevice.Device.Name,
			OS:            networkDevice.Device.OS,
			Version:       networkDevice.Device.Version,
			Address:       networkDevice.Device.Address,
			DeviceStatus:  models.DeviceStatusTypeOffline,
			NetworkStatus: networkDevice.Status,
		}
		if networkDevice.Device.LastSeen.After(time.Now().Add(-models.AssumeOnlineDuration)) {
			item.DeviceStatus = models.DeviceStatusTypeOnline
		}

		res.Devices = append(res.Devices, item)
	}

	return res, err
}

type (
	ChangeNetworkStatusRequest struct {
		Status string `json:"status"`
	}

	ChangeNetworkDurationRequest struct {
		DurationType string `json:"duration_type"`
		Duration     int64  `json:"duration"`
	}

	NetworkOperationResponse struct {
		Success bool `json:"success"`
	}
)

func (s *server) networkUserOperationCheck(ctx context.Context, networkID models.ID, allowRole models.RoleType) error {
	return db.Tx(func(tx *gorm.DB) error {
		userID := jwt.UserIDFromContext(ctx)
		var networkUser models.NetworkUser
		if err := models.NewNetworkUserQuerySet(tx).NetworkIDEq(networkID).UserIDEq(userID).One(&networkUser); err != nil {
			return err
		}
		if allowRole == models.RoleTypeOwner && networkUser.Role != models.RoleTypeOwner {
			return errcode.ErrIllegalOperation
		}
		if allowRole == models.RoleTypeAdmin && (networkUser.Role != models.RoleTypeOwner && networkUser.Role != models.RoleTypeAdmin) {
			return errcode.ErrIllegalOperation
		}
		return nil
	})
}

//ChangeNetworkStatus change the network status with active or inactive
func (s *server) ChangeNetworkStatus(ctx context.Context, r *http.Request, req *ChangeNetworkStatusRequest) (*NetworkOperationResponse, error) {
	vars := Vars(mux.Vars(r))
	networkID := vars.ModelID("network_id")
	if networkID == 0 {
		return nil, errcode.ErrIllegalRequest
	}
	if req.Status != "active" && req.Status != "inactive" {
		return nil, errcode.ErrIllegalRequest
	}
	var res *NetworkOperationResponse
	err := db.Tx(func(tx *gorm.DB) error {
		err := s.networkUserOperationCheck(ctx, networkID, models.RoleTypeOwner)
		if err != nil {
			return err
		}
		var status = models.NetworkStatusTypeActive
		if req.Status == "inactive" {
			status = models.NetworkStatusTypeInactive
		}

		err = models.NewNetworkQuerySet(tx).
			IDEq(networkID).
			GetUpdater().
			SetStatus(status).
			Update()
		if err != nil {
			return err
		}

		res = &NetworkOperationResponse{
			Success: true,
		}
		return nil
	})
	return res, err
}

//ChangeNetworkMemberStatus change the network member status with active or inactive
func (s *server) ChangeNetworkMemberStatus(ctx context.Context, r *http.Request, req *ChangeNetworkStatusRequest) (*NetworkOperationResponse, error) {
	vars := Vars(mux.Vars(r))
	networkID := vars.ModelID("network_id")
	memberID := vars.ModelID("member_id")
	if networkID == 0 || memberID == 0 {
		return nil, errcode.ErrIllegalRequest
	}
	if req.Status != "active" && req.Status != "inactive" {
		return nil, errcode.ErrIllegalRequest
	}
	var res *NetworkOperationResponse
	err := db.Tx(func(tx *gorm.DB) error {
		err := s.networkUserOperationCheck(ctx, networkID, models.RoleTypeOwner)
		if err != nil {
			return err
		}
		var status = models.NetworkStatusTypeActive
		if req.Status == "inactive" {
			status = models.NetworkStatusTypeInactive
		}

		err = models.NewNetworkUserQuerySet(tx).
			NetworkIDEq(networkID).
			UserIDEq(memberID).
			GetUpdater().
			SetStatus(status).
			Update()
		if err != nil {
			return err
		}

		res = &NetworkOperationResponse{
			Success: true,
		}
		return nil
	})
	return res, err
}

//ChangeNetworkMemberDuration change the network member duration with permanent or temporary
func (s *server) ChangeNetworkMemberDuration(ctx context.Context, r *http.Request, req *ChangeNetworkDurationRequest) (*NetworkOperationResponse, error) {
	vars := Vars(mux.Vars(r))
	networkID := vars.ModelID("network_id")
	memberID := vars.ModelID("member_id")
	if networkID == 0 || memberID == 0 {
		return nil, errcode.ErrIllegalRequest
	}
	if req.DurationType != "permanent" && req.DurationType != "temporary" {
		return nil, errcode.ErrIllegalRequest
	}
	var res *NetworkOperationResponse
	err := db.Tx(func(tx *gorm.DB) error {
		err := s.networkUserOperationCheck(ctx, networkID, models.RoleTypeOwner)
		if err != nil {
			return err
		}
		memberUpdater := models.NewNetworkUserQuerySet(tx).
			NetworkIDEq(networkID).
			UserIDEq(memberID).
			GetUpdater()

		memberDeviceUpdater := models.NewNetworkDeviceQuerySet(tx).
			NetworkIDEq(networkID).
			UserIDEq(memberID).
			GetUpdater()

		if req.DurationType == "temporary" {
			expire := time.Unix(req.Duration/1000, 0)
			memberUpdater.SetExpiredAt(&expire)
			memberDeviceUpdater.SetExpiredAt(&expire)
		} else {
			memberUpdater.SetExpiredAt(nil)
			memberDeviceUpdater.SetExpiredAt(nil)
		}

		err = memberUpdater.Update()
		if err != nil {
			return err
		}

		err = memberDeviceUpdater.Update()
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		res = &NetworkOperationResponse{
			Success: true,
		}
		return nil
	})
	return res, err
}

//ChangeNetworkDeviceStatus change the network device status with active or inactive
func (s *server) ChangeNetworkDeviceStatus(ctx context.Context, r *http.Request, req *ChangeNetworkStatusRequest) (*NetworkOperationResponse, error) {
	vars := Vars(mux.Vars(r))
	networkID := vars.ModelID("network_id")
	deviceID := vars.ModelID("device_id")
	if networkID == 0 || deviceID == 0 {
		return nil, errcode.ErrIllegalRequest
	}
	if req.Status != "active" && req.Status != "inactive" {
		return nil, errcode.ErrIllegalRequest
	}
	var res *NetworkOperationResponse
	err := db.Tx(func(tx *gorm.DB) error {
		err := s.networkUserOperationCheck(ctx, networkID, models.RoleTypeOwner)
		if err != nil {
			return err
		}
		var status = models.NetworkStatusTypeActive
		if req.Status == "inactive" {
			status = models.NetworkStatusTypeInactive
		}

		err = models.NewNetworkDeviceQuerySet(tx).
			NetworkIDEq(networkID).
			DeviceIDEq(deviceID).
			GetUpdater().
			SetStatus(status).
			Update()
		if err != nil {
			return err
		}

		res = &NetworkOperationResponse{
			Success: true,
		}
		return nil
	})
	return res, err
}

//DeleteNetwork delete the network
func (s *server) DeleteNetwork(ctx context.Context, r *http.Request) (*NetworkOperationResponse, error) {
	vars := Vars(mux.Vars(r))
	networkID := vars.ModelID("network_id")
	if networkID == 0 {
		return nil, errcode.ErrIllegalRequest
	}
	var res *NetworkOperationResponse
	err := db.Tx(func(tx *gorm.DB) error {
		err := s.networkUserOperationCheck(ctx, networkID, models.RoleTypeOwner)
		if err != nil {
			return err
		}

		err = models.NewNetworkDeviceQuerySet(tx).NetworkIDEq(networkID).Delete()
		if err != nil {
			return err
		}

		err = models.NewNetworkUserQuerySet(tx).NetworkIDEq(networkID).Delete()
		if err != nil {
			return err
		}

		err = models.NewInvitationQuerySet(tx).NetworkIDEq(networkID).Delete()
		if err != nil {
			return err
		}

		err = models.NewNetworkQuerySet(tx).
			IDEq(networkID).
			Delete()
		if err != nil {
			return err
		}

		res = &NetworkOperationResponse{
			Success: true,
		}
		return nil
	})
	return res, err
}

type (
	ChangeMemberPermissionRequest struct {
		Role string `json:"role"`
	}

	ChangeMemberPermissionResponse struct {
		Role models.RoleType `json:"role"`
	}
)

//ChangeNetworkMemberRole change network admin member grant and revoke
func (s *server) ChangeNetworkMemberRole(ctx context.Context, r *http.Request, req *ChangeMemberPermissionRequest) (*ChangeMemberPermissionResponse, error) {
	vars := Vars(mux.Vars(r))
	memberID := vars.ModelID("member_id")
	networkID := vars.ModelID("network_id")
	if memberID == 0 || networkID == 0 {
		return nil, errcode.ErrIllegalRequest
	}

	if req.Role != "admin" && req.Role != "member" {
		return nil, errcode.ErrIllegalRequest
	}

	// Only owner can change the member permission
	var res *ChangeMemberPermissionResponse
	err := db.Tx(func(tx *gorm.DB) error {
		err := s.networkUserOperationCheck(ctx, networkID, models.RoleTypeOwner)
		if err != nil {
			return err
		}

		var role = models.RoleTypeMember
		if req.Role == "admin" {
			role = models.RoleTypeAdmin
		}
		err = models.NewNetworkUserQuerySet(tx).
			NetworkIDEq(networkID).
			UserIDEq(memberID).
			GetUpdater().
			SetRole(role).
			Update()
		if err != nil {
			return err
		}

		res = &ChangeMemberPermissionResponse{
			Role: role,
		}
		return nil
	})

	return res, err
}

type (
	DeleteNetworkUserResponse struct {
		UserID models.ID `json:"user_id"`
	}
	DeleteNetworkDeviceResponse struct {
		DeviceID models.ID `json:"device_id"`
	}
)

//DeleteNetworkUser delete user in network
func (s *server) DeleteNetworkUser(ctx context.Context, r *http.Request) (*DeleteNetworkUserResponse, error) {
	vars := Vars(mux.Vars(r))
	memberID := vars.ModelID("member_id")
	networkID := vars.ModelID("network_id")
	if memberID == 0 || networkID == 0 {
		return nil, errcode.ErrIllegalRequest
	}

	var res *DeleteNetworkUserResponse
	err := db.Tx(func(tx *gorm.DB) error {
		userID := jwt.UserIDFromContext(ctx)
		var role string
		if err := tx.Raw(`select role from network_users where network_id = ? and user_id = ?`, networkID, userID).Scan(&role).Error; err != nil {
			return err
		}
		requestUserRole := models.RoleType(role) // Request user role
		if userID == memberID {                  //delete self
			if requestUserRole == models.RoleTypeOwner {
				return errors.New("owner can not delete self from the network")
			}
		} else {
			if requestUserRole == models.RoleTypeMember {
				return errors.New("member can only delete self")
			}
			if err := tx.Raw(`select role from network_users where network_id = ? and user_id = ?`, networkID, memberID).Scan(&role).Error; err != nil {
				return err
			}
			targetUserRole := models.RoleType(role) // Target user role
			if targetUserRole == models.RoleTypeOwner {
				return errors.New("owner can not be delete")
			}
			if targetUserRole == models.RoleTypeAdmin && requestUserRole != models.RoleTypeOwner {
				return errors.New("admin can only be delete by owner")
			}
		}

		err := models.NewNetworkDeviceQuerySet(tx).
			NetworkIDEq(networkID).
			UserIDEq(memberID).
			Delete()
		if err != nil {
			return err
		}

		err = models.NewNetworkUserQuerySet(tx).
			NetworkIDEq(networkID).
			UserIDEq(memberID).
			Delete()
		if err != nil {
			return err
		}

		res = &DeleteNetworkUserResponse{
			UserID: memberID,
		}

		return nil
	})

	return res, err
}

//DeleteNetworkDevice delete device in network
func (s *server) DeleteNetworkDevice(ctx context.Context, r *http.Request) (*DeleteNetworkDeviceResponse, error) {
	vars := Vars(mux.Vars(r))
	deviceID := vars.ModelID("device_id")
	networkID := vars.ModelID("network_id")
	if deviceID == 0 || networkID == 0 {
		return nil, errcode.ErrIllegalRequest
	}

	var res *DeleteNetworkDeviceResponse
	err := db.Tx(func(tx *gorm.DB) error {
		userID := jwt.UserIDFromContext(ctx)
		var targetDevice models.NetworkDevice
		err := models.NewNetworkDeviceQuerySet(tx).
			PreloadDevice().
			NetworkIDEq(networkID).
			DeviceIDEq(deviceID). // Target device
			One(&targetDevice)
		if err != nil {
			return err
		}
		if userID == targetDevice.Device.UserID {
			return models.NewNetworkDeviceQuerySet(tx).
				NetworkIDEq(networkID).
				DeviceIDEq(deviceID).
				Delete()
		}

		var role string
		if err = tx.Raw(`select role from network_users where network_id = ? and user_id = ?`, networkID, userID).Scan(&role).Error; err != nil {
			return err
		}
		requestUserRole := models.RoleType(role) // Request user role
		if requestUserRole != models.RoleTypeMember {
			return errors.New("member can not delete other's device")
		}

		if err = tx.Raw(`select role from network_users where network_id = ? and user_id = ?`, networkID, targetDevice.Device.UserID).Scan(&role).Error; err != nil {
			return err
		}
		targetUserRole := models.RoleType(role) // Target user role
		if targetUserRole == models.RoleTypeOwner {
			return errors.New("owner device can not be delete by others")
		}

		if targetUserRole == models.RoleTypeAdmin && requestUserRole != models.RoleTypeOwner {
			return errors.New("admin device can only be delete by owner")
		}

		err = models.NewNetworkDeviceQuerySet(tx).
			NetworkIDEq(networkID).
			DeviceIDEq(deviceID).
			Delete()
		if err != nil {
			return err
		}

		return nil
	})
	if err == nil {
		res = &DeleteNetworkDeviceResponse{
			DeviceID: deviceID,
		}
	}

	return res, err
}

type (
	InviteMemberRequest struct {
		UserID        models.ID       `json:"user_id"`
		DeviceLimit   uint            `json:"device_limit"`
		Role          models.RoleType `json:"role"`
		UserExpiredAt int64           `json:"user_expired_at"`
	}

	InviteMemberResponse struct {
		InvitationID models.ID `json:"invitation_id"`
	}
)

func (s *server) InviteMember(ctx context.Context, r *http.Request, req *InviteMemberRequest) (*InviteMemberResponse, error) {
	vars := Vars(mux.Vars(r))
	networkID := vars.ModelID("network_id")
	if networkID == 0 {
		return nil, errcode.ErrIllegalRequest
	}

	if req.UserID == 0 || req.DeviceLimit <= 0 {
		return nil, errcode.ErrIllegalRequest
	}

	var res *InviteMemberResponse
	err := db.Tx(func(tx *gorm.DB) error {
		userID := jwt.UserIDFromContext(ctx)

		var invitationNetworkUser models.NetworkUser //invite user
		if err := models.NewNetworkUserQuerySet(tx).PreloadNetwork().NetworkIDEq(networkID).UserIDEq(userID).One(&invitationNetworkUser); err != nil {
			return err
		}
		if invitationNetworkUser.Role == models.RoleTypeMember {
			return errors.New("network member can not invite user")
		}
		if req.Role == models.RoleTypeAdmin && invitationNetworkUser.Role != models.RoleTypeOwner {
			return errors.New("only network owner can invite admin")
		}
		if invitationNetworkUser.Network.Status == models.NetworkStatusTypeSuspend {
			return errors.New("can not invite member to suspend network")
		}

		// Check duplication invitation
		updateInvitation := false
		var unProcessInvitation models.Invitation
		err := models.NewInvitationQuerySet(tx).NetworkIDEq(networkID).UserIDEq(req.UserID).ExpiredAtGt(time.Now()).StatusEq(models.InvitationStatusTypeUnprocessed).One(&unProcessInvitation)
		switch err {
		case nil:
			updateInvitation = true
		case gorm.ErrRecordNotFound:
			updateInvitation = false
		default:
			return err
		}

		if updateInvitation {
			updater := models.NewInvitationQuerySet(tx).NetworkIDEq(networkID).UserIDEq(req.UserID).GetUpdater()
			updater.SetDeviceLimit(req.DeviceLimit)
			updater.SetRole(req.Role)
			if req.UserExpiredAt != 0 {
				expire := time.Unix(req.UserExpiredAt/1000, 0)
				updater.SetUserExpiredAt(&expire)
			} else {
				updater.SetUserExpiredAt(nil)
			}

			err = updater.SetExpiredAt(time.Now().Add(7 * 24 * time.Hour)).SetStatus(models.InvitationStatusTypeUnprocessed).Update()
			if err != nil {
				return err
			}
			res = &InviteMemberResponse{
				InvitationID: unProcessInvitation.ID,
			}
			return nil
		}

		invitation := &models.Invitation{
			NetworkID:   networkID,
			InvitedByID: userID,
			UserID:      req.UserID,
			DeviceLimit: req.DeviceLimit,
			ExpiredAt:   time.Now().Add(7 * 24 * time.Hour),
		}
		if req.Role == models.RoleTypeAdmin {
			invitation.Role = req.Role
		}
		if req.UserExpiredAt != 0 {
			expire := time.Unix(req.UserExpiredAt/1000, 0)
			invitation.UserExpiredAt = &expire
		}
		tx.Create(invitation)
		if tx.Error != nil {
			return tx.Error
		}

		res = &InviteMemberResponse{
			InvitationID: invitation.ID,
		}

		return nil
	})

	return res, err
}

type (
	EnableNetworksRequest struct {
		NetworkIDs []models.ID `json:"network_ids"`
	}
)

//EnableNetworks enable networks witch is suspend
func (s *server) EnableNetworks(ctx context.Context, req *EnableNetworksRequest) (*NetworkOperationResponse, error) {
	var res *NetworkOperationResponse
	err := db.Tx(func(tx *gorm.DB) error {
		userID := jwt.UserIDFromContext(ctx)
		for _, networkID := range req.NetworkIDs {
			query := models.NewNetworkQuerySet(tx).IDEq(networkID)
			var network models.Network
			if err := query.One(&network); err != nil {
				return err
			}
			if network.CreatedByID != userID {
				return errors.New("only network owner can enable network")
			}
			if network.Status != models.NetworkStatusTypeSuspend {
				return errors.New("can not enable network witch is not suspend")
			}
			if err := query.GetUpdater().SetStatus(models.NetworkStatusTypeActive).Update(); err != nil {
				return err
			}
		}
		res = &NetworkOperationResponse{
			Success: true,
		}

		return nil
	})

	return res, err
}
