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

// DetermRng is a deterministic "random" number generator that implements Read(),
// so it works as mock io.Reader for generating credentials
type DetermRng struct {
	counter int
}

// NewDetermRng returns DetermRng struct with counter initiated
func NewDetermRng() *DetermRng {
	return &DetermRng{
		counter: 0,
	}
}

// Read is the implementation of the io.Reader interface. It generates
// deterministic results, only based on its counter
func (rng *DetermRng) Read(p []byte) (n int, err error) {
	for i := 0; i < len(p); i++ {
		p[i] = byte((i + rng.counter) % 256)
	}
	rng.counter++
	return len(p), nil
}
