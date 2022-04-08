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
	"io"
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

// Config represents the configuration of the gateway server
type Config struct {
	// Basic configurations
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	TLSKey     string `yaml:"tlsKey"`
	TLSCert    string `yaml:"tlsCert"`
	PrivateKey string `yaml:"privateKey"`
	DataDir    string `yaml:"dataDir"`

	Relay *Relay `yaml:"relay"`
	MySQL *MySQL `yaml:"mysql"`
	JWT   *JWT   `yaml:"jwt"`
	SSO   *SSO   `yaml:"sso"`
}

// Relay represents a relay instance with its auth key
type Relay struct {
	AuthKey string `yaml:"authKey"`
}

// GitHub represents the provider:github's configuration
type GitHub struct {
	ClientID     string `yaml:"clientID"`
	ClientSecret string `yaml:"clientSecret"`
}

// SSO represents the sso provider(s) configuration
type SSO struct {
	Redirect string `yaml:"redirect"`
	GitHub   GitHub `yaml:"github"`
}

// MySQL represents the mysql connection configuration
type MySQL struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DB       string `yaml:"db"`
}

// JWT represents the jwt configuration
type JWT struct {
	AccessSecret    string `yaml:"accessSecret"`
	RefreshSecret   string `yaml:"refreshSecret"`
	AccessTokenTTL  uint32 `yaml:"accessTokenTtl"`
	RefreshTokenTTL uint32 `yaml:"refreshTokenTtl"`
}

// Data represents the data configuration
type Data struct {
	IP2LocationDBPath string `yaml:"locationDB"`
}

// New returns a config instance with default value
func New() *Config {
	return &Config{
		Host:    "0.0.0.0",
		Port:    2823,
		TLSKey:  "",
		TLSCert: "",
		DataDir: "./cache/",
		SSO: &SSO{
			Redirect: "http://127.0.0.1:2823",
		},
		MySQL: &MySQL{
			Host:     "127.0.0.1",
			Port:     3306,
			User:     "root",
			Password: "",
			DB:       "pairportal",
		},
		JWT: &JWT{
			AccessSecret:    "the_access_secret",
			RefreshSecret:   "the_refresh_secret",
			AccessTokenTTL:  1000000,
			RefreshTokenTTL: 1000000,
		},
	}
}

// FromReader returns the configuration instance from reader
func FromReader(reader io.Reader) (*Config, error) {
	config := New()
	err := yaml.NewDecoder(reader).Decode(config)
	if err != nil {
		return nil, err
	}
	return config, nil
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
	return FromBytes(data)
}
