// Copyright 2022 PairMesh, Inc.
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

package utils

type DetermRng struct {
	counter int
}

func NewDetermRng() *DetermRng {
	return &DetermRng{
		counter: 0,
	}
}

func (rng *DetermRng) Read(p []byte) (n int, err error) {
	for i := 0; i < len(p); i++ {
		p[i] = byte((i + rng.counter) % 256)
	}
	rng.counter++
	return len(p), nil
}
