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

package jwt

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	stdjwt "github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	// ErrNotFound represents can not find the value with the key
	ErrNotFound = errors.New("not found")

	// ErrUnauthorized represents the current user doesn't have the right to do this operation
	ErrUnauthorized = errors.New("unauthorized")

	// ErrTokenExpired represents the token is expired
	ErrTokenExpired = errors.New("token expired")

	// ErrIllegalParameter represents the input parameter(s) is illegal
	ErrIllegalParameter = errors.New("illegal parameter")

	// ErrInternalServerError represents the server do sth wrong
	ErrInternalServerError = errors.New("internal server error")
)

// Store represents something for store the jwt token
type Store interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Get(ctx context.Context, key string) (interface{}, error)
}

type memoryStore struct {
	m map[string]interface{}
}

func (s memoryStore) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	s.m[key] = value
	return nil
}
func (s memoryStore) Del(ctx context.Context, keys ...string) error {
	for _, k := range keys {
		delete(s.m, k)
	}
	return nil
}

func (s memoryStore) Get(ctx context.Context, key string) (interface{}, error) {
	return nil, nil
}

const refreshUUIDFormat = "%s.%d"

type jwt struct {
	opts options
}

var j *jwt

// Initialize initialize the jwt
func Initialize(accessSecret, refreshSecret string, opts ...Option) error {
	if accessSecret == "" {
		return ErrIllegalParameter
	}
	options := newOptions(opts...)
	options.accessSecret = accessSecret
	options.refreshSecret = refreshSecret

	j = &jwt{
		opts: options,
	}
	zap.L().Info("initialize the jwt successfully")
	return nil
}

var noExpiryTimestamp = time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

// CreateToken create a new jwt token for the user
func (s *jwt) CreateToken(userID uint64, machineID string, vendor uint8, authKeyID uint64, noExpiry bool) (*TokenDetails, error) {
	tok := &TokenDetails{}

	if noExpiry {
		tok.AccessExpiry = noExpiryTimestamp
		tok.RefreshExpiry = noExpiryTimestamp

	} else {
		tok.AccessExpiry = time.Now().Add(s.opts.accessTokenTTL).Unix()
		tok.RefreshExpiry = time.Now().Add(s.opts.refreshTokenTTL).Unix()
	}
	tok.AccessUUID = uuid.New().String()

	tok.RefreshUUID = fmt.Sprintf(refreshUUIDFormat, tok.AccessUUID, userID)

	var err error
	userIDStr := strconv.FormatUint(userID, 10)

	//creating access token
	accessClaims := stdjwt.MapClaims{}
	accessClaims["authorized"] = true
	accessClaims["access_uuid"] = tok.AccessUUID
	accessClaims["user_id"] = userIDStr
	accessClaims["machine_id"] = machineID
	accessClaims["vendor"] = vendor
	accessClaims["exp"] = tok.AccessExpiry
	accessClaims["auth_key_id"] = authKeyID
	at := stdjwt.NewWithClaims(stdjwt.SigningMethodHS256, accessClaims)

	tok.AccessToken, err = at.SignedString([]byte(s.opts.accessSecret))
	if err != nil {
		return nil, err
	}

	//creating refresh token
	refreshClaims := stdjwt.MapClaims{}
	refreshClaims["refresh_uuid"] = tok.RefreshUUID
	refreshClaims["user_id"] = userIDStr
	refreshClaims["exp"] = tok.RefreshExpiry
	refreshClaims["machine_id"] = machineID
	refreshClaims["vendor"] = vendor
	accessClaims["auth_key_id"] = authKeyID
	rt := stdjwt.NewWithClaims(stdjwt.SigningMethodHS256, refreshClaims)
	tok.RefreshToken, err = rt.SignedString([]byte(s.opts.refreshSecret))
	if err != nil {
		return nil, err
	}
	return tok, nil
}

