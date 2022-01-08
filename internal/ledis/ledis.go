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

import (
	"context"
	"errors"
	"fmt"
	"time"

	lediscfg "github.com/ledisdb/ledisdb/config"
	"github.com/ledisdb/ledisdb/ledis"
	"go.uber.org/zap"
)

// ErrIllegalParameter represents the input parameter(s) is illegal
var ErrIllegalParameter = errors.New("illegal parameter")

var shared *ledisStore

type ledisStore struct {
	l *ledis.DB
}

// Initialize initializes the ledis cache.
func Initialize(dataPath string) error {
	cfg := lediscfg.NewConfigDefault()
	cfg.DataDir = dataPath
	l, err := ledis.Open(cfg)
	if err != nil {
		return err
	}

	db, err := l.Select(0)
	if err != nil {
		return err
	}

	shared = &ledisStore{db}

	zap.L().Info("initialize the ledis successfully")
	return nil
}

func (r *ledisStore) Set(_ context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.l.SetEX([]byte(key), int64(expiration.Seconds()), []byte(fmt.Sprintf("%v", value)))
}
func (r *ledisStore) Del(_ context.Context, keys ...string) error {
	var kb [][]byte
	for _, k := range keys {
		kb = append(kb, []byte(k))
	}
	_, err := r.l.Del(kb...)
	return err
}

func (r *ledisStore) Get(_ context.Context, key string) (interface{}, error) {
	res, err := r.l.Get([]byte(key))
	return string(res), err
}

// Shared represents the global redis object
func Shared() *ledisStore {
	return shared
}
