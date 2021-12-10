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

package errcode

type ErrCode int

// NOTE: don't delete any item and resort the order.
const (
	InternalError ErrCode = 1 + iota
	InvalidVersion
	IncompatibleVersion
	InvalidSecretKey
	NotFound
	InvalidToken
	IllegalRequest
	IllegalOperation
	DeviceExceed
)

// NOTE: notify error to mobile platform, don't delete any item and resort the order.
const (
	//StartFailed start failed with reason which we have not classify
	StartFailed = 1000 + iota
	InvalidAccount
	ConfigLoadFailed
	EngineStartFailed
	CreateTunFailed
	TunUpdateFailed
	NetMapRefreshFailed
)
