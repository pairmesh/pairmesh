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

	"github.com/pairmesh/pairmesh/errcode"
	"github.com/pairmesh/pairmesh/internal/jwt"
	"github.com/pairmesh/pairmesh/portal/db"
	"github.com/pairmesh/pairmesh/portal/db/models"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type (
	InvitationListItem struct {
		InvitationID      models.ID `json:"invitation_id"`
		NetworkID         models.ID `json:"network_id"`
		NetworkName       string    `json:"network_name"`
		InvitedByUserName string    `json:"invited_by_user_name"`
		InviteUserName    string    `json:"invite_user_name"`
		InviteDeviceCount uint      `json:"invite_device_count"`
	}
	InvitationListResponse struct {
		Invitations []InvitationListItem `json:"invitations"`
	}
)

// Invitations returns the invitation list associated to the user
func (s *server) Invitations(ctx context.Context, form *fn.Form) (*InvitationListResponse, error) {
	action := form.Get("action")
	if action != "send" && action != "receive" {
		return nil, errcode.ErrIllegalRequest
	}
	userID := jwt.UserIDFromContext(ctx)
	res := &InvitationListResponse{}
	var invitations []models.Invitation
	err := db.Tx(func(tx *gorm.DB) error {
		if action == "send" {
			return models.NewInvitationQuerySet(tx).PreloadNetwork().PreloadInvitedBy().PreloadUser().InvitedByIDEq(userID).All(&invitations)
		}
		return models.NewInvitationQuerySet(tx).PreloadNetwork().PreloadInvitedBy().PreloadUser().UserIDEq(userID).StatusEq(models.InvitationStatusTypeUnprocessed).All(&invitations)
	})
	nowNano := time.Now().UnixNano()
	for _, inv := range invitations {
		if action == "receive" && nowNano > inv.ExpiredAt.UnixNano() {
			continue
		}
		res.Invitations = append(res.Invitations, InvitationListItem{
			InvitationID:      inv.ID,
			NetworkID:         inv.NetworkID,
			NetworkName:       inv.Network.Name,
			InvitedByUserName: inv.InvitedBy.Name,
			InviteUserName:    inv.User.Name,
			InviteDeviceCount: inv.DeviceLimit,
		})
	}
	return res, err
}

type (
	AcceptInvitationRequest struct {
		DeviceIDs []models.ID `json:"device_ids"`
	}
	AcceptInvitationResponse struct {
		Network NetworkItem `json:"network"`
	}
	HandleInvitationRequest struct {
		Action string `json:"action"`
	}
	HandleInvitationResponse struct {
		InvitationID models.ID `json:"invitation_id"`
	}
)

//AcceptInvitation accept invitation and create network device (add device to network)
func (s *server) AcceptInvitation(ctx context.Context, r *http.Request, req *AcceptInvitationRequest) (*AcceptInvitationResponse, error) {
	vars := Vars(mux.Vars(r))
	invitationID := vars.ModelID("invitation_id")
	deviceIDs := req.DeviceIDs
	if invitationID == 0 || len(deviceIDs) == 0 {
		return nil, errcode.ErrIllegalRequest
	}
	var res *AcceptInvitationResponse
	err := db.Tx(func(tx *gorm.DB) error {
		userID := jwt.UserIDFromContext(ctx)
		var invitation models.Invitation
		if err := models.NewInvitationQuerySet(tx).IDEq(invitationID).One(&invitation); err != nil {
			return err
		}
		if invitation.UserID != userID || invitation.DeviceLimit == 0 || invitation.Status != models.InvitationStatusTypeUnprocessed {
			return errors.New("invitation invalid")
		}
		if len(deviceIDs) > int(invitation.DeviceLimit) {
			return errors.New("exceeds invitation device limit")
		}
		if invitation.ExpiredAt.Before(time.Now()) {
			models.NewInvitationQuerySet(tx).IDEq(invitationID).GetUpdater().SetStatus(models.InvitationStatusTypeReject).Update()
			return errors.New("expired invitation are auto reject")
		}
		var invitationNetworkUser models.NetworkUser //invite user
		if err := models.NewNetworkUserQuerySet(tx).PreloadNetwork().NetworkIDEq(invitation.NetworkID).UserIDEq(invitation.InvitedByID).One(&invitationNetworkUser); err != nil {
			return err
		}
		if invitationNetworkUser.Role == models.RoleTypeMember {
			return errors.New("network member can not invite user")
		}
		if invitation.Role == models.RoleTypeAdmin && invitationNetworkUser.Role != models.RoleTypeOwner {
			return errors.New("only network owner can invite admin")
		}
		userInfo := &networkUserInfo{
			networkID: invitation.NetworkID,
			userID:    invitation.UserID,
			role:      invitation.Role,
			expiredAt: invitation.UserExpiredAt,
		}
		if err := s.createNetworkUser(tx, userInfo); err != nil {
			return err
		}

		for _, deviceID := range deviceIDs {
			err := s.createNetworkDevice(tx, &networkDeviceInfo{
				networkID: invitation.NetworkID,
				userID:    userID,
				deviceID:  deviceID,
				expiredAt: invitation.UserExpiredAt,
			})
			if err != nil {
				return err
			}
		}

		if err := models.NewInvitationQuerySet(tx).IDEq(invitationID).GetUpdater().SetStatus(models.InvitationStatusTypeAccept).Update(); err != nil {
			return err
		}

		var network models.Network
		if err := models.NewNetworkQuerySet(tx).PreloadCreatedBy().IDEq(invitation.NetworkID).One(&network); err != nil {
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
			Role:              invitation.Role,
		}
		res = &AcceptInvitationResponse{
			Network: item,
		}

		return nil
	})

	return res, err
}

//HandleInvitation handle invitation (reject or revoke)
func (s *server) HandleInvitation(ctx context.Context, r *http.Request, req *HandleInvitationRequest) (*HandleInvitationResponse, error) {
	vars := Vars(mux.Vars(r))
	invitationID := vars.ModelID("invitation_id")
	if invitationID == 0 {
		return nil, errcode.ErrIllegalRequest
	}
	if req.Action != "reject" && req.Action != "revoke" {
		return nil, errcode.ErrIllegalRequest
	}
	var res *HandleInvitationResponse
	err := db.Tx(func(tx *gorm.DB) error {
		userID := jwt.UserIDFromContext(ctx)
		var invitation models.Invitation
		if err := models.NewInvitationQuerySet(tx).IDEq(invitationID).One(&invitation); err != nil {
			return err
		}
		if invitation.UserID != userID || invitation.DeviceLimit == 0 || invitation.Status != models.InvitationStatusTypeUnprocessed {
			return errors.New("invitation invalid")
		}
		status := models.InvitationStatusTypeReject
		if req.Action == "revoke" {
			status = models.InvitationStatusTypeRevoke
		}

		if err := models.NewInvitationQuerySet(tx).IDEq(invitationID).GetUpdater().SetStatus(status).Update(); err != nil {
			return err
		}
		res = &HandleInvitationResponse{
			InvitationID: invitationID,
		}
		return nil
	})

	return res, err
}
