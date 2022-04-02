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

package server

// Config is the config struct for server side pairbenchmark
type Config struct {
	port     uint16
	isBounce bool
}

func NewConfig(port uint16, isBounce bool) Config {
	return Config{
		port:     port,
		isBounce: isBounce,
	}
}

func (c *Config) Port() uint16 {
	return c.port
}

func (c *Config) IsBounce() bool {
	return c.isBounce
}
