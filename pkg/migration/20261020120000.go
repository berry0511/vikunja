// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package migration

import (
	"regexp"
	"strconv"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

// This is an existing table. Add only the new column; do NOT run Sync2
// against a partial TaskAttachment struct, which could reshape old columns.
// All pre-existing attachments are task attachments.
type legacyCommentAttachment20261020120000 struct {
	ID int64 `xorm:"bigint autoincr not null unique pk"`
	TaskID int64 `xorm:"bigint not null"`
	Comment string `xorm:"text"`
}
func (*legacyCommentAttachment20261020120000) TableName() string { return "task_comments" }

// Legacy comment-editor uploads used the normal task attachment endpoint,
// so no source marker exists. Only relabel files referenced by a comment on
// the SAME task, without touching file bytes or metadata.
var legacyCommentFileLink20261020120000 = regexp.MustCompile(`/tasks/([0-9]+)/attachments/([0-9]+)\b`)

func backfillCommentAttachmentSources20261020120000(tx *xorm.Engine) error {
	var cursor int64
	for {
		var comments []*legacyCommentAttachment20261020120000
		if err := tx.Where("id > ?", cursor).OrderBy("id ASC").Limit(200).Find(&comments); err != nil { return err }
		if len(comments) == 0 { break }
		for _, comment := range comments {
			cursor = comment.ID
			for _, match := range legacyCommentFileLink20261020120000.FindAllStringSubmatch(comment.Comment, -1) {
				taskID, err := strconv.ParseInt(match[1], 10, 64)
				if err != nil || taskID != comment.TaskID { continue }
				attachmentID, err := strconv.ParseInt(match[2], 10, 64)
				if err != nil || attachmentID <= 0 { continue }
				if _, err = tx.Exec("UPDATE task_attachments SET source = ? WHERE id = ? AND task_id = ? AND source = ?", "comment", attachmentID, taskID, "task"); err != nil { return err }
			}
		}
		if len(comments) < 200 { break }
	}
	return nil
}

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
			if err := tx.AddColumn(&attachmentSource20261020120000{}, "source"); err != nil { return err }
			return backfillCommentAttachmentSources20261020120000(tx)
		},
		// Deliberately no destructive rollback of recorded file origins.
		Rollback: func(tx *xorm.Engine) error { return nil },
	})
}
