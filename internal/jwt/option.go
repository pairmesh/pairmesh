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

import "time"

// AccessDetails represents a helper object for manage jwt token
type AccessDetails struct {
	UserID     uint64
	AccessUUID string
	MachineID  string
	AuthKeyID  uint64
}

// TokenDetails represents a full object about the  jwt token
type TokenDetails struct {
	AccessToken   string
	RefreshToken  string
	AccessUUID    string
	RefreshUUID   string
	AccessExpiry  int64
	RefreshExpiry int64
}

// Option represents a handler for adjust the default option(s)
type Option func(*options)

type options struct {
	store           Store
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration

	accessSecret  string
	refreshSecret string
}

func newOptions(opts ...Option) options {
	opt := options{
		accessTokenTTL:  time.Minute * 1500,
		refreshTokenTTL: time.Hour * 24 * 7,
		store: memoryStore{
			m: map[string]interface{}{},
		},
	}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

// AccessTokenTTL adjust the access token's ttl in seconds
func AccessTokenTTL(seconds uint32) Option {
	return func(o *options) {
		if seconds == 0 {
			return
		}
		o.accessTokenTTL = time.Duration(seconds) * time.Second
	}
}

// RefreshTokenTTL adjust the refresh token's ttl in seconds
func RefreshTokenTTL(seconds uint32) Option {
	return func(o *options) {
		if seconds == 0 {
			return
		}
		o.refreshTokenTTL = time.Duration(seconds) * time.Second
	}
}

// WithStore adjust the default store for jwt token
func WithStore(s Store) Option {
	return func(o *options) {
		if s == nil {
			return
		}
		o.store = s
	}
}
