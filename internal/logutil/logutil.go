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

package logutil

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// InitLogger initialize the global logger
func InitLogger() {
	// Preflight the logger
	config := zap.NewDevelopmentEncoderConfig()
	encoder := zapcore.NewConsoleEncoder(config)
	logger := zap.New(zapcore.NewCore(encoder, os.Stdout, Level()))
	zap.ReplaceGlobals(logger)
}
