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

package config

import "github.com/pairmesh/pairmesh/bench"

// ServerConfig is the config struct for server side pairbench
type ServerConfig struct {
	mode     bench.ModeType
	port     uint16
	isBounce bool
}

// NewServerConfig function returns a new ServerConfig with given parameters
func NewServerConfig(
	mode bench.ModeType,
	port uint16,
	isBounce bool,
) ServerConfig {
	return ServerConfig{
		mode:     mode,
		port:     port,
		isBounce: isBounce,
	}
}

// Mode return the mode of the server
func (c *ServerConfig) Mode() bench.ModeType {
	return c.mode
}

// Port returns the port of the server
func (c *ServerConfig) Port() uint16 {
	return c.port
}

// IsBounce returns whether the server bounces back incoming data,
// or just echoes OK as minimal payload message
func (c *ServerConfig) IsBounce() bool {
	return c.isBounce
}
