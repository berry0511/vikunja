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

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The DB flag, not an untrusted caller claim, enables cross-user deletion.
// An instance admin may moderate another author's comment but cannot silently
// rewrite the user's words, and the immutable history must survive deletion.
func TestAdminCanDeleteOthersComment(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	author := &user.User{ID: 1}
	admin := &user.User{ID: 2, IsAdmin: false}
	comment := &TaskComment{TaskID: 1, Comment: "moderation evidence"}
	require.NoError(t, comment.Create(s, author))
	_, err := s.ID(admin.ID).Cols("is_admin").Update(&user.User{IsAdmin: true})
	require.NoError(t, err)
	stored := &TaskComment{ID: comment.ID, TaskID: 1}
	can, err := stored.CanDelete(s, admin)
	require.NoError(t, err)
	assert.True(t, can, "database-promoted admin can moderate comment")
	can, err = stored.CanUpdate(s, admin)
	require.NoError(t, err)
	assert.False(t, can, "admin override is deletion-only, not rewriting another user's comment")

	_, err = s.ID(admin.ID).Cols("is_admin").Update(&user.User{IsAdmin: false})
	require.NoError(t, err)
	can, err = stored.CanDelete(s, &user.User{ID: 2, IsAdmin: true})
	require.NoError(t, err)
	assert.False(t, can, "revoked admin JWT must not grant cross-user delete rights")
}
