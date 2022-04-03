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

import "github.com/pairmesh/pairmesh/benchmark"

// ClientConfig is the config struct for client side pairbench
type ClientConfig struct {
	mode     benchmark.ModeType
	endpoint string
	port     uint16
	clients  uint16
	payload  uint32
	duration uint16
}

// NewConfig function returns ClientConfig struct with parameters given
func NewClientConfig(
	mode benchmark.ModeType,
	endpoint string,
	port uint16,
	clients uint16,
	payload uint32,
	duration uint16,
) ClientConfig {
	return ClientConfig{
		mode:     mode,
		endpoint: endpoint,
		port:     port,
		clients:  clients,
		payload:  payload,
		duration: duration,
	}
}

// Mode returns the mode of the server to connect to
func (c *ClientConfig) Mode() benchmark.ModeType {
	return c.mode
}

// Endpoint returns the endpoint of the server to connect to
func (c *ClientConfig) Endpoint() string {
	return c.endpoint
}

// Port returns the port of the server to connect to
func (c *ClientConfig) Port() uint16 {
	return c.port
}

// Clients returns the number of client workers to spawn
func (c *ClientConfig) Clients() uint16 {
	return c.clients
}

// Payload returns the size of message payload data
func (c *ClientConfig) Payload() uint32 {
	return c.payload
}

// Duration returns the duration of the test case
func (c *ClientConfig) Duration() uint16 {
	return c.duration
}
