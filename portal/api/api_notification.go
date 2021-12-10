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

package api

import (
	"github.com/pairmesh/pairmesh/portal/db"
	"github.com/pairmesh/pairmesh/portal/db/models"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type (
	Notification struct {
		ID             models.ID                 `json:"id"`
		Title          string                    `json:"title"`
		Content        string                    `json:"content"`
		Link           string                    `json:"link"`
		EmergencyLevel models.EmergencyLevelType `json:"emergency_level"`
	}
	NotificationResponse struct {
		Notifications []Notification `json:"notifications"`
	}
)

//Notifications system notification for every user
func (s *server) Notifications() (*NotificationResponse, error) {
	res := &NotificationResponse{}
	err := db.Tx(func(tx *gorm.DB) error {
		var notifications []models.Notification
		err := models.NewNotificationQuerySet(tx).OrderAscByEmergencyLevel().All(&notifications)
		if err != nil && err != gorm.ErrRecordNotFound {
			return errors.WithStack(err)
		}
		for _, notification := range notifications {
			res.Notifications = append(res.Notifications, Notification{
				ID:             notification.ID,
				Title:          notification.Title,
				Content:        notification.Content,
				Link:           notification.Link,
				EmergencyLevel: notification.EmergencyLevel,
			})
		}
		return nil
	})

	return res, err
}
