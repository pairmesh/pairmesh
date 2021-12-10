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

	"github.com/pairmesh/pairmesh/cmd/pairportal/db"
	"github.com/pairmesh/pairmesh/cmd/pairportal/db/models"
	"github.com/pairmesh/pairmesh/internal/jwt"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type (
	UserProfileResponse struct {
		UserID     models.ID `json:"user_id"`
		Name       string    `json:"name"`
		Avatar     string    `json:"avatar"`
		Origin     string    `json:"origin"`
		CreateDate int64     `json:"create_date"`
	}
)

func (s *server) UserProfile(ctx context.Context) (*UserProfileResponse, error) {
	var res *UserProfileResponse
	err := db.Tx(func(tx *gorm.DB) error {
		userID := jwt.UserIDFromContext(ctx)
		var user models.User
		err := models.NewUserQuerySet(tx).IDEq(userID).One(&user)
		if err != nil {
			return errors.WithStack(err)
		}
		res = &UserProfileResponse{
			UserID:     userID,
			Name:       user.Name,
			Avatar:     user.Avatar,
			Origin:     user.Origin,
			CreateDate: user.CreatedAt.UnixNano() / 1e6,
		}
		return nil
	})
	return res, err
}
