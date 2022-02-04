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

package config

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/flynn/noise"
	"github.com/pairmesh/pairmesh/security"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Config represents the configuration of the relay server
type Config struct {
	// Name is a unique node name (across all regions).
	// It is not a host name.
	// It's typically of the form "1b", "2a", "3b", etc. (region
	// ID + suffix within that region)
	Name string `yaml:"name"`

	// Region is the Region of the RelayRegion that this node
	// is running in.
	Region string `yaml:"region"`

	// Host describes the host information about the relay server.
	Host string `yaml:"host,omitempty"`

	Port int `yaml:"port,omitempty"`

	// Port optionally specifies a STUN port to use.
	// Zero means 3478.
	// To disable STUN on this node, use -1.
	// https://datatracker.ietf.org/doc/html/rfc5389#section-18.4
	STUNPort int `yaml:"stunPort,omitempty"`

	DHKey  *security.DHKey `yaml:"dhKey"`
	Portal *Portal         `yaml:"portal"`
}

// Portal represents the gateway instance configuration
type Portal struct {
	Key string `yaml:"key"`
	URL string `yaml:"url"`
	// Keepalive interval between portal/relay services.
	KeepaliveInterval time.Duration `yaml:"keepaliveInterval,omitempty"`
	// Sync peer online status after a sync interval.
	SyncInterval time.Duration `yaml:"syncInterval,omitempty"`
}

// New returns a config instance with default value
func New() *Config {
	return &Config{
		Name:     "1a",
		Region:   "testing",
		Port:     2328,
		STUNPort: 3478,

		Portal: &Portal{
			Key:               "testing-relay-server",
			URL:               "http://127.0.0.1:2823",
			KeepaliveInterval: 5 * time.Minute,
			SyncInterval:      5 * time.Minute,
		},
	}
}

// FromReader returns the configuration instance from reader
func FromReader(reader io.Reader) (*Config, error) {
	c := New()
	err := yaml.NewDecoder(reader).Decode(c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// FromBytes returns the configuration instance from bytes
func FromBytes(data []byte) (*Config, error) {
	reader := bytes.NewBuffer(data)
	return FromReader(reader)
}

// FromPath returns the configuration instance from file path
func FromPath(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg, err := FromBytes(data)
	if err != nil {
		return nil, err
	}

	if cfg.DHKey == nil || cfg.DHKey.Public.Len() != noise.DH25519.DHLen() {
		// Generate the static key for the current node.
		staticKey, err := noise.DH25519.GenerateKeypair(rand.Reader)
		if err != nil {
			return nil, err
		}
		cfg.DHKey = security.FromNoiseDHKey(staticKey)
		zap.L().Info("Generate key", zap.String("publicKey", base64.RawStdEncoding.EncodeToString(staticKey.Public)))

		// Save to the configuration.
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, os.ModePerm)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		err = yaml.NewEncoder(file).Encode(cfg)
		if err != nil {
			return nil, err
		}
	}

	return cfg, nil
}
