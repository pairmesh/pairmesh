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

package redis

import (
	"context"
	"errors"
	"time"

	stdredis "github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// ErrIllegalParameter represents the input parameter(s) is illegal
var ErrIllegalParameter = errors.New("illegal parameter")

var r *redis

// Initialize initializes the redis
func Initialize(conn string, opts ...Option) error {
	options := newOptions(opts...)
	r = &redis{
		opts: options,
	}

	r.init(conn)
	zap.L().Info("initialize the redis successfully")
	return nil
}

type redis struct {
	opts options
	cli  *stdredis.Client
}

func (r *redis) init(conn string) error {
	if conn == "" {
		return ErrIllegalParameter
	}

	zap.L().Info("init redis with normal mode")
	r.initSingle(conn)

	_, err := r.cli.Ping(context.Background()).Result()
	if err != nil {
		zap.L().Error("ping failed", zap.Error(err))
	}
	return err
}

func (r *redis) initSingle(conn string) {
	r.cli = stdredis.NewClient(&stdredis.Options{
		Addr:        conn,
		DB:          r.opts.db,
		Password:    r.opts.password,
		DialTimeout: r.opts.timeout,
	})
}

func (r *redis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.cli.Set(context.Background(), key, value, expiration).Err()
}
func (r *redis) Del(ctx context.Context, keys ...string) error {
	return r.cli.Del(ctx, keys...).Err()
}

func (r *redis) Get(ctx context.Context, key string) (interface{}, error) {
	return r.cli.Get(ctx, key).Result()
}

// Shared represents the global redis object
func Shared() *redis {
	return r
}
