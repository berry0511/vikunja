// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package migration

import (
	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

// This is an existing table. Add only the new column; do NOT run Sync2
// against a partial TaskAttachment struct, which could reshape old columns.
// All pre-existing attachments are task attachments.
type attachmentSource20261020120000 struct {
	ID int64 `xorm:"bigint autoincr not null unique pk"`
	Source string `xorm:"varchar(16) not null default 'task'"`
}
func (*attachmentSource20261020120000) TableName() string { return "task_attachments" }

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID: "20261020120000",
		Description: "Distinguish task files from comment-uploaded attachments",
		Migrate: func(tx *xorm.Engine) error {
			return tx.AddColumn(&attachmentSource20261020120000{}, "source")
		},
		// Deliberately no destructive rollback of recorded file origins.
		Rollback: func(tx *xorm.Engine) error { return nil },
	})
}
