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

package systray

import "sync"

var globalRegistry = newRegistry()

// registry is used to store all menu items.
type registry struct {
	mu     sync.RWMutex
	items  map[uint32]*MenuItem
	events chan *MenuItem
}

// newRegistry returns a new registry instance.
func newRegistry() *registry {
	return &registry{
		items:  map[uint32]*MenuItem{},
		events: make(chan *MenuItem, 64),
	}
}

// Store stores the item to the registry and the old one will be overwrited.
func (r *registry) Store(item *MenuItem) {
	r.mu.Lock()
	r.items[item.id] = item
	r.mu.Unlock()
}

// MenuItem returns the MenuItem corresponding to the id.
func (r *registry) MenuItem(id uint32) *MenuItem {
	r.mu.RLock()
	item := r.items[id]
	r.mu.RUnlock()
	return item
}

// Events returns the events list.
func Events() <-chan *MenuItem {
	return globalRegistry.events
}
