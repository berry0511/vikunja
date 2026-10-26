// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
// Licensed under the GNU Affero General Public License version 3 or later.

package migration

import (
    "fmt"
    "time"

    "code.vikunja.io/api/pkg/db"
    "code.vikunja.io/api/pkg/log"

    "xorm.io/xorm"
)

// ensureSystemLabelCatalog runs during database startup, after normal migrations.
// It is idempotent and does not need the first administrator to exist. This
// also repairs installs whose migration ledger reports success while the
// global_labels table is absent.
func ensureSystemLabelCatalog(x *xorm.Engine) error {
    if x == nil {
        var err error
        x, err = db.CreateDBEngine()
        if err != nil {
            return fmt.Errorf("open database for system labels: %w", err)
        }
        defer x.Close()
    }

    session := x.NewSession()
    defer session.Close()
    if err := session.Begin(); err != nil {
        return fmt.Errorf("begin system label seed: %w", err)
    }
    defer session.Rollback()

    if _, err := session.Exec("CREATE TABLE IF NOT EXISTS global_labels (label_id BIGINT PRIMARY KEY)"); err != nil {
        return fmt.Errorf("create global_labels catalog: %w", err)
    }

    type idRow struct { ID int64 `xorm:"'id'"` }
    defaults := []struct{ title, color string }{
        {"Bug", "d73a49"},
        {"需求", "0969da"},
        {"优化", "1a7f37"},
        {"测试", "8250df"},
        {"文档", "bf8700"},
    }
    for _, item := range defaults {
        row := idRow{}
        found, err := session.SQL("SELECT id FROM labels WHERE title = ? ORDER BY id LIMIT 1", item.title).Get(&row)
        if err != nil {
            return fmt.Errorf("read default label %q: %w", item.title, err)
        }
        if !found {
            now := time.Now().UTC()
            _, err = session.Exec(
                "INSERT INTO labels (title, description, hex_color, created_by_id, created, updated) VALUES (?, ?, ?, ?, ?, ?)",
                item.title, "", item.color, int64(0), now, now,
            )
            if err != nil {
                return fmt.Errorf("create default label %q: %w", item.title, err)
            }
            found, err = session.SQL("SELECT id FROM labels WHERE title = ? ORDER BY id LIMIT 1", item.title).Get(&row)
            if err != nil || !found {
                return fmt.Errorf("read back default label %q: found=%t err=%w", item.title, found, err)
            }
        }

        exists := idRow{}
        mapped, err := session.SQL("SELECT label_id AS id FROM global_labels WHERE label_id = ?", row.ID).Get(&exists)
        if err != nil {
            return fmt.Errorf("read catalog membership %q: %w", item.title, err)
        }
        if !mapped {
            if _, err := session.Exec("INSERT INTO global_labels (label_id) VALUES (?)", row.ID); err != nil {
                return fmt.Errorf("register default label %q: %w", item.title, err)
            }
        }
    }

    if err := session.Commit(); err != nil {
        return fmt.Errorf("commit system labels: %w", err)
    }
    log.Infof("System label catalog initialized: %d defaults checked", len(defaults))
    return nil
}
