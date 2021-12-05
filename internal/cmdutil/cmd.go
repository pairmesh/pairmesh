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

package cmdutil

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// Example represents an example of command line
type Example struct {
	Example string
	Comment string
}

// Examples represents a group of examples
type Examples []Example

// String implements the fmt.Stringer interface
func (es Examples) String() string {
	if len(es) == 0 {
		return ""
	}

	var max int
	for _, e := range es {
		if len(e.Example) > max {
			max = len(e.Example)
		}
	}

	var all []string
	for _, e := range es {
		all = append(all, fmt.Sprintf("  %s%s# %s", Highlight(e.Example), strings.Repeat(" ", max-len(e.Example)+3), e.Comment))
	}

	return strings.Join(all, "\n")
}

// Run starts a command
func Run(cmd *cobra.Command) {
	rand.Seed(time.Now().UnixNano())

	if err := cmd.Execute(); err != nil {
		zap.L().Error("Run command failed", zap.Error(err))
		os.Exit(1)
	}
}

// Bold converts the string to bolder display in the terminal
func Bold(f string, args ...interface{}) string {
	return color.New(color.Bold).Sprintf(f, args...)
}

// Underline add the underline display in the terminal for the string
func Underline(f string, args ...interface{}) string {
	return color.New(color.Bold, color.Underline).Sprintf(f, args...)
}

// Highlight highlights the string in the terminal
func Highlight(f string, args ...interface{}) string {
	return color.New(color.Bold, color.FgHiCyan).Sprintf(f, args...)
}
