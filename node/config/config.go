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
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/denisbrodbeck/machineid"
	"github.com/flynn/noise"
	"github.com/pairmesh/pairmesh/constant"
	"go.uber.org/zap"
)

const (
	configDirName  = "pairmesh"
	configFileName = "pairmesh.conf"
)

// configFilePath is used to customize the configuration file path.
var configFilePath string

// Config represents the current node's configuration
type Config struct {
	Token      string      `json:"token"`
	FastKey    string      `json:"fast_key"`
	DHKey      noise.DHKey `json:"dh_key"`
	Port       int         `json:"port"`
	MachineID  string      `json:"machine_id"`
	OnceAlert  bool        `json:"once_alert"`
	LocaleName string      `json:"locale_name"`
}

// SetConfigDir overrides the default configuration file path.
func SetConfigDir(dir string) {
	configFilePath = filepath.Join(dir, configFileName)
}

func (c *Config) path() string {
	if configFilePath != "" {
		return configFilePath
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join("./", configDirName, configFileName)
	}
	return filepath.Join(dir, configDirName, configFileName)
}

// Load loads the configuration from disk
func (c *Config) Load() error {
	path := c.path()
	r, err := os.Open(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	changed := false
	defer func() {
		if r != nil {
			_ = r.Close()
		}
	}()

	err = json.NewDecoder(r).Decode(c)
	if err != nil {
		zap.L().Error("Load configuration failed", zap.String("path", path), zap.Error(err))
		changed = true
	}

	// Check if previous port available.
	var portAvailable bool
	if c.Port != 0 {
		portAvailable = func() bool {
			conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: c.Port})
			if err != nil {
				return false
			}
			defer conn.Close()
			return true
		}()
	}

	if c.Port == 0 || !portAvailable {
		// Random select a free port to serve the current node
		port, err := func() (int, error) {
			conn, err := net.ListenUDP("udp", &net.UDPAddr{})
			if err != nil {
				return -1, err
			}
			defer conn.Close()
			return conn.LocalAddr().(*net.UDPAddr).Port, nil
		}()
		if err != nil {
			return err
		}

		changed = true
		c.Port = port
	}

	if len(c.DHKey.Public) != noise.DH25519.DHLen() {
		// Generate the static key for the current node.
		staticKey, err := noise.DH25519.GenerateKeypair(rand.Reader)
		if err != nil {
			return err
		}
		c.DHKey = staticKey
		zap.L().Info("Generate DHKey", zap.String("publicKey", base64.RawStdEncoding.EncodeToString(staticKey.Public)))

		changed = true
	}

	if c.MachineID == "" {
		machineID, err := machineid.ProtectedID(constant.MachineIDProtect)
		if err != nil {
			zap.L().Error("Retrieve machine id failed", zap.Error(err))
		}
		c.MachineID = machineID
	}

	if changed {
		err := c.Save()
		if err != nil {
			zap.L().Error("Save configuration failed", zap.Error(err))
		}
	}

	return nil
}

// Clone returns a clone of the config
func (c *Config) Clone() *Config {
	key := noise.DHKey{
		Public:  make([]byte, len(c.DHKey.Public)),
		Private: make([]byte, len(c.DHKey.Private)),
	}
	copy(key.Public, c.DHKey.Public)
	copy(key.Private, c.DHKey.Private)
	return &Config{
		Token:      c.Token,
		DHKey:      key,
		Port:       c.Port,
		MachineID:  c.MachineID,
		OnceAlert:  c.OnceAlert,
		LocaleName: c.LocaleName,
	}
}

// Save saves the configuration to disk
func (c *Config) Save() error {
	path := c.path()
	_, err := os.Create(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	zap.L().Info("Save the latest configuration", zap.String("path", path), zap.Reflect("config", c))

	if errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(filepath.Dir(path), os.ModePerm)
		if err != nil {
			return err
		}
	}

	w, err := os.Create(path)
	if err != nil {
		return err
	}
	defer w.Close()

	return json.NewEncoder(w).Encode(c)
}

// IsGuest checks the current user whether login into the PairMesh service or not
func (c *Config) IsGuest() bool {
	return c.Token == ""
}

// IsAuthKey checks if the token of the config is a valid auth key
func (c *Config) IsAuthKey() bool {
	return strings.HasPrefix(c.Token, constant.PrefixAuthKey)
}

// IsBearer checks if the token of the config has jwt token fomatted prefix
func (c *Config) IsBearer() bool {
	return strings.HasPrefix(c.Token, constant.PrefixJwtToken)
}

// SetJWTToken signs given token with jwt token as prefix
func (c *Config) SetJWTToken(tok string) error {
	c.Token = constant.PrefixJwtToken + " " + tok
	return c.Save()
}

// SetLocaleName sets locale name of the config
func (c *Config) SetLocaleName(name string) error {
	c.LocaleName = name
	return c.Save()
}
