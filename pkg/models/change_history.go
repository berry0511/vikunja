// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"fmt"
	"strconv"
	"time"

	"code.vikunja.io/api/pkg/web"
	"xorm.io/xorm"
)

// ChangeHistory is an append-only, free operational change journal.
// No user-facing update/delete methods or routes exist. Rows are stored in
// the SAME database transaction as the task/project edit, not asynchronously.
// Old/new values can contain user-created rich text and must only be displayed
// as plain text by clients.
type ChangeHistory struct {
	ID         int64     `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true"`
	EntityType string    `xorm:"varchar(16) not null index" json:"entity_type" readOnly:"true"`
	EntityID   int64     `xorm:"bigint not null index" json:"entity_id" readOnly:"true"`
	ActorID    int64     `xorm:"bigint not null" json:"actor_id" readOnly:"true"`
	Username   string    `xorm:"-" json:"username" readOnly:"true"`
	Action     string    `xorm:"varchar(16) not null" json:"action" readOnly:"true"`
	Field      string    `xorm:"varchar(64) not null" json:"field" readOnly:"true"`
	OldValue   string    `xorm:"longtext" json:"old_value" readOnly:"true"`
	NewValue   string    `xorm:"longtext" json:"new_value" readOnly:"true"`
	Created    time.Time `xorm:"created not null" json:"created" readOnly:"true"`
}

func (*ChangeHistory) TableName() string { return "change_history" }

func historicalTime(t time.Time) string {
	if t.IsZero() { return "" }
	return t.UTC().Format(time.RFC3339Nano)
}

type historyField struct {
	name string
	oldValue, newValue string
}

func saveHistoryChanges(s *xorm.Session, entityType string, id int64, auth web.Auth, changes []historyField) error {
	if id <= 0 || auth == nil { return fmt.Errorf("history entry missing entity or actor") }
	for _, change := range changes {
		if change.oldValue == change.newValue { continue }
		entry := &ChangeHistory{
			EntityType: entityType, EntityID: id, ActorID: auth.GetID(),
			Action: "update", Field: change.name, OldValue: change.oldValue, NewValue: change.newValue,
		}
		if _, err := s.Insert(entry); err != nil { return err }
	}
	return nil
}

func saveHistoryDeletion(s *xorm.Session, entityType string, id int64, auth web.Auth) error {
	if id <= 0 || auth == nil { return fmt.Errorf("history deletion entry missing entity or actor") }
	_, err := s.Insert(&ChangeHistory{
		EntityType: entityType, EntityID: id, ActorID: auth.GetID(),
		Action: "delete", Field: "deleted",
		OldValue: "false", NewValue: "true",
	})
	return err
}

func taskHistoryDiff(before, after *Task) []historyField {
	return []historyField{
		{"title", before.Title, after.Title},
		{"description", before.Description, after.Description},
		{"project_id", strconv.FormatInt(before.ProjectID, 10), strconv.FormatInt(after.ProjectID, 10)},
		{"done", strconv.FormatBool(before.Done), strconv.FormatBool(after.Done)},
		{"done_at", historicalTime(before.DoneAt), historicalTime(after.DoneAt)},
		{"due_date", historicalTime(before.DueDate), historicalTime(after.DueDate)},
		{"start_date", historicalTime(before.StartDate), historicalTime(after.StartDate)},
		{"end_date", historicalTime(before.EndDate), historicalTime(after.EndDate)},
		{"priority", strconv.FormatInt(before.Priority, 10), strconv.FormatInt(after.Priority, 10)},
		{"percent_done", strconv.FormatFloat(before.PercentDone, 'f', -1, 64), strconv.FormatFloat(after.PercentDone, 'f', -1, 64)},
		{"repeat_after", strconv.FormatInt(before.RepeatAfter, 10), strconv.FormatInt(after.RepeatAfter, 10)},
		{"repeat_mode", fmt.Sprint(before.RepeatMode), fmt.Sprint(after.RepeatMode)},
		{"hex_color", before.HexColor, after.HexColor},
		{"bucket_id", strconv.FormatInt(before.BucketID, 10), strconv.FormatInt(after.BucketID, 10)},
		{"cover_image_attachment_id", strconv.FormatInt(before.CoverImageAttachmentID, 10), strconv.FormatInt(after.CoverImageAttachmentID, 10)},
	}
}

func projectHistoryDiff(before, after *Project) []historyField {
	parentID := func(p *Project) string {
		if p.ParentProjectID == nil { return "0" }
		return strconv.FormatInt(*p.ParentProjectID, 10)
	}
	return []historyField{
		{"title", before.Title, after.Title},
		{"description", before.Description, after.Description},
		{"is_archived", strconv.FormatBool(before.IsArchived), strconv.FormatBool(after.IsArchived)},
		{"identifier", before.Identifier, after.Identifier},
		{"hex_color", before.HexColor, after.HexColor},
		{"parent_project_id", parentID(before), parentID(after)},
		{"position", strconv.FormatFloat(before.Position, 'f', -1, 64), strconv.FormatFloat(after.Position, 'f', -1, 64)},
	}
}

// ListHistory returns an entity-scoped page. All callers MUST authorize read
// access to that entity before invoking it. No change-history write API is
// exposed, even to non-admin project editors.
func ListHistory(s *xorm.Session, entityType string, id int64, page, perPage int) ([]*ChangeHistory, int64, error) {
	if (entityType != "task" && entityType != "project") || id <= 0 { return nil, 0, fmt.Errorf("invalid history entity") }
	if page <= 0 { page = 1 }
	if perPage <= 0 || perPage > 100 { perPage = 50 }
	where := "entity_type = ? AND entity_id = ?"
	var count ChangeHistory
	total, err := s.Where(where, entityType, id).Count(&count)
	if err != nil { return nil, 0, err }
	rows := make([]*ChangeHistory, 0)
	err = s.Where(where, entityType, id).
		OrderBy("id DESC").
		Limit(perPage, (page-1)*perPage).
		Find(&rows)
	return rows, total, err
}
