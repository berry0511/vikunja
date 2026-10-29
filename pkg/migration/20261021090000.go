// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package migration

import (
	"fmt"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID: "20261021090000",
		Description: "Add optional project start, end and due dates",
		Migrate: func(tx *xorm.Engine) error {
			quote := tx.Dialect().Quoter().Quote
			for _, col := range []string{"start_date", "end_date", "due_date"} {
				stmt := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s TIMESTAMP NULL", quote("projects"), quote(col))
				if _, err := tx.Exec(stmt); err != nil {
					return fmt.Errorf("add project planning date %s: %w", col, err)
				}
			}
			return nil
		},
		Rollback: func(tx *xorm.Engine) error { return nil },
	})
}
