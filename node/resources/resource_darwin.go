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
	_ "embed"

	"github.com/progrium/macdriver/cocoa"
	"github.com/progrium/macdriver/core"
)

//go:embed logo_darwin.png
var logo []byte

//go:embed disabled_logo_darwin.png
var disabledLogo []byte

//go:embed alert_logo_darwin.png
var alertLogo []byte

var (
	Logo         cocoa.NSImage
	DisabledLogo cocoa.NSImage
	AlertLogo    cocoa.NSImage
)

func init() {
	Logo = cocoa.NSImage_InitWithData(core.NSData_WithBytes(logo, uint64(len(logo))))
	DisabledLogo = cocoa.NSImage_InitWithData(core.NSData_WithBytes(disabledLogo, uint64(len(disabledLogo))))
	AlertLogo = cocoa.NSImage_InitWithData(core.NSData_WithBytes(alertLogo, uint64(len(alertLogo))))
}
