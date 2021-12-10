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

package queryutil

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/pairmesh/pairmesh/cmd/pairportal/db"
	"github.com/pairmesh/pairmesh/cmd/pairportal/db/models"
	"github.com/pairmesh/pairmesh/security"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UpsertMERPNode update or insert the relay server's information
func UpsertMERPNode(node *models.RelayServer) error {
	return db.Tx(func(tx *gorm.DB) error {
		tx.Save(node)
		return tx.Error
	})
}

// CreateUser create a new user
func CreateUser(user *models.User, ssoUser interface{}) error {
	err := db.Tx(func(tx *gorm.DB) error {
		err := tx.Create(user).Error
		if err != nil {
			return err
		}

		switch t := ssoUser.(type) {
		case *models.GithubUser:
			t.UserID = user.ID
		case *models.WechatUser:
			t.UserID = user.ID
		}

		return tx.Create(ssoUser).Error
	}, nil)

	return err
}

// BuildUser generate a stub user for sso
func BuildUser() models.User {
	secretKey := [32]byte{}
	rand.Read(secretKey[:])
	salt := uuid.New().String()
	return models.User{
		Salt:      salt,
		Hash:      security.Hash(uuid.New().String(), salt),
		SecretKey: base64.RawStdEncoding.EncodeToString(secretKey[:]),
	}
}
