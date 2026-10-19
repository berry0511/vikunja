// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"errors"
	"testing"
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func worklogFixture(projectID, taskID int64) *Worklog {
	return &Worklog{
		ProjectID: projectID, TaskID: taskID,
		WorkDate: time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC),
		Minutes: 90, Comment: "Daily work",
	}
}

func TestWorklogValidation(t *testing.T) {
	valid := worklogFixture(1, 1)
	require.NoError(t, ValidateWorklog(valid))
	for name, mutate := range map[string]func(*Worklog){
		"no project": func(w *Worklog) { w.ProjectID = 0 },
		"missing task": func(w *Worklog) { w.TaskID = 0 },
		"invalid task": func(w *Worklog) { w.TaskID = -1 },
		"zero minutes": func(w *Worklog) { w.Minutes = 0 },
		"too many minutes": func(w *Worklog) { w.Minutes = 1441 },
		"non-midnight date": func(w *Worklog) { w.WorkDate = w.WorkDate.Add(time.Hour) },
		"local timezone": func(w *Worklog) { w.WorkDate = w.WorkDate.In(time.FixedZone("UTC plus 8", 8*3600)) },
		"long comment": func(w *Worklog) { w.Comment = string(make([]rune, 2001)) },
	} {
		t.Run(name, func(t *testing.T) {
			copy := *valid
			mutate(&copy)
			assert.True(t, errors.Is(ValidateWorklog(&copy), ErrWorklogInvalid))
		})
	}
}

func TestWorklogPermissions(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	// The existing fixture has project 1 / task 1 owned by user 1,
	// shared project 3 readable to users 1 and 2, and user 4 as a stranger.
	allowed, err := worklogFixture(1, 1).CanCreate(s, &user.User{ID: 1})
	require.NoError(t, err)
	assert.True(t, allowed)
	allowed, err = worklogFixture(3, 32).CanCreate(s, &user.User{ID: 2})
	require.NoError(t, err)
	assert.True(t, allowed)
	allowed, err = worklogFixture(1, 1).CanCreate(s, &user.User{ID: 4})
	require.NoError(t, err)
	assert.False(t, allowed)
	allowed, err = worklogFixture(3, 1).CanCreate(s, &user.User{ID: 1})
	require.NoError(t, err)
	assert.False(t, allowed, "task must belong to selected project")
	allowed, err = worklogFixture(1, 1).CanCreate(s, &LinkSharing{ID: 1, ProjectID: 1})
	require.NoError(t, err)
	assert.False(t, allowed)

	onlySelf, err := WorklogScope(s, &user.User{ID: 2}, 3, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), onlySelf)
	_, err = WorklogScope(s, &user.User{ID: 2}, 3, 3)
	require.Error(t, err, "member must not query another person's entries")
	all, err := WorklogScope(s, &user.User{ID: 3}, 3, 0)
	require.NoError(t, err)
	assert.Zero(t, all, "project owner can request all people")
	_, err = WorklogScope(s, &user.User{ID: 4}, 3, 0)
	require.Error(t, err)
	_, err = WorklogScope(s, &LinkSharing{ID: 3, ProjectID: 3}, 3, 0)
	require.Error(t, err)
}

func TestWorklogSelfOnlyMutation(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	w := worklogFixture(3, 32)
	w.UserID = 3
	_, err := s.Insert(w)
	require.NoError(t, err)
	t.Cleanup(func() {
		clean := db.NewSession()
		defer clean.Close()
		_, err := clean.ID(w.ID).Delete(&Worklog{})
		require.NoError(t, err)
	})
	edit := worklogFixture(3, 32)
	edit.ID = w.ID
	can, err := edit.CanUpdate(s, &user.User{ID: 3})
	require.NoError(t, err)
	assert.True(t, can)
	can, err = edit.CanUpdate(s, &user.User{ID: 1})
	require.NoError(t, err)
	assert.False(t, can)
	can, err = (&Worklog{ID: w.ID}).CanDelete(s, &user.User{ID: 1})
	require.NoError(t, err)
	assert.False(t, can)
	can, err = (&Worklog{ID: w.ID}).CanDelete(s, &user.User{ID: 3})
	require.NoError(t, err)
	assert.True(t, can)
}
