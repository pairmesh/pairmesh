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
	"database/sql"
	"errors"
	"fmt"

	"github.com/pairmesh/pairmesh/portal/config"
	"github.com/pairmesh/pairmesh/portal/db/models"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var initialized = atomic.Bool{}
var globalDB *gorm.DB

// Initialize initialize the database
func Initialize(cfg *config.MySQL) error {
	if initialized.Swap(true) {
		return errors.New("initialize twice")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=true&loc=Local", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DB)
	db, err := gorm.Open(mysql.New(mysql.Config{DSN: dsn}), &gorm.Config{Logger: &Logger{}})
	if err != nil {
		return err
	}

	allTables := []interface{}{
		&models.User{},
		&models.AuthKey{},
		&models.NetworkUser{},
		&models.Invitation{},
		&models.Network{},
		&models.Device{},
		&models.RelayServer{},
		&models.GithubUser{},
		&models.WechatUser{},
	}

	// Create table if not exists
	err = db.AutoMigrate(allTables...)
	if err != nil {
		return err
	}

	db.Logger.LogMode(logger.Info)

	// The global database instance.
	globalDB = db

	zap.L().Info("Preflight the database successfully",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("user", cfg.User),
		zap.String("db", cfg.DB))

	return nil
}

// Tx start a transaction as a block, return error will rollback, otherwise to commit.
func Tx(fc func(tx *gorm.DB) error, opts ...*sql.TxOptions) error {
	return globalDB.Transaction(fc, opts...)
}

// Create inserts a record into the database
func Create(bean interface{}) error {
	return globalDB.Transaction(func(tx *gorm.DB) error {
		tx.Create(bean)
		return tx.Error
	})
}
