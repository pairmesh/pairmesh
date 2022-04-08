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

	"github.com/pairmesh/pairmesh/pkg/jwt"
	"github.com/pairmesh/pairmesh/portal/db"
	"github.com/pairmesh/pairmesh/portal/db/models"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type (
	// UserProfileResponse is the response to a user profile request
	UserProfileResponse struct {
		UserID     models.ID `json:"user_id"`
		Name       string    `json:"name"`
		Email      string    `json:"email"`
		Avatar     string    `json:"avatar"`
		Origin     string    `json:"origin"`
		CreateDate int64     `json:"create_date"`
	}
)

// UserProfile collects the profile of the user and formats into UserProfileResponse
func (s *server) UserProfile(ctx context.Context) (*UserProfileResponse, error) {
	var res *UserProfileResponse
	err := db.Tx(func(tx *gorm.DB) error {
		userID := models.ID(jwt.UserIDFromContext(ctx))
		var user models.User
		err := models.NewUserQuerySet(tx).IDEq(userID).One(&user)
		if err != nil {
			return errors.WithStack(err)
		}
		res = &UserProfileResponse{
			UserID:     userID,
			Name:       user.Name,
			Email:      user.Email,
			Avatar:     user.Avatar,
			Origin:     user.Origin,
			CreateDate: user.CreatedAt.UnixNano() / 1e6,
		}
		return nil
	})
	return res, err
}
