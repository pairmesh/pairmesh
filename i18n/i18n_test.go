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

package i18n_test

import (
	"testing"

	"github.com/pairmesh/pairmesh/i18n"
	"github.com/stretchr/testify/assert"
)

func TestL(t *testing.T) {
	err := i18n.SetLocale("zh_CN")
	assert.Nil(t, err)

	s := i18n.L("tray.login")

	assert.Equal(t, s, "登入")
}
