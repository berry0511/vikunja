// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"testing"
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangeHistoryDiffIncludesTaskDatesAndTitle(t *testing.T) {
	old := &Task{Title: "Before", StartDate: time.Date(2026, 10, 19, 8, 0, 0, 0, time.UTC)}
	newTask := *old
	newTask.Title = "After"
	newTask.StartDate = old.StartDate.Add(24 * time.Hour)
	newTask.EndDate = time.Date(2026, 10, 20, 17, 0, 0, 0, time.UTC)
	changes := taskHistoryDiff(old, &newTask)
	fields := map[string]historyField{}
	for _, item := range changes {
		if item.oldValue != item.newValue { fields[item.name] = item }
	}
	assert.Len(t, fields, 3)
	assert.Equal(t, "Before", fields["title"].oldValue)
	assert.Equal(t, "After", fields["title"].newValue)
	assert.Contains(t, fields["start_date"].oldValue, "2026-10-19")
	assert.Contains(t, fields["start_date"].newValue, "2026-10-20")
	assert.Contains(t, fields["end_date"].newValue, "17:00:00")
}

func TestChangeHistoryAppendOnlyAndRollback(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	actor := &user.User{ID: 1}
	const taskID int64 = 1
	s := db.NewSession()
	baselineRows, baselineCount, err := ListHistory(s, "task", taskID, 1, 50)
	require.NoError(t, err)
	_ = baselineRows
	err = saveHistoryChanges(s, "task", taskID, actor, []historyField{
		{"title", "Old", "New"},
		{"priority", "1", "1"}, // no-op must not create a row
	})
	require.NoError(t, err)
	rows, count, err := ListHistory(s, "task", taskID, 1, 50)
	require.NoError(t, err)
	assert.Equal(t, baselineCount+1, count)
	assert.Len(t, rows, int(baselineCount)+1)
	assert.Equal(t, "New", rows[0].NewValue)
	require.NoError(t, s.Rollback())
	require.NoError(t, s.Close())

	read := db.NewSession()
	defer read.Close()
	_, count, err = ListHistory(read, "task", taskID, 1, 50)
	require.NoError(t, err)
	assert.Equal(t, baselineCount, count, "aborted transaction must not leave orphan history")
}
