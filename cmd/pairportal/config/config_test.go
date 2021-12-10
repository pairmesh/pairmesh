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

package config_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/pairmesh/pairmesh/cmd/pairportal/config"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	cfg := config.New()
	a := assert.New(t)
	a.NotNil(cfg)
	a.Equal(cfg.Port, 2823)
	a.Equal(cfg.TLSCert, "")
	a.Equal(cfg.MySQL.Port, 3306)
	a.Equal(cfg.MySQL.Password, "")
	a.Equal(cfg.MySQL.DB, "zetagateway")
}

func TestFromBytes(t *testing.T) {
	data := []byte(`
host: 0.0.0.0
port: 2824
tls-key: "/path/to/tls/key"
tls-cert: "/path/to/tls/cert"

mysql:
  host: 127.0.0.1
  port: 4000
  user: root
  password: "password"
  db: zetagateway
`)
	cfg, err := config.FromBytes(data)

	a := assert.New(t)
	a.Nil(err)
	a.Equal(cfg.TLSCert, "/path/to/tls/cert")
	a.Equal(cfg.MySQL.Port, 4000)
	a.Equal(cfg.MySQL.Password, "password")
	a.Equal(cfg.MySQL.DB, "zetagateway")
}

func TestFromPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), uuid.New().String())
	err := ioutil.WriteFile(path, []byte(`
host = "0.0.0.0"
port = 2823
tls_key = ""
tls_cert = "/path/to/tls/cert"
private_key = "/path/to/private-key"

[relay]
auth_key = "test"

[features]
disable_pay = true

[sso]
redirect = "http://192.168.0.101:8080"

[mysql]
host = "127.0.0.1"
port = 3306
user = "root"
password = "123456"
db = "pairportal"
`), os.ModePerm)

	a := assert.New(t)
	a.Nil(err)

	cfg, err := config.FromPath(path)
	a.Nil(err)
	a.Equal(cfg.TLSCert, "/path/to/tls/cert")
	a.Equal(cfg.PrivateKey, "/path/to/private-key")
	a.Equal(cfg.Relay.AuthKey, "test")
	a.Equal(cfg.MySQL.Port, 3306)
	a.Equal(cfg.MySQL.Password, "123456")
	a.Equal(cfg.MySQL.DB, "pairportal")
}
