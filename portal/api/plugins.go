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
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/pairmesh/pairmesh/internal/ledis"

	"github.com/coreos/go-semver/semver"
	"github.com/pairmesh/pairmesh/constant"
	"github.com/pairmesh/pairmesh/errcode"
	"github.com/pairmesh/pairmesh/pkg/jwt"
	"github.com/pairmesh/pairmesh/pkg/logutil"
	"github.com/pairmesh/pairmesh/portal/db"
	"github.com/pairmesh/pairmesh/portal/db/models"
	"github.com/pairmesh/pairmesh/version"
	"github.com/pingcap/fn"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type versionKey struct{}

// setupMiddleware sets up all middlewares, e.g:
// 1. Load all plugins
// 2. Set the error encoder
// 3. set the response encoder
func setupMiddleware() {
	fn.Plugin(func(ctx context.Context, request *http.Request) (context.Context, error) {
		if logutil.IsEnablePortal() {
			zap.L().Debug("Incoming request", zap.Stringer("url", request.URL))
		}
		return context.WithValue(ctx, constant.KeyRawRequest, request), nil
	})

	type failure struct {
		Code  errcode.ErrCode `json:"code"`
		Error string          `json:"error"`
	}

	// Define a error encoder to unify all error response
	fn.SetErrorEncoder(func(ctx context.Context, err error) interface{} {
		request := ctx.Value(constant.KeyRawRequest).(*http.Request)
		zap.L().Error("Request failure",
			zap.String("api", request.RequestURI),
			zap.String("method", request.Method),
			zap.String("remote", request.RemoteAddr),
			zap.Error(err))

		code := errcode.InternalError

		var e errcode.Error
		if errors.As(err, &e) {
			code = e.Code
		}
		return &failure{
			Code:  code,
			Error: err.Error(),
		}
	})

	// Define a body response to unify all success response
	fn.SetResponseEncoder(func(ctx context.Context, payload interface{}) interface{} {
		if logutil.IsEnablePortal() {
			request := ctx.Value(constant.KeyRawRequest).(*http.Request)
			zap.L().Debug("Request success",
				zap.String("api", request.RequestURI),
				zap.String("method", request.Method),
				zap.String("remote", request.RemoteAddr),
				zap.Reflect("data", payload))
		}
		return payload
	})
}

// extractToken extract the token from the raw string
func extractToken(raw string) (string, string, error) {
	fields := strings.Split(raw, " ")
	if len(fields) == 2 {
		return strings.TrimSpace(fields[0]), strings.TrimSpace(fields[1]), nil
	}
	return "", "", errcode.ErrInvalidToken
}

// extractTokenFromRequest extract the token from http's Authorization header
func extractTokenFromRequest(r *http.Request) (string, string, error) {
	token := r.Header.Get(constant.HeaderAuthentication)
	return extractToken(token)
}

// extractMachineIDFromRequest extract the machine id from http's Authorization header
func extractMachineIDFromRequest(r *http.Request) string {
	return r.Header.Get(constant.HeaderXMachineID)
}

func peerAuthKeyValidator(ctx context.Context, key, machineID string) (context.Context, error) {
	var authKey models.AuthKey
	err := db.Tx(func(tx *gorm.DB) error {
		return models.NewAuthKeyQuerySet(tx).KeyEq(key).One(&authKey)
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ctx, errcode.ErrNotFound
		}
		return ctx, errcode.ErrServerInternal
	}

	if authKey.DeletedAt != nil {
		zap.L().Error("The auth key is invalid", zap.String("key", key), zap.Any("deleted_at", authKey.DeletedAt))
		return ctx, errcode.ErrInvalidToken
	}

	// one-machine
	if authKey.Type == models.KeyTypeOneOff && authKey.MachineID != machineID && authKey.MachineID != "" {
		zap.L().Error("Mismatch auth key's machine id", zap.String("actual", machineID), zap.String("want", authKey.MachineID))
		return ctx, errcode.ErrInvalidToken
	}

	ctx = jwt.ContextWithUserID(ctx, uint64(authKey.UserID))
	ctx = jwt.ContextWithMachineID(ctx, machineID)
	ctx = jwt.ContextWithAuthKeyID(ctx, uint64(authKey.ID))
	return ctx, nil
}

