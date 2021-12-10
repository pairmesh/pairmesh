// Copyright 2021 PairMesh.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package resources

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"

	"github.com/lxn/walk"
)

//go:embed logo_windows.png
var logo []byte

//go:embed disabled_logo_windows.png
var disabledLogo []byte

var (
	Logo         walk.Image
	LogoDisabled walk.Image
)

func init() {
	Logo = mustLoadIcon(logo)
	LogoDisabled = mustLoadIcon(disabledLogo)
}

func mustLoadIcon(b []byte) walk.Image {
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	ret, err := walk.NewIconFromImageForDPI(img, 96) // NEW
	if err != nil {
		panic(err)
	}
	return ret
}
