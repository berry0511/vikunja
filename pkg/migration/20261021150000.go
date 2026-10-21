// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
// Licensed under the GNU Affero General Public License as published by the Free Software Foundation,
// either version 3 of the License, or (at your option) any later version.
package migration

import (
	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

// Merge only duplicate Bug labels of the SAME owner. Labels with different
// owners must retain their access semantics and may legitimately share a title.
func consolidateDuplicateBugLabels20261021150000(tx *xorm.Engine) error {
	type labelRow struct {
		ID int64 `xorm:"'id'"`
		CreatedByID int64 `xorm:"'created_by_id'"`
	}
	rows := []labelRow{}
	if err := tx.SQL("SELECT id, created_by_id FROM labels WHERE title = ? ORDER BY created_by_id, id", "Bug").Find(&rows); err != nil {
		return err
	}
	canonical := map[int64]int64{}
	for _, row := range rows {
		keep, exists := canonical[row.CreatedByID]
		if !exists {
			canonical[row.CreatedByID] = row.ID
			continue
		}
		// Preserve every task association. Do not create a second relation for
		// tasks that already have the canonical label.
		if _, err := tx.Exec("UPDATE label_tasks SET label_id = ? WHERE label_id = ? AND NOT EXISTS (SELECT 1 FROM (SELECT task_id FROM label_tasks WHERE label_id = ?) existing WHERE existing.task_id = label_tasks.task_id)", keep, row.ID, keep); err != nil {
			return err
		}
		if _, err := tx.Exec("DELETE FROM label_tasks WHERE label_id = ?", row.ID); err != nil {
			return err
		}
		if _, err := tx.Exec("DELETE FROM labels WHERE id = ?", row.ID); err != nil {
			return err
		}
	}
	return nil
}
func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID: "20261021150000",
		Description: "Consolidate same-owner duplicate Bug labels and preserve task links",
		Migrate: consolidateDuplicateBugLabels20261021150000,
		Rollback: func(tx *xorm.Engine) error { return nil },
	})
}
