// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package apiv2

import (
	"context"
	"net/http"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/user"
	"github.com/danielgtaylor/huma/v2"
)

type historyListBody struct {
	Body Paginated[*models.ChangeHistory]
}

func RegisterChangeHistoryRoutes(api huma.API) {
	Register(api, huma.Operation{
		OperationID: "task-history-list",
		Summary: "Read immutable task change history",
		Description: "Returns field-level before/after values. Requires read permission on the task. No edit or delete history endpoints exist.",
		Method: http.MethodGet, Path: "/tasks/{task}/history", Tags: []string{"task"},
	}, taskHistoryList)
	Register(api, huma.Operation{
		OperationID: "project-history-list",
		Summary: "Read immutable project change history",
		Description: "Returns field-level before/after values. Requires read permission on the project. No edit or delete history endpoints exist.",
		Method: http.MethodGet, Path: "/projects/{project}/history", Tags: []string{"projects"},
	}, projectHistoryList)
	Register(api, huma.Operation{
		OperationID: "task-comment-history-list",
		Summary: "Read immutable comment edit history",
		Description: "Read the text before and after each edit. Parent-task read access required; no edit/delete history endpoint exists.",
		Method: http.MethodGet, Path: "/tasks/{task}/comments/{commentid}/history", Tags: []string{"task_comments"},
	}, commentHistoryList)
}

func init() { AddRouteRegistrar(RegisterChangeHistoryRoutes) }

func readChangeHistory(ctx context.Context, entityType string, id, taskID int64, page, perPage int) (*historyListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil { return nil, err }
	// Historical values can contain content later removed from the current
	// document. Public link shares must not expose superseded private data.
	if _, isShare := a.(*models.LinkSharing); isShare {
		return nil, huma.Error403Forbidden("History is not available through link shares")
	}
	if id < 1 { return nil, huma.Error404NotFound("Entity not found") }
	s := db.NewSession()
	defer s.Close()

	var allowed bool
	switch entityType {
	case "task":
		allowed, _, err = (&models.Task{ID: id}).CanRead(s, a)
	case "project":
		allowed, _, err = (&models.Project{ID: id}).CanRead(s, a)
	case "comment":
		if taskID <= 0 { return nil, huma.Error404NotFound("Comment not found") }
		// Validate comment-to-task ownership, not merely access to an unrelated
		// task: otherwise a caller could query another task's comment history.
		comment := &models.TaskComment{ID: id, TaskID: taskID}
		if readErr := comment.ReadOne(s, a); readErr != nil {
			return nil, translateDomainError(readErr)
		}
		allowed, _, err = (&models.Task{ID: taskID}).CanRead(s, a)
	default:
		return nil, huma.Error404NotFound("Entity not found")
	}
	if err != nil { return nil, translateDomainError(err) }
	if !allowed { return nil, huma.Error403Forbidden("Cannot read this history") }
	rows, total, err := models.ListHistory(s, entityType, id, page, perPage)
	if err != nil { return nil, translateDomainError(err) }

	// Historical entries survive user deletion: keep their actor ID intact
	// even if the user row no longer exists.
	ids := make([]int64, 0, len(rows))
	seen := make(map[int64]bool)
	for _, row := range rows {
		if row.ActorID > 0 && !seen[row.ActorID] {
			seen[row.ActorID] = true
			ids = append(ids, row.ActorID)
		}
	}
	if len(ids) > 0 {
		actors, err := user.GetUsersByIDs(s, ids)
		if err != nil { return nil, translateDomainError(err) }
		for _, row := range rows {
			if actor := actors[row.ActorID]; actor != nil {
				row.Username = actor.Username
			}
		}
	}
	return &historyListBody{Body: NewPaginated(rows, total, page, perPage)}, nil
}

func taskHistoryList(ctx context.Context, in *struct {
	ID int64 `path:"task"`
	ListParams
}) (*historyListBody, error) {
	return readChangeHistory(ctx, "task", in.ID, 0, in.Page, in.PerPage)
}

func projectHistoryList(ctx context.Context, in *struct {
	ID int64 `path:"project"`
	ListParams
}) (*historyListBody, error) {
	return readChangeHistory(ctx, "project", in.ID, 0, in.Page, in.PerPage)
}

func commentHistoryList(ctx context.Context, in *struct {
	TaskID int64 `path:"task"`
	ID int64 `path:"commentid"`
	ListParams
}) (*historyListBody, error) {
	return readChangeHistory(ctx, "comment", in.ID, in.TaskID, in.Page, in.PerPage)
}
