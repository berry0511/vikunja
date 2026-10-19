// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package migration

import (
	"time"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

// Only a new table is synced. Never Sync a partial schema onto existing tables.
type worklogs20261019163000 struct {
	ID        int64     `xorm:"bigint autoincr not null unique pk"`
	UserID    int64     `xorm:"bigint not null index"`
	ProjectID int64     `xorm:"bigint not null index"`
	TaskID    int64     `xorm:"bigint not null default 0 index"`
	WorkDate  time.Time `xorm:"not null index"`
	Minutes   int       `xorm:"int not null"`
	Comment   string    `xorm:"text"`
	Created   time.Time `xorm:"created not null"`
	Updated   time.Time `xorm:"updated not null"`
}

func (worklogs20261019163000) TableName() string { return "worklogs" }

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20261019163000",
		Description: "Add separate daily project worklogs for manual time recording",
		Migrate: func(tx *xorm.Engine) error {
			return tx.Sync2(worklogs20261019163000{})
		},
		Rollback: func(tx *xorm.Engine) error {
			return tx.DropTables(worklogs20261019163000{})
		},
	})
}
