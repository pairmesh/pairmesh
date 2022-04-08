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
	"strconv"

	"github.com/pairmesh/pairmesh/errcode"
	"github.com/pairmesh/pairmesh/pkg/jwt"
	"github.com/pairmesh/pairmesh/portal/db"
	"github.com/pairmesh/pairmesh/portal/db/models"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type (
	// InvitationListItem is the single item struct of a invitation in invitation list
	InvitationListItem struct {
		InvitationID       models.ID `json:"invitation_id"`
		NetworkID          models.ID `json:"network_id"`
		NetworkName        string    `json:"network_name"`
		InvitedByUserName  string    `json:"invited_by_user_name"`
		InvitedByUserEmail string    `json:"invited_by_user_email"`
		InviteUserName     string    `json:"invite_user_name"`
		InviteDeviceCount  uint      `json:"invite_device_count"`
	}

	// InvitationListResponse is the struct with a list of invitations
	InvitationListResponse struct {
		Invitations []InvitationListItem `json:"invitations"`
	}
)

// Invitations returns the invitation list associated to the user
func (s *server) Invitations(ctx context.Context) (*InvitationListResponse, error) {
	userID := models.ID(jwt.UserIDFromContext(ctx))
	res := &InvitationListResponse{}
	var invitations []models.Invitation
	err := db.Tx(func(tx *gorm.DB) error {
		return models.NewInvitationQuerySet(tx).
			PreloadNetwork().
			PreloadInvitedBy().
			PreloadUser().
			UserIDEq(userID).
			All(&invitations)
	})
	for _, inv := range invitations {
		res.Invitations = append(res.Invitations, InvitationListItem{
			InvitationID:       inv.ID,
			NetworkID:          inv.NetworkID,
			NetworkName:        inv.Network.Name,
			InvitedByUserName:  inv.InvitedBy.Name,
			InvitedByUserEmail: inv.InvitedBy.Email,
			InviteUserName:     inv.User.Name,
		})
	}
	return res, err
}

type (
	// HandleInvitationRequest is the request to handle invitation
	HandleInvitationRequest struct {
		Action string `json:"action"`
	}

	// HandleInvitationResponse is the response to handle invitation
	HandleInvitationResponse struct {
		InvitationID models.ID `json:"invitation_id"`
	}
)

// HandleInvitation handles invitation with request as given parameters
func (s *server) HandleInvitation(ctx context.Context, raw *http.Request, req *HandleInvitationRequest) (*HandleInvitationResponse, error) {
	vars := mux.Vars(raw)
	id, found := vars["invitation_id"]
	if !found {
		return nil, errcode.ErrIllegalRequest
	}
	invitationID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, err
	}
	if invitationID <= 0 {
		return nil, errcode.ErrIllegalRequest
	}
	if req.Action != "join" && req.Action != "decline" {
		return nil, errcode.ErrIllegalRequest
	}

	userID := models.ID(jwt.UserIDFromContext(ctx))
	var res *HandleInvitationResponse
	err = db.Tx(func(tx *gorm.DB) error {
		var invitation models.Invitation
		err := models.NewInvitationQuerySet(tx).
			IDEq(models.ID(invitationID)).
			One(&invitation)
		if err != nil {
			return err
		}

		if userID != invitation.UserID {
			return errcode.ErrIllegalRequest
		}

		err = models.NewInvitationQuerySet(tx).
			IDEq(models.ID(invitationID)).
			Delete()
		if err != nil {
			return err
		}

		if req.Action == "join" {
			teamUser := &models.NetworkUser{
				NetworkID: invitation.NetworkID,
				UserID:    userID,
				Role:      invitation.Role,
			}
			tx.Create(teamUser)
			if tx.Error != nil {
				return tx.Error
			}
		}

		res = &HandleInvitationResponse{
			InvitationID: models.ID(invitationID),
		}
		return nil
	})
	return res, err
}
