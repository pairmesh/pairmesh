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
	"fmt"
	"net/http"
	"strings"

	stdjwt "github.com/dgrijalva/jwt-go"
)

// ExtractToken extract the token from the raw string
func ExtractToken(raw string) (string, string, error) {
	fields := strings.Split(raw, " ")
	if len(fields) == 2 {
		return strings.TrimSpace(fields[0]), strings.TrimSpace(fields[1]), nil
	}
	return "", "", ErrNotFound
}

// ExtractTokenFromRequest extract the token from http's Authorization header
func ExtractTokenFromRequest(r *http.Request) (string, string, error) {
	token := r.Header.Get("Authorization")
	return ExtractToken(token)
}

// Valid valid and convert the token string to a object
func Valid(tokenStr string) (*stdjwt.Token, error) {
	return tokenValid(tokenStr, j.opts.accessSecret)
}
func tokenValid(tokenStr, accessSecret string) (*stdjwt.Token, error) {
	token, err := VerifyToken(tokenStr, accessSecret)
	if err != nil {
		return nil, err
	}
	if _, ok := token.Claims.(stdjwt.Claims); !ok || !token.Valid { //nolint
		return nil, err
	}
	return token, nil

}

// VerifyToken parse, validate, and return a token
func VerifyToken(token string, accessSecret string) (*stdjwt.Token, error) {
	tok, err := stdjwt.Parse(token, func(tk *stdjwt.Token) (interface{}, error) {
		if _, ok := tk.Method.(*stdjwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %valext", tk.Header["alg"])
		}
		return []byte(accessSecret), nil
	})

	if err == nil {
		return tok, nil
	}

	if isTokenExpiredError(err) {
		return nil, ErrTokenExpired
	}
	return nil, ErrUnauthorized
}

func isTokenExpiredError(err error) bool {
	if err == nil {
		return false
	}
	switch e := err.(type) {
	case *stdjwt.ValidationError:
		switch e.Errors {
		case stdjwt.ValidationErrorExpired:
			return true
		}
	}
	return false
}

type machineIDKey struct{}

// ContextWithMachineID returns a new `context.Context` that holds a machine id
func ContextWithMachineID(ctx context.Context, machineID string) context.Context {
	return context.WithValue(ctx, machineIDKey{}, machineID)
}

// MachineIDFromContext returns the `machine id` previously associated with `ctx`, or
// `empty string if no such `machine id` could be found.
func MachineIDFromContext(ctx context.Context) string {
	val := ctx.Value(machineIDKey{})
	if machineID, ok := val.(string); ok {
		return machineID
	}
	return ""
}

type authKeyIDKey struct{}

// ContextWithAuthKeyID returns a new `context.Context` that holds a auth key's id
func ContextWithAuthKeyID(ctx context.Context, keyID uint64) context.Context {
	return context.WithValue(ctx, authKeyIDKey{}, keyID)
}

// AuthKeyIDFromContext returns the `auth key's id` previously associated with `ctx`,or
// `0` if no such `key id` could be found.
func AuthKeyIDFromContext(ctx context.Context) uint64 {
	val := ctx.Value(authKeyIDKey{})
	if id, ok := val.(uint64); ok {
		return id
	}
	return 0
}

type userIDKey struct{}

// ContextWithUserID returns a new `context.Context` that holds a uid
func ContextWithUserID(ctx context.Context, userID uint64) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

// UserIDFromContext returns the `uid` previously associated with `ctx`, or
// `0` if no such `uid` could be found.
func UserIDFromContext(ctx context.Context) uint64 {
	val := ctx.Value(userIDKey{})
	if uid, ok := val.(uint64); ok {
		return uid
	}
	return 0
}
