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

	"github.com/pairmesh/pairmesh/errcode"
	"github.com/pairmesh/pairmesh/pkg/jwt"
	"github.com/pairmesh/pairmesh/portal/db"
	"github.com/pairmesh/pairmesh/portal/db/models"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type (
	NetworkRequest struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	NetworkItem struct {
		NetworkID   models.ID       `json:"network_id"`
		Name        string          `json:"name"`
		Description string          `json:"description"`
		CreatedAt   int64           `json:"created_at"`
		MemberCount int64           `json:"member_count"`
		DeviceCount int64           `json:"device_count"`
		Role        models.RoleType `json:"role"`
	}

	NetworkResponse struct {
		Network NetworkItem `json:"network"`
	}
)

// CreateNetwork creates network
func (s *server) CreateNetwork(ctx context.Context, req *NetworkRequest) (*NetworkResponse, error) {
	var res *NetworkResponse
	err := db.Tx(func(tx *gorm.DB) error {
		userID := models.ID(jwt.UserIDFromContext(ctx))

		network := &models.Network{
			Name:        req.Name,
			Description: req.Description,
			CreatedByID: userID,
		}
		if err := db.Create(network); err != nil {
			return err
		}

		networkUser := &models.NetworkUser{
			NetworkID: network.ID,
			UserID:    userID,
			Role:      models.RoleTypeOwner,
		}
		if err := db.Create(networkUser); err != nil {
			return err
		}

		uc, err := models.NewDeviceQuerySet(tx).UserIDEq(userID).Count()
		if err != nil {
			return err
		}

		item := NetworkItem{
			NetworkID:   network.ID,
			Name:        network.Name,
			Description: network.Description,
			CreatedAt:   network.CreatedAt.Unix(),
			MemberCount: 1,
			DeviceCount: uc,
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

		err := models.NewNetworkQuerySet(tx).
			IDEq(networkID).
			GetUpdater().
			SetName(req.Name).
			SetDescription(req.Description).
			Update()
		if err != nil {
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
	NetworkInfoResponse struct {
		Networks []NetworkItem `json:"networks"`
	}
)

// NetworkList returns the networks which the user or the device have group in
func (s *server) NetworkList(ctx context.Context) (*NetworkInfoResponse, error) {
	userID := models.ID(jwt.UserIDFromContext(ctx))

	res := &NetworkInfoResponse{}
	err := db.Tx(func(tx *gorm.DB) error {
		var networkUsers []models.NetworkUser
		err := models.NewNetworkUserQuerySet(tx).PreloadNetwork().UserIDEq(userID).All(&networkUsers)
		if err != nil {
			return err
		}

		for _, networkUser := range networkUsers {
			uc, dc, err := models.NetworkStats(tx, networkUser.NetworkID)
			if err != nil {
				return err
			}

			item := NetworkItem{
				NetworkID:   networkUser.NetworkID,
				Name:        networkUser.Network.Name,
				Description: networkUser.Network.Description,
				CreatedAt:   networkUser.CreatedAt.UnixNano() / 1e6,
				MemberCount: uc,
				DeviceCount: dc,
				Role:        networkUser.Role,
			}

			res.Networks = append(res.Networks, item)
		}
		return nil
	})
	return res, err
}

type (
	NetworkMemberItem struct {
		NetworkID models.ID       `json:"network_id"`
		UserID    models.ID       `json:"user_id"`
		Name      string          `json:"name"`
		Email     string          `json:"email"`
		JoinTime  int64           `json:"join_time"`
		Role      models.RoleType `json:"role"`
	}

	NetworkMemberResponse struct {
		Members []NetworkMemberItem `json:"members"`
		Owner   bool                `json:"owner"`
		Admin   bool                `json:"admin"`
	}
)

// NetworkMembers returns the members of the network
func (s *server) NetworkMembers(ctx context.Context, r *http.Request) (*NetworkMemberResponse, error) {
	vars := Vars(mux.Vars(r))
	networkID := vars.ModelID("network_id")
	if networkID == 0 {
		return nil, errcode.ErrIllegalRequest
	}
	userID := models.ID(jwt.UserIDFromContext(ctx))

	var res *NetworkMemberResponse
	err := db.Tx(func(tx *gorm.DB) error {
		currentUser := models.NetworkUser{}
		if err := models.NewNetworkUserQuerySet(tx).NetworkIDEq(networkID).UserIDEq(userID).One(&currentUser); err != nil {
			return errcode.ErrIllegalOperation
		}

		var networkUsers []models.NetworkUser
		if err := models.NewNetworkUserQuerySet(tx).PreloadUser().NetworkIDEq(networkID).All(&networkUsers); err != nil {
			return err
		}

		res = &NetworkMemberResponse{
			Owner: currentUser.Role == models.RoleTypeOwner,
			Admin: currentUser.Role == models.RoleTypeAdmin || currentUser.Role == models.RoleTypeOwner,
		}
		for _, user := range networkUsers {
			item := NetworkMemberItem{
				NetworkID: networkID,
				UserID:    user.UserID,
				Name:      user.User.Name,
				Email:     user.User.Email,
				JoinTime:  user.CreatedAt.Unix(),
				Role:      user.Role,
			}
			res.Members = append(res.Members, item)
		}
		return nil
	})
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
		userID := models.ID(jwt.UserIDFromContext(ctx))
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

//DeleteNetwork delete the network
func (s *server) DeleteNetwork(ctx context.Context, r *http.Request) (*NetworkOperationResponse, error) {
	vars := Vars(mux.Vars(r))
	networkID := vars.ModelID("network_id")
	if networkID == 0 {
		return nil, errcode.ErrIllegalRequest
	}
	userID := models.ID(jwt.UserIDFromContext(ctx))

	var res *NetworkOperationResponse
	err := db.Tx(func(tx *gorm.DB) error {
		var network models.Network
		err := models.NewNetworkQuerySet(tx).IDEq(networkID).One(&network)
		if err != nil {
			return err
		}
		if network.CreatedByID != userID {
			return errcode.ErrIllegalOperation
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
	userID := vars.ModelID("user_id")
	networkID := vars.ModelID("network_id")
	if userID == 0 || networkID == 0 {
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
			UserIDEq(userID).
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
	userID := vars.ModelID("user_id")
	networkID := vars.ModelID("network_id")
	if userID == 0 || networkID == 0 {
		return nil, errcode.ErrIllegalRequest
	}

	var res *DeleteNetworkUserResponse
	err := db.Tx(func(tx *gorm.DB) error {
		userIDFromJwt := models.ID(jwt.UserIDFromContext(ctx))
		var role string
		if err := tx.Raw(`select role from network_users where network_id = ? and user_id = ?`, networkID, userID).Scan(&role).Error; err != nil {
			return err
		}
		requestUserRole := models.RoleType(role) // Request user role
		if userID == userIDFromJwt {             //delete self
			if requestUserRole == models.RoleTypeOwner {
				return errors.New("owner can not delete self from the network")
			}
		} else {
			if requestUserRole == models.RoleTypeMember {
				return errors.New("member can only delete self")
			}
			if err := tx.Raw(`select role from network_users where network_id = ? and user_id = ?`, networkID, userID).Scan(&role).Error; err != nil {
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

		err := models.NewNetworkUserQuerySet(tx).
			NetworkIDEq(networkID).
			UserIDEq(userID).
			Delete()
		if err != nil {
			return err
		}

		res = &DeleteNetworkUserResponse{
			UserID: userID,
		}

		return nil
	})

	return res, err
}

type (
	InviteMemberRequest struct {
		Email string          `json:"email"`
		Role  models.RoleType `json:"role"`
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

	if req.Email == "" {
		return nil, errcode.ErrIllegalRequest
	}

	userID := models.ID(jwt.UserIDFromContext(ctx))

	var res *InviteMemberResponse
	err := db.Tx(func(tx *gorm.DB) error {
		var invitationNetworkUser models.NetworkUser //invite user
		err := models.NewNetworkUserQuerySet(tx).
			PreloadNetwork().
			NetworkIDEq(networkID).
			UserIDEq(userID).
			One(&invitationNetworkUser)
		if err != nil {
			return err
		}
		if invitationNetworkUser.Role == models.RoleTypeMember {
			return errors.New("network member can not invite user")
		}
		if req.Role == models.RoleTypeAdmin && invitationNetworkUser.Role != models.RoleTypeOwner {
			return errors.New("only network owner can invite admin")
		}

		user := models.User{}
		if err := models.NewUserQuerySet(tx).EmailEq(req.Email).One(&user); err != nil {
			return errors.Errorf("cannot find user %s", req.Email)
		}

		// Check duplication invitation
		updateInvitation := false
		var unProcessInvitation models.Invitation
		err = models.NewInvitationQuerySet(tx).
			NetworkIDEq(networkID).
			UserIDEq(user.ID).
			One(&unProcessInvitation)
		switch err {
		case nil:
			updateInvitation = true
		case gorm.ErrRecordNotFound:
			updateInvitation = false
		default:
			return err
		}

		if updateInvitation {
			updater := models.NewInvitationQuerySet(tx).NetworkIDEq(networkID).UserIDEq(user.ID).GetUpdater()
			updater.SetRole(req.Role)
			res = &InviteMemberResponse{
				InvitationID: unProcessInvitation.ID,
			}
			return nil
		}

		invitation := &models.Invitation{
			NetworkID:   networkID,
			InvitedByID: userID,
			UserID:      user.ID,
		}
		if req.Role == models.RoleTypeAdmin {
			invitation.Role = req.Role
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
