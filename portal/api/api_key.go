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

	"github.com/pairmesh/pairmesh/errcode"
	"github.com/pairmesh/pairmesh/pkg/jwt"
	"github.com/pairmesh/pairmesh/portal/db"
	"github.com/pairmesh/pairmesh/portal/db/models"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type (
	// KeyListItem is the single item struct for a key in a key list
	KeyListItem struct {
		KeyID   models.ID      `json:"key_id"`
		Type    models.KeyType `json:"type"`
		Key     string         `json:"key"`
		Created time.Time      `json:"created"`
		Expiry  time.Time      `json:"expiry"`
		Enabled bool           `json:"enabled"`
	}

	// KeyListResponse is the response to a key list request
	KeyListResponse struct {
		Keys []KeyListItem `json:"keys"`
	}
)

// KeyList returns a key list in format of KeyListResponse
func (s *server) KeyList(ctx context.Context) (*KeyListResponse, error) {
	userID := models.ID(jwt.UserIDFromContext(ctx))

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
			Key:     key.Key[:12] + "...",
			Created: key.CreatedAt,
			Expiry:  key.ExpiredAt,
			Enabled: key.Enabled,
		})
	}

	return res, nil
}

type (
	// KeyType as string alias, represents type of a key
	KeyType string

	// CreateKeyRequest is the request to create a key
	CreateKeyRequest struct {
		Type models.KeyType `json:"type"`
	}

	// CreateKeyResponse is the response to the request to create a key
	CreateKeyResponse struct {
		KeyID   models.ID      `json:"key_id"`
		Type    models.KeyType `json:"type"`
		Key     string         `json:"key"`
		Created time.Time      `json:"created"`
		Expiry  time.Time      `json:"expiry"`
		Enabled bool           `json:"enabled"`
	}
)

// CreateKey handles key creation
func (s *server) CreateKey(ctx context.Context, req *CreateKeyRequest) (*CreateKeyResponse, error) {
	if req.Type != models.KeyTypeOneOff && req.Type != models.KeyTypeReusable {
		return nil, errcode.ErrIllegalRequest
	}

	userID := models.ID(jwt.UserIDFromContext(ctx))
	newKey := uuid.New()
	key := &models.AuthKey{
		UserID:    userID,
		Key:       fmt.Sprintf("pmkey-%s", hex.EncodeToString(newKey[:])),
		Type:      req.Type,
		ExpiredAt: time.Now().Add(90 * 24 * time.Hour),
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
		Expiry:  key.ExpiredAt,
		Enabled: true,
	}

	return res, nil
}

type (
	// ChangeKeyRequest is request to change key
	ChangeKeyRequest struct {
		Op string `json:"op"`
	}

	// ChangeKeyResponse is response to requests to change key
	ChangeKeyResponse struct {
	}

	// ExchangeKeyResponse is response to requests to exchange key
	ExchangeKeyResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
)

// ChangeKey updates key in database and returns response accordingly
func (s *server) ChangeKey(ctx context.Context, r *http.Request, req *ChangeKeyRequest) (*ChangeKeyResponse, error) {
	vars := Vars(mux.Vars(r))
	if req.Op != "enable" && req.Op != "disable" {
		return nil, errcode.ErrIllegalRequest
	}

	keyID := vars.ModelID("key_id")
	if keyID == 0 {
		return nil, nil
	}

	userID := models.ID(jwt.UserIDFromContext(ctx))
	return &ChangeKeyResponse{}, db.Tx(func(tx *gorm.DB) error {
		return models.NewAuthKeyQuerySet(tx).UserIDEq(userID).IDEq(keyID).GetUpdater().SetEnabled(req.Op == "enable").Update()
	})
}

type (
	// DeleteKeyResponse is struct as response to Deleting a key
	DeleteKeyResponse struct {
	}
)

// DeleteKey handles key deletion
func (s *server) DeleteKey(ctx context.Context, r *http.Request) (*DeleteKeyResponse, error) {
	vars := Vars(mux.Vars(r))
	keyID := vars.ModelID("key_id")
	if keyID == 0 {
		return nil, nil
	}

	userID := models.ID(jwt.UserIDFromContext(ctx))
	return &DeleteKeyResponse{}, db.Tx(func(tx *gorm.DB) error {
		return models.NewAuthKeyQuerySet(tx).UserIDEq(userID).IDEq(keyID).Delete()
	})
}

// ExchangeKey handles key exchange
func (s *server) ExchangeKey(ctx context.Context) (*ExchangeKeyResponse, error) {
	machineID := jwt.MachineIDFromContext(ctx)
	keyID := models.ID(jwt.AuthKeyIDFromContext(ctx))

	// iOS Oneoff key
	if keyID == 0 {
		userID := models.ID(jwt.UserIDFromContext(ctx))
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
