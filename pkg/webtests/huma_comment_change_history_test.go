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
	"fmt"
	"net/http"
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHumaCommentChangeHistory(t *testing.T) {
	e, err := setupTestEnv()
	require.NoError(t, err)
	owner := humaTokenFor(t, &testuser1)
	stranger := humaTokenFor(t, &testuser2)

	created := humaRequest(t, e, http.MethodPost, "/api/v2/tasks/1/comments",
		`{"comment":"original comment"}`, owner, "")
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	var comment struct { ID int64 `json:"id"` }
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &comment))
	require.Positive(t, comment.ID)
	url := fmt.Sprintf("/api/v2/tasks/1/comments/%d", comment.ID)

	changed := humaRequest(t, e, http.MethodPatch, url,
		`{"comment":"updated comment"}`, owner, "application/merge-patch+json")
	require.Equal(t, http.StatusOK, changed.Code, changed.Body.String())

	readURL := url + "/history"
	denied := humaRequest(t, e, http.MethodGet, readURL, "", stranger, "")
	assert.Equal(t, http.StatusForbidden, denied.Code, denied.Body.String())
	visible := humaRequest(t, e, http.MethodGet, readURL, "", owner, "")
	require.Equal(t, http.StatusOK, visible.Code, visible.Body.String())
	var history historyPageForTest
	require.NoError(t, json.Unmarshal(visible.Body.Bytes(), &history))
	require.GreaterOrEqual(t, history.Total, int64(1))
	var updated bool
	for _, row := range history.Items {
		if row.Field == "comment" && row.Action == "update" {
			updated = true
			assert.Contains(t, row.OldValue, "original comment")
			assert.Contains(t, row.NewValue, "updated comment")
			assert.Equal(t, int64(1), row.ActorID)
		}
	}
	assert.True(t, updated, "comment update must store original and updated bodies")

	// Deleting a comment removes the live record but not its audit evidence.
	gone := humaRequest(t, e, http.MethodDelete, url, "", owner, "")
	require.Equal(t, http.StatusNoContent, gone.Code, gone.Body.String())
	s := db.NewSession()
	defer s.Close()
	rows, total, err := models.ListHistory(s, "comment", comment.ID, 1, 50)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(2))
	var deletion bool
	for _, row := range rows {
		if row.Action == "delete" {
			deletion = true
			assert.Contains(t, row.OldValue, "updated comment")
		}
	}
	assert.True(t, deletion, "comment deletion must retain a historical snapshot")

	// Neither administrators nor ordinary users can rewrite history through
	// a public mutation endpoint.
	for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete} {
		rec := humaRequest(t, e, method, readURL, `{"old_value":"forged"}`, owner, "")
		assert.True(t, rec.Code == http.StatusNotFound || rec.Code == http.StatusMethodNotAllowed,
			"unexpected %s history mutation: %d %s", method, rec.Code, rec.Body.String())
	}
}
