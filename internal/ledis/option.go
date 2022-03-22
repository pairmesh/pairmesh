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

package ledis

import "time"

// Option represents a handler for adjust the default option(s)
type Option func(*options)

type options struct {
	timeout  time.Duration
	password string
	db       int
}

// Timeout returns the option which changes the default dial timeout
func Timeout(seconds uint32) Option {
	return func(o *options) {
		if seconds == 0 {
			return
		}
		o.timeout = time.Duration(seconds) * time.Second
	}
}

// Password returns the option which sets the password for auth
func Password(p string) Option {
	return func(o *options) {
		if p == "" {
			return
		}
		o.password = p
	}
}

// DB returns the option which sets the current working DB
func DB(db int) Option {
	return func(o *options) {
		if db >= 16 {
			db = 0
		}
		o.db = db

	}
}
