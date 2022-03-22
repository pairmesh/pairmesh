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

package models

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/pairmesh/pairmesh/security"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateUser create a new user
func CreateUser(tx *gorm.DB, user *User, ssoUser interface{}) error {
	err := tx.Create(user).Error
	if err != nil {
		return err
	}

	switch t := ssoUser.(type) {
	case *GithubUser:
		t.UserID = user.ID
	case *WechatUser:
		t.UserID = user.ID
	}

	return tx.Create(ssoUser).Error
}

// BuildUser generate a stub user for sso
func BuildUser() (User, error) {
	secretKey := [32]byte{}
	_, err := rand.Read(secretKey[:])
	if err != nil {
		return User{}, err
	}
	salt := uuid.New().String()
	return User{
		Salt:      salt,
		Hash:      security.Hash(uuid.New().String(), salt),
		SecretKey: base64.RawStdEncoding.EncodeToString(secretKey[:]),
	}, nil
}
