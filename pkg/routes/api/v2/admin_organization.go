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
	"fmt"
	"net/http"
	"strings"
	"time"

	"code.vikunja.io/api/pkg/db"
	"github.com/danielgtaylor/huma/v2"
)

type organizationTask struct {
	ID int64 `json:"id"`
	Title string `json:"title"`
	ProjectID int64 `json:"project_id"`
	ProjectName string `json:"project_name"`
	Done bool `json:"done"`
	Priority int64 `json:"priority"`
	StartDate time.Time `json:"start_date"`
	EndDate time.Time `json:"end_date"`
	DueDate time.Time `json:"due_date"`
}

type organizationTeam struct {
	ID int64 `json:"id"`
	Name string `json:"name"`
	Members int64 `json:"members"`
	Projects int64 `json:"projects"`
}

type organizationTasksBody struct { Body Paginated[organizationTask] }
type organizationTeamsBody struct { Body Paginated[organizationTeam] }

func init() { AddRouteRegistrar(RegisterOrganizationRoutes) }

// The /admin path is guarded by gateV2AdminRoutes, which performs a fresh
// DB-backed IsAdmin check per request. Never mount these operations elsewhere.
func RegisterOrganizationRoutes(api huma.API) {
	Register(api, huma.Operation{
		OperationID: "admin-organization-tasks",
		Summary: "Instance-wide tasks (admin)",
		Description: "Paginated task overview, with optional team, project and assignee filters. Access is restricted to instance administrators.",
		Method: http.MethodGet, Path: "/admin/organization/tasks", Tags: []string{"admin"},
	}, organizationTasksList)
	Register(api, huma.Operation{
		OperationID: "admin-organization-teams",
		Summary: "Instance-wide team summary (admin)",
		Description: "Paginated summary of teams, memberships and shared projects, restricted to instance administrators.",
		Method: http.MethodGet, Path: "/admin/organization/teams", Tags: []string{"admin"},
	}, organizationTeamsList)
}

func organizationTasksList(ctx context.Context, in *struct {
	TeamID int64 `query:"team_id" minimum:"0"`
	ProjectID int64 `query:"project_id" minimum:"0"`
	AssigneeID int64 `query:"assignee_id" minimum:"0"`
	Done *bool `query:"done"`
	ListParams
}) (*organizationTasksBody, error) {
	if _, err := adminDoerFromCtx(ctx); err != nil { return nil, err }
	s := db.NewSession()
	defer s.Close()
	where := []string{"1=1"}
	args := make([]any, 0, 8)
	if in.TeamID > 0 {
		where = append(where, "EXISTS (SELECT 1 FROM team_projects tp WHERE tp.project_id = tasks.project_id AND tp.team_id = ?)")
		args = append(args, in.TeamID)
	}
	if in.ProjectID > 0 {
		where = append(where, "tasks.project_id = ?")
		args = append(args, in.ProjectID)
	}
	if in.AssigneeID > 0 {
		where = append(where, "EXISTS (SELECT 1 FROM task_assignees ta WHERE ta.task_id = tasks.id AND ta.user_id = ?)")
		args = append(args, in.AssigneeID)
	}
	if in.Done != nil {
		where = append(where, "tasks.done = ?")
		args = append(args, *in.Done)
	}
	if in.Q != "" {
		where = append(where, "LOWER(tasks.title) LIKE ?")
		args = append(args, "%" + strings.ToLower(in.Q) + "%")
	}
	base := " FROM tasks INNER JOIN projects ON projects.id = tasks.project_id WHERE " + strings.Join(where, " AND ")
	var count struct { Total int64 `xorm:"'total'"` }
	exists, err := s.SQL("SELECT COUNT(*) AS total" + base, args...).Get(&count)
	if err != nil { return nil, translateDomainError(err) }
	if !exists { count.Total = 0 }
	page, perPage := in.Page, in.PerPage
	if page < 1 { page = 1 }
	if perPage < 1 { perPage = 25 }
	if perPage > 100 { perPage = 100 }
	rows := []organizationTask{}
	selectSQL := "SELECT tasks.id AS id, tasks.title AS title, tasks.project_id AS project_id, projects.title AS project_name, tasks.done AS done, tasks.priority AS priority, tasks.start_date AS start_date, tasks.end_date AS end_date, tasks.due_date AS due_date" + base + " ORDER BY tasks.id DESC LIMIT ? OFFSET ?"
	queryArgs := append(append([]any{}, args...), perPage, (page-1)*perPage)
	if err := s.SQL(selectSQL, queryArgs...).Find(&rows); err != nil {
		return nil, translateDomainError(fmt.Errorf("organization tasks query: %w", err))
	}
	return &organizationTasksBody{Body: NewPaginated(rows, count.Total, page, perPage)}, nil
}

func organizationTeamsList(ctx context.Context, in *ListParams) (*organizationTeamsBody, error) {
	if _, err := adminDoerFromCtx(ctx); err != nil { return nil, err }
	s := db.NewSession()
	defer s.Close()
	page, perPage := in.Page, in.PerPage
	if page < 1 { page = 1 }
	if perPage < 1 { perPage = 25 }
	if perPage > 100 { perPage = 100 }
	where := ""
	args := []any{}
	if in.Q != "" {
		where = " WHERE LOWER(teams.name) LIKE ?"
		args = append(args, "%" + strings.ToLower(in.Q) + "%")
	}
	var count struct { Total int64 `xorm:"'total'"` }
	exists, err := s.SQL("SELECT COUNT(*) AS total FROM teams" + where, args...).Get(&count)
	if err != nil { return nil, translateDomainError(err) }
	if !exists { count.Total = 0 }
	rows := []organizationTeam{}
	q := "SELECT teams.id AS id, teams.name AS name, " +
		"(SELECT COUNT(*) FROM team_members tm WHERE tm.team_id = teams.id) AS members, " +
		"(SELECT COUNT(*) FROM team_projects tp WHERE tp.team_id = teams.id) AS projects " +
		"FROM teams" + where + " ORDER BY teams.name ASC LIMIT ? OFFSET ?"
	queryArgs := append(append([]any{}, args...), perPage, (page-1)*perPage)
	if err := s.SQL(q, queryArgs...).Find(&rows); err != nil { return nil, translateDomainError(err) }
	return &organizationTeamsBody{Body: NewPaginated(rows, count.Total, page, perPage)}, nil
}
