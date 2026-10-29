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

// Create a standalone journal without foreign keys; deleting a task or
// project must never erase its history. Never Sync a partial model to
// alter existing task/project schemas.
type changeHistory20261020114500 struct {
	ID int64 `xorm:"bigint autoincr not null unique pk"`
	EntityType string `xorm:"varchar(16) not null index"`
	EntityID int64 `xorm:"bigint not null index"`
	ActorID int64 `xorm:"bigint not null"`
	Action string `xorm:"varchar(16) not null"`
	Field string `xorm:"varchar(64) not null"`
	OldValue string `xorm:"longtext"`
	NewValue string `xorm:"longtext"`
	Created time.Time `xorm:"created not null"`
}

func (changeHistory20261020114500) TableName() string { return "change_history" }

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID: "20261020114500",
		Description: "Add permanent read-only task and project change history",
		Migrate: func(tx *xorm.Engine) error { return tx.Sync2(changeHistory20261020114500{}) },
		Rollback: func(tx *xorm.Engine) error { return tx.DropTables(changeHistory20261020114500{}) },
	})
}
