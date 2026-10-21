// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
// Licensed under the GNU Affero General Public License version 3 or later.

package migration

import (
 "fmt"
 "code.vikunja.io/api/pkg/log"
 "time"
 "src.techknowlogick.com/xormigrate"
 "xorm.io/xorm"
)

// Backfill instances that ran the earlier catalog migration before the first
// administrator existed. All labels are created by the system, not by a user.
func init() {
 migrations = append(migrations, &xormigrate.Migration{
  ID: "20261021190000",
  Description: "Backfill immutable default label catalog at database initialization",
  Migrate: func(tx *xorm.Engine) error {
   log.Infof("migration.default_labels starting id=20261021190000")
   if _, err := tx.Exec("CREATE TABLE IF NOT EXISTS global_labels (label_id BIGINT PRIMARY KEY)"); err != nil { return err }
   type idRow struct { ID int64 `xorm:"'id'"` }
   for _, item := range []struct{ title, color string }{
    {"Bug", "d73a49"}, {"需求", "0969da"}, {"优化", "1a7f37"}, {"测试", "8250df"}, {"文档", "bf8700"},
   } {
    existing := idRow{}
    found, err := tx.SQL("SELECT id FROM labels WHERE title = ? ORDER BY id LIMIT 1", item.title).Get(&existing)
    if err != nil { log.Errorf("migration.default_labels query failed title=%q err=%v", item.title, err); return err }
    if !found {
     now := time.Now().UTC()
     if _, err = tx.Exec("INSERT INTO labels (title, description, hex_color, created_by_id, created, updated) VALUES (?, ?, ?, ?, ?, ?)", item.title, "", item.color, int64(0), now, now); err != nil { return err }
     found, err = tx.SQL("SELECT id FROM labels WHERE title = ? ORDER BY id LIMIT 1", item.title).Get(&existing)
     if err != nil { return err }
     if !found { return fmt.Errorf("seeded label %q not found", item.title) }
    }
    row := idRow{}
    registered, err := tx.SQL("SELECT label_id AS id FROM global_labels WHERE label_id = ?", existing.ID).Get(&row)
    if err != nil { return err }
    if !registered {
     if _, err = tx.Exec("INSERT INTO global_labels (label_id) VALUES (?)", existing.ID); err != nil { return err }
    }
   }
   log.Infof("migration.default_labels finished id=20261021190000 count=5")
   return nil
  },
  Rollback: func(tx *xorm.Engine) error { return nil },
 })
}
