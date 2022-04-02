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

package client

// Config is the config struct for client side pairbenchmark
type Config struct {
	endpoint string
	port     uint16
	clients  uint16
	payload  uint32
	duration uint16
}

func NewConfig(
	endpoint string,
	port uint16,
	clients uint16,
	payload uint32,
	duration uint16,
) Config {
	return Config{
		endpoint: endpoint,
		port:     port,
		clients:  clients,
		payload:  payload,
		duration: duration,
	}
}

func (c *Config) Endpoint() string {
	return c.endpoint
}

func (c *Config) Port() uint16 {
	return c.port
}

func (c *Config) Clients() uint16 {
	return c.clients
}

func (c *Config) Payload() uint32 {
	return c.payload
}

func (c *Config) Duration() uint16 {
	return c.duration
}
