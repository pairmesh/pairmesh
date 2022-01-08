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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestFromPath(t *testing.T) {
	dir := t.TempDir()
	fileName := fmt.Sprintf("%s.toml", uuid.New().String())
	path := filepath.Join(dir, fileName)
	data := []byte(`
name = "1a"
region = "1"
host = "127.0.0.1"
port = 2328
stun_port = 3478

[portal]
key = "my-testing-relay"
url = "http://127.0.0.1:2823"
`)
	err := ioutil.WriteFile(path, data, os.ModePerm)
	assert.Nil(t, err)

	cfg, err := FromPath(path)
	assert.Nil(t, err)

	assert.NotNil(t, cfg.DHKey.Public)
	assert.NotNil(t, cfg.DHKey.Private)

	data, err = ioutil.ReadFile(path)
	assert.Nil(t, err)

	assert.Contains(t, string(data), "dh_key")

	// Double check
	cfg2, err := FromPath(path)
	assert.Nil(t, err)
	assert.Equal(t, cfg.DHKey.Public, cfg2.DHKey.Public)
	assert.Equal(t, cfg.DHKey.Private, cfg2.DHKey.Private)
}
