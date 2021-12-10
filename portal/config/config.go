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

	"github.com/BurntSushi/toml"
)

// Config represents the configuration of the gateway server
type Config struct {
	// Basic configurations
	Host       string `toml:"host"`
	Port       int    `toml:"port"`
	TLSKey     string `toml:"tls_key"`
	TLSCert    string `toml:"tls_cert"`
	PrivateKey string `toml:"private_key"`

	Relay *Relay `toml:"relay"`
	MySQL *MySQL `toml:"mysql"`
	JWT   *JWT   `toml:"jwt"`
	Redis *Redis `toml:"redis"`
	SSO   *SSO   `toml:"sso"`
	Data  *Data  `toml:"data"`
}

type Relay struct {
	AuthKey string `toml:"auth_key"`
}

// GitHub represents the provider:github's configuration
type GitHub struct {
	ClientID     string `toml:"clientID"`
	ClientSecret string `toml:"clientSecret"`
}

// WeChat represents the provider:wechat's configuration
type WeChat struct {
	ClientID                string `toml:"clientID"`
	ClientSecret            string `toml:"clientSecret"`
	MchId                   string `toml:"mchid"`
	ServiceNo               string `toml:"serviceNo"`
	SerialNo                string `toml:"serialNo"`
	ApiV3Key                string `toml:"apiV3Key"`
	ApiClientPrivateKeyPath string `toml:"apiClientPrivateKeyPath"`
}

// SSO represents the sso provider(s) configuration
type SSO struct {
	Redirect     string `toml:"redirect"`
	GitHub       GitHub `toml:"github"`
	WeChat       WeChat `toml:"wechat"`
	WeChatMobile WeChat `toml:"wechat_mobile"`
}

// MySQL represents the mysql connection configuration
type MySQL struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	DB       string `toml:"db"`
}

// JWT represents the jwt configuration
type JWT struct {
	AccessSecret    string `toml:"accessSecret"`
	RefreshSecret   string `toml:"refreshSecret"`
	AccessTokenTTL  uint32 `toml:"accessTokenTtl"`
	RefreshTokenTTL uint32 `toml:"refreshTokenTtl"`
}

// Redis represents the redis configuration
type Redis struct {
	DB       int    `toml:"db"`
	Conn     string `toml:"conn"`
	Timeout  uint32 `toml:"timeout"`
	Password string `toml:"password"`
}

// Data represents the data configuration
type Data struct {
	IP2LocationDBPath string `toml:"ip2LocationDBPath"`
}

// New returns a config instance with default value
func New() *Config {
	return &Config{
		Host:    "0.0.0.0",
		Port:    2823,
		TLSKey:  "",
		TLSCert: "",
		SSO: &SSO{
			Redirect: "http://127.0.0.1:2823",
		},
		MySQL: &MySQL{
			Host:     "127.0.0.1",
			Port:     3306,
			User:     "root",
			Password: "",
			DB:       "meshportal",
		},
		JWT: &JWT{
			AccessSecret:    "the_access_secret",
			RefreshSecret:   "the_refresh_secret",
			AccessTokenTTL:  1000000,
			RefreshTokenTTL: 1000000,
		},
		Redis: &Redis{
			Conn:     "127.0.0.1:6379",
			Timeout:  3000,
			Password: "",
			DB:       0,
		},
		Data: &Data{
			IP2LocationDBPath: "",
		},
	}
}

// FromReader returns the configuration instance from reader
func FromReader(reader io.Reader) (*Config, error) {
	config := New()
	_, err := toml.NewDecoder(reader).Decode(config)
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