// RefreshToken renew a jwt token pair for the user
func (s *jwt) RefreshToken(refreshToken string) (map[string]string, error) {
	token, err := stdjwt.Parse(refreshToken, func(token *stdjwt.Token) (interface{}, error) {
		//Make sure that the token method confirm to "SigningMethodHMAC"
		if _, ok := token.Method.(*stdjwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.opts.refreshSecret), nil
	})

	//if there is an error, the token must have expired
	if err != nil {
		zap.L().Error("the token is expired", zap.String("refresh_token", refreshToken), zap.Error(err))
		return nil, ErrUnauthorized
	}
	//is token valid?
	if _, ok := token.Claims.(stdjwt.Claims); !ok && !token.Valid { //nolint
		zap.L().Error("the token is invalid", zap.String("refresh_token", refreshToken))
		return nil, ErrUnauthorized
	}

	//since token is valid, get the uuid:
	claims, ok := token.Claims.(stdjwt.MapClaims) //the token claims should conform to MapClaims
	if !ok || !token.Valid {
		zap.L().Error("the token is expired", zap.String("refresh_token", refreshToken))
		return nil, ErrUnauthorized
	}

	refreshUUID, ok := claims["refresh_uuid"].(string) //convert the interface to string
	if !ok {
		zap.L().Error("the token's payload is invalid", zap.String("refresh_token", refreshToken))
		return nil, ErrUnauthorized
	}

	userID, err := strconv.ParseUint(fmt.Sprintf("%.f", claims["user_id"]), 10, 64)
	if err != nil {
		zap.L().Error("the token's payload is invalid", zap.String("refresh_token", refreshToken))
		return nil, ErrUnauthorized
	}

	//delete the previous Refresh AccessToken
	err = s.DeleteAuth(refreshUUID)
	if err != nil { //if any goes wrong
		zap.L().Error("delete the old refresh token failed",
			zap.String("refresh_token", refreshToken))
		return nil, ErrUnauthorized
	}

	//create new pairs of refresh and access tokens
	vendor := uint8(claims["vendor"].(float32))

	//todo: parse noExpiry & authKeyId from refresh token
	tok, err := s.CreateToken(userID, claims["machine_id"].(string), vendor, 0, false)
	if err != nil {
		zap.L().Error(
			"create token failed",
			zap.Uint64("user_id", userID),
			zap.String("refresh_token", refreshToken),
			zap.Error(err))

		return nil, ErrInternalServerError
	}

	//save the tokens metadata to redis
	saveErr := s.CreateAuth(userID, tok)
	if saveErr != nil {
		zap.L().Error(
			"save token failed",
			zap.Uint64("user_id", userID),
			zap.String("refresh_token", refreshToken),
			zap.Error(saveErr))

		return nil, ErrInternalServerError
	}
	m := map[string]string{
		"access_token":  tok.AccessToken,
		"refresh_token": tok.RefreshToken,
	}

	return m, nil

}

// DeleteAuth delete a access & refresh pair
func (s *jwt) DeleteAuth(givenUUID string) error {
	return s.opts.store.Del(context.Background(), givenUUID)
}

// CreateAuth save a access & refresh pair
func (s *jwt) CreateAuth(uid uint64, tok *TokenDetails) (err error) {

	now := time.Now()
	at := time.Unix(tok.AccessExpiry, 0) //converting Unix to UTC(to Time object)
	rt := time.Unix(tok.RefreshExpiry, 0)

	ctx := context.Background()

	userID := strconv.Itoa(int(uid))
	err = s.opts.store.Set(ctx, tok.AccessUUID, userID, at.Sub(now))
	if err != nil {
		return
	}

	defer func() {
		if err != nil {
			e := s.opts.store.Del(ctx, tok.AccessUUID)
			if e != nil {
				zap.L().Error("delete token failed", zap.String("key", tok.AccessUUID), zap.Error(e))
			}
		}
	}()

	return s.opts.store.Set(ctx, tok.RefreshUUID, userID, rt.Sub(now))
}

// todo: broadcast it ?
func (s *jwt) DeleteTokens(metadata *AccessDetails) error {
	err := s.opts.store.Del(context.Background(), metadata.AccessUUID)
	if err != nil {
		return err
	}
	refreshUUID := fmt.Sprintf(refreshUUIDFormat, metadata.AccessUUID, metadata.UserID)
	err = s.opts.store.Del(context.Background(), refreshUUID)
	return err
}

// FetchAuth whenever a request is made that requires authentication, the method is called
func (s *jwt) FetchAuth(auth *AccessDetails) (uint64, error) {
	uid, err := s.opts.store.Get(context.Background(), auth.AccessUUID)
	if err != nil {
		return 0, err
	}
	userID, _ := strconv.ParseUint(uid.(string), 10, 64)
	return userID, nil

}
func (s *jwt) ExtractTokenMetadata(r *http.Request) (*AccessDetails, error) {
	_, tokenStr, err := ExtractTokenFromRequest(r)
	if err != nil {
		return nil, ErrUnauthorized
	}

	token, err := VerifyToken(tokenStr, s.opts.accessSecret)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(stdjwt.MapClaims)
	if ok && token.Valid {
		accessUUID, ok := claims["access_uuid"].(string)
		if !ok {
			return nil, err
		}
		userID, err := strconv.ParseUint(claims["user_id"].(string), 10, 64)
		if err != nil {
			return nil, err
		}

		machineID, ok := claims["machine_id"].(string)
		if !ok {
			return nil, err
		}

		var authKeyID uint64
		if claims["auth_key_id"] != nil {
			authKeyID = uint64(claims["auth_key_id"].(float64))
			if err != nil {
				return nil, err
			}
		}

		return &AccessDetails{
			AccessUUID: accessUUID,
			UserID:     userID,
			MachineID:  machineID,
			AuthKeyID:  authKeyID,
		}, nil
	}
	return nil, err
}

func (s *jwt) Logout(metadata *AccessDetails) error {
	return s.DeleteTokens(metadata)
}

// CreateTokenPair create & save the token pair and return them
func (s *jwt) CreateTokenPair(userID uint64, machineID string, vendor uint8, authKeyID uint64, noExpiry bool) (string, string, error) {
	ts, err := s.CreateToken(userID, machineID, vendor, authKeyID, noExpiry)
	if err != nil {
		return "", "", err
	}

	err = s.CreateAuth(userID, ts)
	if err != nil {
		return "", "", err
	}
	return ts.AccessToken, ts.RefreshToken, nil

}

// Shared represents the global jwt object
func Shared() *jwt {
	return j
}
