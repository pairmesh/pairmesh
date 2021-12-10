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
	"os"

	"github.com/pairmesh/pairmesh/constant"
)

// APIGateway returns the gateway address. The PAIRMESH_GATEWAY_API environment variable
// will be prioritized to be returned to the caller. And the default gateway address
// will be used if no customized address found.
func APIGateway() string {
	if g, found := os.LookupEnv("PAIRMESH_GATEWAY_API"); found {
		return g
	}
	return constant.DefaultAPIGateway
}

// MyGateway returns the gateway address. The PAIRMESH_GATEWAY_MY environment variable
// will be prioritized to be returned to the caller. And the default gateway address
// will be used if no customized address found.
func MyGateway() string {
	if g, found := os.LookupEnv("PAIRMESH_GATEWAY_MY"); found {
		return g
	}
	return constant.DefaultMyGateway
}
