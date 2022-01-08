//go:build darwin && !go1.11
// +build darwin,!go1.11

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

package tun

func setNonBlock(fd int) error {
	// There's a but pre-go1.11 that causes 'resource temporarily unavailable'
	// error in non-blocking mode. So just skip it here. Close() won't be able
	// to unblock a pending read, but that's better than being broken.
	return nil
}
