// Copyright 2022 PairMesh, Inc.
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

package server

import (
	"errors"

	"github.com/pairmesh/pairmesh/bench"
	"github.com/pairmesh/pairmesh/bench/config"
)

// Job is the interface of a server, which could be started with Start function
type Job interface {
	Start() error
}

// Run function starts a simple echo server as server side of pairbench
// If cfg.isBounce is true, it echoes whatever it hears from incoming connection
// Otherwise, it always echoes "OK" as minimal backward payload
func Run(cfg *config.ServerConfig) error {
	var job Job
	switch {
	case cfg.Mode() == bench.ModeTypeEcho:
		job = NewEchoServer(cfg)
		return job.Start()
	case cfg.Mode() == bench.ModeTypeRelay:
		job = NewRelayServer(cfg)
		return job.Start()
	default:
		return errors.New("invalid mode specified when starting a server (supported mode: echo/relay)")
	}
}