func peerOneoffValidator(ctx context.Context, key, machineID string) (context.Context, error) {
	val, err := ledis.Shared().Get(ctx, key)
	if err != nil {
		zap.L().Error("The oneoff key is invalid", zap.String("key", key), zap.Error(err))
		return ctx, errcode.ErrInvalidToken
	}

	userID, err := strconv.ParseUint(val.(string), 10, 64)
	if err != nil {
		zap.L().Error("The oneoff key user id is invalid", zap.String("userID", val.(string)), zap.Error(err))
		return ctx, errcode.ErrInvalidToken
	}

	ctx = jwt.ContextWithUserID(ctx, userID)
	ctx = jwt.ContextWithMachineID(ctx, machineID)
	return ctx, nil
}

func withVersion(ctx context.Context, version string) context.Context {
	return context.WithValue(ctx, versionKey{}, version)
}

func versionFromContext(ctx context.Context) string {
	return ctx.Value(versionKey{}).(string)
}

func peerTokenValidator(ctx context.Context, r *http.Request) (context.Context, error) {
	v := r.Header.Get(constant.HeaderXClientVersion)
	if v == "" {
		return ctx, errcode.ErrIllegalRequest
	}
	ver, err := semver.NewVersion(v)
	if err != nil {
		return ctx, err
	}
	if ver.Major < version.MajorVersion {
		return nil, errcode.ErrIllegalRequest
	}
	return tokenValidator(withVersion(ctx, v), r)
}

func tokenValidator(ctx context.Context, r *http.Request) (context.Context, error) {
	prefix, token, err := extractTokenFromRequest(r)
	if err != nil {
		zap.L().Error("Extract token from request is failed", zap.Error(err))
		return ctx, err
	}

	if prefix != constant.PrefixAuthKey && prefix != constant.PrefixJwtToken && prefix != constant.PrefixFastKey {
		zap.L().Error("Invalid token prefix", zap.String("prefix", prefix))
		return nil, errcode.ErrInvalidToken
	}

	machineID := extractMachineIDFromRequest(r)

	// use auth key
	if prefix == constant.PrefixAuthKey {
		return peerAuthKeyValidator(ctx, token, machineID)
	}

	if prefix == constant.PrefixFastKey {
		return peerOneoffValidator(ctx, token, machineID)
	}

	// use jwt token
	auth, err := jwt.Shared().ExtractTokenMetadata(r)
	if err != nil {
		zap.L().Error("Extract token metadata is failed", zap.Error(err))
		return ctx, errcode.ErrInvalidToken
	}
	if auth.MachineID != machineID {
		zap.L().Error("Mismatch machine id", zap.String("actual", machineID), zap.String("want", auth.MachineID))
		return ctx, errcode.ErrInvalidToken
	}

	userID, err := jwt.Shared().FetchAuth(auth)
	if err != nil {
		zap.L().Error("Fetch auth info is failed", zap.Error(err))
		return ctx, errcode.ErrInvalidToken
	}

	ctx = jwt.ContextWithUserID(ctx, userID)
	ctx = jwt.ContextWithMachineID(ctx, machineID)
	ctx = jwt.ContextWithAuthKeyID(ctx, auth.AuthKeyID)
	return ctx, nil
}

func relayAuthKeyValidator(relayAuthKey string) fn.PluginFunc {
	return func(ctx context.Context, request *http.Request) (context.Context, error) {
		key := request.Header.Get(constant.HeaderAuthentication)
		if key != relayAuthKey {
			return ctx, errcode.ErrInvalidAuthKey
		}
		return ctx, nil
	}
}

type Vars map[string]string

func (v Vars) Uint64(key string) uint64 {
	val, found := v[key]
	if !found {
		return 0
	}
	n, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func (v Vars) ModelID(key string) models.ID {
	return models.ID(v.Uint64(key))
}
