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

package main

import (
	"os"
	"path/filepath"
	"time"

	ps "github.com/mitchellh/go-ps"
	"github.com/pairmesh/pairmesh/internal/messagebox"
)

func main() {
	exec, err := os.Executable()
	if err != nil {
		messagebox.Fatal("Cannot found executable path", err.Error())
	}
	dir := filepath.Dir(exec)
	path, err := filepath.Abs(filepath.Join(dir, "../Daemon/PairMesh"))
	if err != nil {
		messagebox.Fatal("Daemon executable path not found", err.Error())
	}

	go runNativeApp(path)

	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		pl, err := ps.Processes()
		if err != nil {
			messagebox.Fatal("List processes failed", err.Error())
		}
		for _, p := range pl {
			if p.Executable() == "PairMesh" {
				os.Exit(0)
			}
		}
	}
}
