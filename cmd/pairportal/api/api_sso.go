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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/pairmesh/pairmesh/internal/redis"

	"github.com/pairmesh/pairmesh/cmd/pairportal/db"
	"github.com/pairmesh/pairmesh/cmd/pairportal/db/models"
	"github.com/pairmesh/pairmesh/cmd/pairportal/sso"
	"github.com/pairmesh/pairmesh/internal/jwt"

	"github.com/pingcap/fn"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ssoServer struct {
	client   *http.Client
	redirect string
}

var errUnknownSSOProvider = errors.New("unknown sso provider")

func newSSOServer(redirect string) *ssoServer {
	srv := ssoServer{
		redirect: redirect,
		client:   &http.Client{},
	}
	return &srv
}

func (s *ssoServer) userInfo(name sso.Vendor, token *sso.Token) (*models.User, bool, error) {
	p := sso.WithName(name)
	if p == nil {
		zap.L().Error("unknown sso provider", zap.Any("name", name))
		return nil, false, errUnknownSSOProvider
	}
	return p.UserInfo(token)
}

type SSOMethod struct {
	Name string `json:"name"`
	Link string `json:"link"`
}

// SSOMethods returns all SSO login methods
func (s *ssoServer) SSOMethods(form *fn.Form) ([]*SSOMethod, error) {
	links := sso.AuthCodeLinks(s.redirect, form.Get("node"))

	var res []*SSOMethod
	for _, link := range links {
		res = append(res, &SSOMethod{
			Name: link.Name,
			Link: link.Link,
		})
	}

	return res, nil
}

type UserInfo struct {
	ID     uint64 `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	Origin string `json:"origin"`
}

type CallbackResponse struct {
	NotifyClient int       `json:"notify_client"`
	AccessToken  string    `json:"access_token"`
	User         *UserInfo `json:"user"`
}

func (s *ssoServer) AuthCallback(ctx context.Context, form *fn.Form) (*CallbackResponse, error) {
	code := form.Get("code")
	if code == "" {
		return nil, fmt.Errorf("illegal parameter, no code")
	}

	p := form.IntOrDefault("p", -1)
	if p < 0 {
		return nil, fmt.Errorf("illegal parameter")
	}

	vendor := sso.Vendor(p)
	provider := sso.WithName(vendor)
	if provider == nil {
		return nil, fmt.Errorf("illegal parameter: unrecognized provider")
	}

	token, err := provider.AccessToken(code)
	if err != nil {
		return nil, err
	}

	user, _, err := s.userInfo(vendor, token)
	if err != nil {
		return nil, err
	}

	if form.Get("t") == "fk" {
		token := uuid.New().String()
		err := redis.Shared().Set(ctx, token, fmt.Sprintf("%d", user.ID), time.Second*60)
		if err != nil {
			return nil, fmt.Errorf("internal error to save token: %s", err)
		}

		userInfo := &UserInfo{
			ID:     uint64(user.ID),
			Name:   user.Name,
			Avatar: user.Avatar,
			Origin: user.Origin,
		}
		res := &CallbackResponse{
			AccessToken: token,
			//RefreshToken: ts.RefreshToken,
			User: userInfo,
		}

		return res, nil
	}

	// Desktop use node information to pass the machine info
	type nodeInfo struct {
		Port    int    `json:"port"`
		Machine string `json:"machine"`
	}
	info := &nodeInfo{}
	if node := form.Get("node"); node != "" {
		data, err := base64.RawStdEncoding.DecodeString(node)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(data, info)
		if err != nil {
			return nil, err
		}
	}

	// Mobile will call the RESTful API directly use m parameter
	if info.Machine == "" {
		info.Machine = form.Get("m")
	}

	userID := uint64(user.ID)
	ts, err := jwt.Shared().CreateToken(userID, info.Machine, uint8(vendor), 0, false)
	if err != nil {
		return nil, err
	}

	err = jwt.Shared().CreateAuth(userID, ts)
	if err != nil {
		return nil, err
	}

	userInfo := &UserInfo{
		ID:     userID,
		Name:   user.Name,
		Avatar: user.Avatar,
		Origin: user.Origin,
	}
	res := &CallbackResponse{
		NotifyClient: info.Port,
		AccessToken:  ts.AccessToken,
		//RefreshToken: ts.RefreshToken,
		User: userInfo,
	}

	return res, nil
}

func (s *ssoServer) Logout(w http.ResponseWriter, r *http.Request) {
	metadata, err := jwt.Shared().ExtractTokenMetadata(r)
	if err != nil {
		zap.L().Info("parse token is failed", zap.Error(err))
		return
	}

	if err = jwt.Shared().Logout(metadata); err != nil {
		zap.L().Error("delete token is failed", zap.Error(err))
		return
	}

	// not exchange from auth key
	if metadata.AuthKeyID == 0 {
		return
	}

	now := time.Now()
	err = db.Tx(func(tx *gorm.DB) error {
		var authKey models.AuthKey
		if err = models.NewAuthKeyQuerySet(tx).IDEq(models.ID(metadata.AuthKeyID)).One(&authKey); err != nil {
			return fmt.Errorf("get auth key's by id is failed: %w", err)
		}

		// not a ephemeral key
		if authKey.Type != models.KeyTypeEphemeral {
			return nil
		}

		updater := models.NewAuthKeyQuerySet(tx).IDEq(models.ID(metadata.AuthKeyID)).GetUpdater()
		if err = updater.SetDeletedAt(&now).Update(); err != nil {
			return fmt.Errorf("delete key is failed: %w", err)
		}
		return nil
	})
	if err != nil {
		zap.L().Error("", zap.Error(err))
	}

}
