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
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/pairmesh/pairmesh/cmd/pairportal/db"
	"github.com/pairmesh/pairmesh/cmd/pairportal/db/models"
	"github.com/pairmesh/pairmesh/errcode"
	"github.com/pairmesh/pairmesh/internal/jwt"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type (
	KeyListItem struct {
		KeyID   models.ID      `json:"key_id"`
		Type    models.KeyType `json:"type"`
		Key     string         `json:"key"`
		Created time.Time      `json:"created"`
		Expiry  time.Time      `json:"expiry"`
	}
	KeyListResponse struct {
		Keys []KeyListItem `json:"keys"`
	}
)

func (s *server) KeyList(ctx context.Context) (*KeyListResponse, error) {
	userID := jwt.UserIDFromContext(ctx)

	var keys []models.AuthKey
	err := db.Tx(func(tx *gorm.DB) error {
		return models.NewAuthKeyQuerySet(tx).UserIDEq(userID).OrderDescByCreatedAt().All(&keys)
	})
	if err != nil {
		return nil, err
	}

	res := &KeyListResponse{}
	for _, key := range keys {
		res.Keys = append(res.Keys, KeyListItem{
			KeyID:   key.ID,
			Type:    key.Type,
			Key:     key.Key,
			Created: key.CreatedAt,
		})
	}

	return res, nil
}

type (
	KeyType string

	CreateKeyRequest struct {
		Type models.KeyType `json:"type"`
	}
	CreateKeyResponse struct {
		KeyID   models.ID      `json:"key_id"`
		Type    models.KeyType `json:"type"`
		Key     string         `json:"key"`
		Created time.Time      `json:"created"`
	}
)

func (s *server) CreateKey(ctx context.Context, req *CreateKeyRequest) (*CreateKeyResponse, error) {
	if req.Type != models.KeyTypeOneOff &&
		req.Type != models.KeyTypeReusable &&
		req.Type != models.KeyTypeEphemeral {
		return nil, errcode.ErrIllegalRequest
	}

	userID := jwt.UserIDFromContext(ctx)
	newKey := uuid.New()
	key := &models.AuthKey{
		UserID: userID,
		Key:    fmt.Sprintf("mskey-%s", hex.EncodeToString(newKey[:])),
		Type:   req.Type,
	}

	err := db.Create(key)
	if err != nil {
		return nil, err
	}

	res := &CreateKeyResponse{
		KeyID:   key.ID,
		Type:    key.Type,
		Key:     key.Key,
		Created: key.CreatedAt,
	}

	return res, nil
}

type (
	ChangeKeyRequest struct {
	}
	ChangeKeyResponse struct {
	}

	ExchangeKeyResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
)

func (s *server) ChangeKey(ctx context.Context) (*ChangeKeyResponse, error) {
	return nil, nil
}

type (
	DeleteKeyResponse struct {
	}
)

func (s *server) DeleteKey(ctx context.Context, r *http.Request) (*DeleteKeyResponse, error) {
	vars := Vars(mux.Vars(r))
	keyId := vars.ModelID("key_id")
	if keyId == 0 {
		return nil, nil
	}

	userID := jwt.UserIDFromContext(ctx)
	return &DeleteKeyResponse{}, db.Tx(func(tx *gorm.DB) error {
		return models.NewAuthKeyQuerySet(tx).UserIDEq(userID).IDEq(keyId).Delete()
	})
}

func (s *server) ExchangeKey(ctx context.Context) (*ExchangeKeyResponse, error) {
	machineID := jwt.MachineIDFromContext(ctx)
	keyID := jwt.AuthKeyIDFromContext(ctx)

	// iOS Oneoff key
	if keyID == 0 {
		userID := jwt.UserIDFromContext(ctx)
		accessToken, refreshToken, err := jwt.Shared().CreateTokenPair(uint64(userID), machineID, 0, uint64(keyID), true)
		if err != nil {
			return nil, err
		}

		resp := &ExchangeKeyResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}
		return resp, nil
	}

	var authKey models.AuthKey
	err := db.Tx(func(tx *gorm.DB) error {
		return models.NewAuthKeyQuerySet(tx).IDEq(keyID).One(&authKey)
	})
	if err != nil {
		return nil, err
	}

	accessToken, refreshToken, err := jwt.Shared().CreateTokenPair(uint64(authKey.UserID), machineID, 0, uint64(keyID), true)
	if err != nil {
		return nil, err
	}

	// first time keyID exchange, save the machine id for one-machine check
	if authKey.Type == models.KeyTypeOneOff && authKey.MachineID == "" {
		err = db.Tx(func(tx *gorm.DB) error {
			updater := models.NewAuthKeyQuerySet(tx).IDEq(keyID).GetUpdater()
			return updater.SetMachineID(machineID).Update()
		})
		if err != nil {
			return nil, err
		}
	}

	resp := &ExchangeKeyResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	return resp, nil
}
