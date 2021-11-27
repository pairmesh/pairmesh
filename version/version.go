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

package version

import (
	"fmt"
	"runtime"
)

var (
	// MajorVersion is the major version of UnifyShare
	MajorVersion int64 = 0
	// MinorVersion is the minor version of UnifyShare
	MinorVersion int64 = 1
	// PatchVersion is the patch version of UnifyShare
	PatchVersion int64 = 0
	// GitHash is the current git commit hash
	GitHash = "Unknown"
	// GitBranch is the current git branch name
	GitBranch = "Unknown"
)

// Version is the semver of TiOps
type Version struct {
	major     int64
	minor     int64
	patch     int64
	gitBranch string
	gitHash   string
	goVersion string
}

// NewVersion creates a Version object
func NewVersion() *Version {
	return &Version{
		major:     MajorVersion,
		minor:     MinorVersion,
		patch:     PatchVersion,
		gitBranch: GitBranch,
		gitHash:   GitHash,
		goVersion: runtime.Version(),
	}
}

// SemVer returns Version in semver format
func (v *Version) SemVer() string {
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
}

// String converts Version to a string
func (v *Version) String() string {
	return v.SemVer()
}

// FullInfo returns full version and build info
func (v *Version) FullInfo() string {
	return fmt.Sprintf("%s (%s/%s) %s",
		v.String(),
		v.gitBranch,
		v.gitHash,
		v.goVersion)
}
