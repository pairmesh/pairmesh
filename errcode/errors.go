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

package errcode

import (
	"errors"
	"net/http"

	"github.com/pingcap/fn"
)

var (
	ErrVersionMissing       = withcode(errors.New("version header is missing"), InvalidVersion)
	ErrMajorVersionMismatch = withcode(errors.New("major version mismatch"), IncompatibleVersion)
	ErrInvalidAuthKey       = fn.ErrorWithStatusCode(withcode(errors.New("invalid authentication privateKey"), InvalidSecretKey), http.StatusUnauthorized)
	ErrInvalidToken         = fn.ErrorWithStatusCode(withcode(errors.New("invalid token"), InvalidToken), http.StatusForbidden)
	ErrServerInternal       = withcode(errors.New("server internal error"), InternalError)
	ErrNotFound             = withcode(errors.New("not found"), NotFound)
	ErrIllegalRequest       = withcode(errors.New("illegal request"), IllegalRequest)
	ErrIllegalOperation     = withcode(errors.New("illegal operation"), IllegalOperation)
	ErrDeviceExceed         = withcode(errors.New("device exceed"), DeviceExceed)
)

// Error represent a dedicated error type, which contain the API status code
type Error struct {
	Code ErrCode
	Err  error
}

// Error implements the error interface
func (e Error) Error() string {
	return e.Err.Error()
}

func (e Error) Unwrap() error {
	return e.Err
}

// ErrorWithCode returns a error with the specified error message and code
func withcode(err error, code ErrCode) error {
	return Error{
		Code: code,
		Err:  err,
	}
}
