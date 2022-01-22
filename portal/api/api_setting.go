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

	"github.com/pairmesh/pairmesh/errcode"
	"github.com/pairmesh/pairmesh/pkg/jwt"
	"github.com/pairmesh/pairmesh/portal/db"
	"github.com/pairmesh/pairmesh/portal/db/models"

	"gorm.io/gorm"
)

type (
	UserProfileSettingRequest struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	UserProfileSettingResponse struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
)

func (s *server) UserProfileSetting(ctx context.Context, req *UserProfileSettingRequest) (*UserProfileSettingResponse, error) {
	if req.Name == "" && req.Email == "" {
		return nil, errcode.ErrIllegalRequest
	}

	// TODO: validate

	var res *UserProfileSettingResponse
	err := db.Tx(func(tx *gorm.DB) error {
		userID := models.ID(jwt.UserIDFromContext(ctx))
		var user models.User
		err := models.NewUserQuerySet(tx).IDEq(userID).One(&user)
		if err != nil {
			return err
		}

		res = &UserProfileSettingResponse{
			Name:  user.Name,
			Email: user.Email,
		}

		// Nothing changed.
		if req.Name == user.Name && req.Email == user.Email {
			return nil
		}

		updater := models.NewUserQuerySet(tx).IDEq(userID).GetUpdater()
		if req.Name != "" && req.Name != user.Name {
			res.Name = req.Name
			updater.SetName(req.Name)
		}
		if req.Email != "" && req.Email != user.Email {
			res.Email = req.Email
			updater.SetEmail(req.Email)
		}

		return updater.Update()
	})

	return res, err
}
