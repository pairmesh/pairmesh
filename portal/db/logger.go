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

package db

import (
	"context"
	"fmt"
	"time"

	"github.com/pairmesh/pairmesh/pkg/logutil"
	"go.uber.org/zap"
	"gorm.io/gorm/logger"
)

// Logger is the customized logger for gorm
type Logger struct {
	level logger.LogLevel
}

// LogMode implements the logger.Interface interface
func (l *Logger) LogMode(level logger.LogLevel) logger.Interface {
	l.level = level
	return l
}

// Info implements the logger.Interface interface
func (l *Logger) Info(_ context.Context, format string, args ...interface{}) {
	if l.level < logger.Info {
		return
	}
	zap.L().Info(fmt.Sprintf(format, args...))
}

// Warn implements the logger.Interface interface
func (l *Logger) Warn(_ context.Context, format string, args ...interface{}) {
	if l.level < logger.Warn {
		return
	}
	zap.L().Warn(fmt.Sprintf(format, args...))
}

// Error implements the logger.Interface interface
func (l *Logger) Error(_ context.Context, format string, args ...interface{}) {
	if l.level < logger.Error {
		return
	}
	zap.L().Error(fmt.Sprintf(format, args...))
}

// Trace implements the logger.Interface interface
func (l *Logger) Trace(_ context.Context, _ time.Time, fc func() (string, int64), err error) {
	if logutil.IsEnablePortal() {
		sql, _ := fc()
		if err != nil {
			zap.L().Error("GORM Trace",
				zap.String("SQL", sql),
				zap.Error(err))
		} else {
			zap.L().Debug("GORM Trace",
				zap.String("SQL", sql),
				zap.Error(err))
		}
	}
}
