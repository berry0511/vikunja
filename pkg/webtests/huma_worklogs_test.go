// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package webtests

import (
	"net/http"
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Cross-project reports must apply each project's permissions independently:
// being a member of one project does not grant access to another author's
// entries there, nor to unrelated projects.
func TestHumaWorklogsAllProjects(t *testing.T) {
	e, err := setupTestEnv()
	require.NoError(t, err)

	// webtests provides testuser1 and testuser2 only. Load the existing
	// fixture owner of project 3 instead of referencing a nonexistent global.
	s := db.NewSession()
	ownerOfProject3, err := user.GetUserByID(s, 3)
	require.NoError(t, err)
	require.NoError(t, s.Close())

	create := func(token, body string) {
		t.Helper()
		rec := humaRequest(t, e, http.MethodPost, "/api/v2/worklogs", body, token, "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	}
	create(humaTokenFor(t, &testuser1), `{"project_id":1,"task_id":1,"work_date":"2026-10-19T00:00:00Z","minutes":60,"comment":"alpha"}`)
	create(humaTokenFor(t, ownerOfProject3), `{"project_id":3,"task_id":32,"work_date":"2026-10-19T00:00:00Z","minutes":45,"comment":"private-beta"}`)

	query := "/api/v2/worklogs?project_id=0&user_id=0&task_id=0&from_date=2026-10-19&to_date=2026-10-19"
	owner := humaRequest(t, e, http.MethodGet, query, "", humaTokenFor(t, &testuser1), "")
	require.Equal(t, http.StatusOK, owner.Code, owner.Body.String())
	assert.Contains(t, owner.Body.String(), `"project_id":1`)
	assert.Contains(t, owner.Body.String(), `"project_title":`)

	// user2 has project 3 read permission but is not its administrator.
	reader := humaRequest(t, e, http.MethodGet, query, "", humaTokenFor(t, &testuser2), "")
	require.Equal(t, http.StatusOK, reader.Code, reader.Body.String())
	assert.NotContains(t, reader.Body.String(), "private-beta")
	assert.NotContains(t, reader.Body.String(), `"project_id":1`)

	csv := humaRequest(t, e, http.MethodGet,
		"/api/v2/worklogs/export?project_id=0&user_id=0&task_id=0&from_date=2026-10-19&to_date=2026-10-19",
		"", humaTokenFor(t, &testuser1), "")
	require.Equal(t, http.StatusOK, csv.Code, csv.Body.String())
	assert.Contains(t, csv.Body.String(), "date,project,project_id,username,user_id,task_id,minutes,hours,comment")

	// Non-admins cannot force access to other people's project hours via
	// the all-projects summary or by specifying an explicit user id.
	forced := humaRequest(t, e, http.MethodGet,
		"/api/v2/worklogs/summary?project_id=0&user_id=3&task_id=0&from_date=2026-10-19&to_date=2026-10-19",
		"", humaTokenFor(t, &testuser2), "")
	require.Equal(t, http.StatusOK, forced.Code, forced.Body.String())
	assert.NotContains(t, forced.Body.String(), "private-beta")
}
