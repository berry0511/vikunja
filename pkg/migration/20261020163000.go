// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
// Licensed under AGPL-3.0-or-later.

package migration

import (
    "src.techknowlogick.com/xormigrate"
    "xorm.io/xorm"
)

// Normalize legacy classification labels. Ordinary work remains an ordinary task.
// Keep the existing defect label IDs so all existing Bug links continue working.
func init() {
    migrations = append(migrations, &xormigrate.Migration{
        ID: "20261020163000",
        Description: "Normalize Bug label and retire obsolete work-order label",
        Migrate: func(tx *xorm.Engine) error {
            if _, err := tx.Exec("UPDATE labels SET title = ? WHERE title IN (?, ?)", "Bug", "工单·缺陷", "工单-缺陷"); err != nil {
                return err
            }
            // Remove the now-obsolete classification from tasks before deleting its label.
            if _, err := tx.Exec("DELETE FROM label_tasks WHERE label_id IN (SELECT id FROM labels WHERE title IN (?, ?, ?))",
                "工单·任务", "工单-任务", "工作工单"); err != nil {
                return err
            }
            _, err := tx.Exec("DELETE FROM labels WHERE title IN (?, ?, ?)",
                "工单·任务", "工单-任务", "工作工单")
            return err
        },
        Rollback: func(tx *xorm.Engine) error { return nil },
    })
}
