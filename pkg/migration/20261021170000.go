// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
// Licensed under the GNU Affero General Public License as published by the Free Software Foundation,
// either version 3 of the License, or (at your option) any later version.

package migration

import (
	"fmt"
	"time"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

// Public label catalog. Existing task->label relations and label ownership are
// deliberately preserved. Only one canonical existing label per title is shared.
func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID: "20261021170000",
		Description: "Seed organization-wide labels without replacing task label references",
		Migrate: func(tx *xorm.Engine) error {
			if _, err := tx.Exec("CREATE TABLE global_labels (label_id BIGINT PRIMARY KEY)"); err != nil {
				return fmt.Errorf("create public label catalog: %w", err)
			}
			type idRow struct { ID int64 `xorm:"'id'"` }
			catalog := []struct {title, color string}{
				{"Bug", "d73a49"},
				{"需求", "0969da"},
				{"优化", "1a7f37"},
				{"测试", "8250df"},
				{"文档", "bf8700"},
			}
			for _, entry := range catalog {
				label := idRow{}
				found, err := tx.SQL("SELECT id FROM labels WHERE title = ? ORDER BY id LIMIT 1", entry.title).Get(&label)
				if err != nil { return err }
				if !found {
					now := time.Now().UTC()
					if _, err := tx.Exec("INSERT INTO labels (title, description, hex_color, created_by_id, created, updated) VALUES (?, ?, ?, ?, ?, ?)",
						entry.title, "", entry.color, int64(0), now, now); err != nil { return err }
					found, err = tx.SQL("SELECT id FROM labels WHERE title = ? ORDER BY id LIMIT 1", entry.title).Get(&label)
					if err != nil { return err }
					if !found { return fmt.Errorf("cannot find new shared label %q", entry.title) }
				}
				if _, err := tx.Exec("INSERT INTO global_labels (label_id) VALUES (?)", label.ID); err != nil { return err }
			}
			return nil
		},
		Rollback: func(tx *xorm.Engine) error { return nil },
	})
}
