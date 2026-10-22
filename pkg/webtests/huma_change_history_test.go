// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package webtests

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type historyPageForTest struct {
	Items []struct {
		EntityType string `json:"entity_type"`
		ActorID int64 `json:"actor_id"`
		Action string `json:"action"`
		Field string `json:"field"`
		OldValue string `json:"old_value"`
		NewValue string `json:"new_value"`
	} `json:"items"`
	Total int64 `json:"total"`
}

func TestHumaChangeHistory(t *testing.T) {
	e, err := setupTestEnv()
	require.NoError(t, err)
	owner := humaTokenFor(t, &testuser1)
	other := humaTokenFor(t, &testuser2)
	get := func(path, token string) historyPageForTest {
		t.Helper()
		rec := humaRequest(t, e, http.MethodGet, path, "", token, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var history historyPageForTest
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &history))
		return history
	}
	patch := func(path, body string) {
		t.Helper()
		rec := humaRequest(t, e, http.MethodPatch, path, body, owner, "application/merge-patch+json")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	}
	patch("/api/v2/tasks/1", `{"title":"Audited task","start_date":"2026-10-20T08:00:00Z","end_date":"2026-10-20T17:00:00Z"}`)
	history := get("/api/v2/tasks/1/history", owner)
	require.GreaterOrEqual(t, history.Total, int64(3))
	fields := make(map[string]string)
	for _, item := range history.Items {
		assert.Equal(t, "task", item.EntityType)
		assert.Equal(t, int64(1), item.ActorID)
		fields[item.Field] = item.NewValue
	}
	assert.Equal(t, "Audited task", fields["title"])
	assert.Contains(t, fields["start_date"], "2026-10-20")
	assert.Contains(t, fields["end_date"], "2026-10-20")

	// A repeated update with identical values should not duplicate the history.
	countBefore := history.Total
	patch("/api/v2/tasks/1", `{"title":"Audited task"}`)
	history = get("/api/v2/tasks/1/history", owner)
	assert.Equal(t, countBefore, history.Total)

	patch("/api/v2/projects/1", `{"title":"Audited project"}`)
	history = get("/api/v2/projects/1/history", owner)
	var projectTitleSeen bool
	for _, item := range history.Items {
		if item.Field == "title" && item.NewValue == "Audited project" {
			projectTitleSeen = true
			assert.Equal(t, "project", item.EntityType)
		}
	}
	assert.True(t, projectTitleSeen)

	for _, path := range []string{"/api/v2/tasks/1/history", "/api/v2/projects/1/history"} {
		read := humaRequest(t, e, http.MethodGet, path, "", other, "")
		assert.Equal(t, http.StatusForbidden, read.Code, read.Body.String())
		write := humaRequest(t, e, http.MethodDelete, path, "", owner, "")
		assert.True(t, write.Code == http.StatusNotFound || write.Code == http.StatusMethodNotAllowed,
			"history should have no deletion endpoint, got %d: %s", write.Code, write.Body.String())
	}
}
