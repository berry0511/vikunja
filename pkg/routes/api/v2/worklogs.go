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
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/user"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
)

// Worklogs are original, date-based manual records. No licensed feature flag,
// time_entries table, or paid time-tracking endpoint is changed or reused.
type dailyWorklogListBody struct {
	Body Paginated[*models.Worklog]
}

type dailyWorklogSummary struct {
	Days         []dailyWorklogDay `json:"days" doc:"Daily totals, grouped by person and calendar date."`
	TotalMinutes int               `json:"total_minutes" doc:"Total time in minutes across the filtered records."`
	RecordCount  int               `json:"record_count" doc:"Number of source worklog records."`
}

type dailyWorklogDay struct {
	WorkDate string `json:"work_date" doc:"Calendar date in YYYY-MM-DD format."`
	ProjectID int64 `json:"project_id" doc:"Project containing these hours."`
	ProjectTitle string `json:"project_title" doc:"Project name."`
	UserID   int64  `json:"user_id" doc:"Person's user id."`
	Username string `json:"username" doc:"Person's username."`
	Minutes  int    `json:"minutes" doc:"Minutes logged by this person on this date."`
}

type dailyWorklogSummaryBody struct {
	Body dailyWorklogSummary
}

// Huma only binds query parameters declared directly on an operation input.
type dailyWorklogFilterInput struct {
	ProjectID int64  `query:"project_id" required:"true" minimum:"0" doc:"Project id or zero for all accessible projects."`
	UserID    int64  `query:"user_id" doc:"Filter by person. Zero means everyone for project admins, or self for other members."`
	TaskID    int64  `query:"task_id" minimum:"0" doc:"Optional task id to filter this project's worklogs; zero includes every task."`
	FromDate  string `query:"from_date" required:"true" doc:"First included calendar day, YYYY-MM-DD."`
	ToDate    string `query:"to_date" required:"true" doc:"Last included calendar day, YYYY-MM-DD."`
}

func RegisterDailyWorklogRoutes(api huma.API) {
	tags := []string{"worklogs"}
	Register(api, huma.Operation{
		OperationID: "worklogs-list",
		Summary: "List manual daily worklogs",
		Description: "List entries for one project and inclusive date range. Members see their own records; project admins may request any person's or everyone's records.",
		Method: http.MethodGet, Path: "/worklogs", Tags: tags,
	}, dailyWorklogsList)
	Register(api, huma.Operation{
		OperationID: "worklogs-create",
		Summary: "Log minutes of work on a calendar date",
		Description: "Creates a dated manual work record under a required task in the project. Authenticated user is always the author; one record can cover up to 24 hours.",
		Method: http.MethodPost, Path: "/worklogs", Tags: tags,
	}, dailyWorklogsCreate)
	Register(api, huma.Operation{
		OperationID: "worklogs-update",
		Summary: "Edit your worklog",
		Description: "Only the original author may edit the date, minutes, task and note. Historical project and author cannot change.",
		Method: http.MethodPut, Path: "/worklogs/{id}", Tags: tags,
	}, dailyWorklogsUpdate)
	Register(api, huma.Operation{
		OperationID: "worklogs-delete",
		Summary: "Delete your worklog",
		Description: "Only the original author can remove a dated work record.",
		Method: http.MethodDelete, Path: "/worklogs/{id}", Tags: tags,
	}, dailyWorklogsDelete)
	Register(api, huma.Operation{
		OperationID: "worklogs-summary",
		Summary: "Summarize daily effort by person",
		Description: "Groups matching records by person and calendar day. Project admins can see the whole project, members only their own rows.",
		Method: http.MethodGet, Path: "/worklogs/summary", Tags: tags,
	}, dailyWorklogsSummary)
	Register(api, huma.Operation{
		OperationID: "worklogs-export-summary",
		Summary: "Export daily effort totals by person",
		Description: "Downloads a CSV of each person's daily minutes and hours for the filtered project and date range; only project admins can export another person's data.",
		Method: http.MethodGet, Path: "/worklogs/export-summary", Tags: tags,
		Responses: map[string]*huma.Response{
			"200": {Description: "Daily summary in UTF-8 CSV.",
				Content: map[string]*huma.MediaType{
					"text/csv": {Schema: &huma.Schema{Type: huma.TypeString, Format: "binary"}},
				},
			},
		},
	}, dailyWorklogsExportSummary)
	Register(api, huma.Operation{
		OperationID: "worklogs-export",
		Summary: "Export daily worklogs to UTF-8 CSV",
		Description: "Downloads full matching detail rows, not just the current page. Uses the same project and user permission rules as the list and summary.",
		Method: http.MethodGet, Path: "/worklogs/export", Tags: tags,
		Responses: map[string]*huma.Response{
			"200": {Description: "UTF-8 CSV attachment.",
				Content: map[string]*huma.MediaType{
					"text/csv": {Schema: &huma.Schema{Type: huma.TypeString, Format: "binary"}},
				},
			},
		},
	}, dailyWorklogsExport)
}

func init() { AddRouteRegistrar(RegisterDailyWorklogRoutes) }

func dailyWorklogError(err error) error {
	if err == nil { return nil }
	if errors.Is(err, models.ErrWorklogInvalid) {
		return huma.Error400BadRequest("Invalid worklog, project, person or calendar date")
	}
	var forbidden models.ErrGenericForbidden
	if errors.As(err, &forbidden) {
		return huma.Error403Forbidden("Not permitted to access these worklogs")
	}
	return translateDomainError(err)
}

func dailyWorklogDates(from, to string) (time.Time, time.Time, error) {
	start, err := time.Parse("2006-01-02", from)
	if err != nil { return time.Time{}, time.Time{}, models.ErrWorklogInvalid }
	end, err := time.Parse("2006-01-02", to)
	if err != nil || end.Before(start) || end.After(start.AddDate(10, 0, 0)) {
		return time.Time{}, time.Time{}, models.ErrWorklogInvalid
	}
	return start, end.AddDate(0, 0, 1), nil
}

// Project ID discovery for all-project reports never returns raw entries or
// project IDs to the caller. Every selected project's rows are read only after
// WorklogScope authorizes the request and restricts user visibility.
func dailyWorklogData(ctx context.Context, in dailyWorklogFilterInput) ([]*models.Worklog, error) {
	a, err := authFromCtx(ctx)
	if err != nil { return nil, err }
	if in.ProjectID < 0 || in.UserID < 0 || in.TaskID < 0 || (in.ProjectID == 0 && in.TaskID > 0) {
		return nil, dailyWorklogError(models.ErrWorklogInvalid)
	}
	start, end, err := dailyWorklogDates(in.FromDate, in.ToDate)
	if err != nil { return nil, dailyWorklogError(err) }
	s := db.NewSession()
	defer s.Close()

	projects := make([]*models.Project, 0)
	if in.ProjectID > 0 {
		// Explicit project requests fail with 403, rather than silently
		// returning an empty report for a project the caller cannot view.
		if _, err := models.WorklogScope(s, a, in.ProjectID, in.UserID); err != nil {
			return nil, dailyWorklogError(err)
		}
		p, err := models.GetProjectSimpleByID(s, in.ProjectID)
		if err != nil { return nil, translateDomainError(err) }
		projects = append(projects, p)
	} else {
		// Find only projects which actually have time entries in the requested
		// date range. Project.ReadAll is a presentation/pagination API, not an
		// exhaustive worklog scope: pseudo projects and pagination would make
		// bulk reports depend on unrelated navigation state.
		//
		// This first query reveals no data to the caller. Each candidate is
		// independently authorized with WorklogScope below before any record
		// is read, summarized, or exported.
		candidates := make([]*models.Worklog, 0)
		err := s.Distinct("project_id").
			Where("work_date >= ? AND work_date < ?", start, end).
			Find(&candidates)
		if err != nil { return nil, translateDomainError(err) }
		for _, candidate := range candidates {
			if candidate.ProjectID <= 0 { continue }
			p, err := models.GetProjectSimpleByID(s, candidate.ProjectID)
			if err != nil {
				// A deleted project may leave historical worklogs. It is no
				// longer an accessible project, so omit it in all-project views.
				var missing models.ErrProjectDoesNotExist
				if errors.As(err, &missing) { continue }
				return nil, translateDomainError(err)
			}
			projects = append(projects, p)
		}
	}

	rows := make([]*models.Worklog, 0)
	for _, project := range projects {
		userID, err := models.WorklogScope(s, a, project.ID, in.UserID)
		if err != nil {
			// A cross-project request does not grant a member access to
			// someone else's entries in projects where they are not an admin.
			// Explicit project queries above still return 403.
			var forbidden models.ErrGenericForbidden
			if in.ProjectID == 0 && errors.As(err, &forbidden) { continue }
			return nil, dailyWorklogError(err)
		}
		projectRows, err := models.ListProjectWorklogs(s, project.ID, userID, in.TaskID, start, end)
		if err != nil { return nil, translateDomainError(err) }
		for _, w := range projectRows {
			w.ProjectTitle = project.Title
		}
		rows = append(rows, projectRows...)
	}

	// Resolve authors once, retaining historical rows when an author no
	// longer exists. Sort globally rather than concatenating project blocks.
	ids, seen := make([]int64, 0), make(map[int64]bool)
	for _, w := range rows {
		if !seen[w.UserID] { ids = append(ids, w.UserID); seen[w.UserID] = true }
	}
	if len(ids) > 0 {
		users, err := user.GetUsersByIDs(s, ids)
		if err != nil { return nil, translateDomainError(err) }
		for _, w := range rows {
			if u := users[w.UserID]; u != nil { w.Username = u.Username }
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if !rows[i].WorkDate.Equal(rows[j].WorkDate) { return rows[i].WorkDate.Before(rows[j].WorkDate) }
		if rows[i].ProjectID != rows[j].ProjectID { return rows[i].ProjectID < rows[j].ProjectID }
		return rows[i].ID < rows[j].ID
	})
	return rows, nil
}

func dailyWorklogsList(ctx context.Context, in *struct {
	ProjectID int64  `query:"project_id" required:"true" minimum:"0"`
	UserID    int64  `query:"user_id"`
	TaskID    int64  `query:"task_id" minimum:"0"`
	FromDate  string `query:"from_date" required:"true"`
	ToDate    string `query:"to_date" required:"true"`
	ListParams
}) (*dailyWorklogListBody, error) {
	rows, err := dailyWorklogData(ctx, dailyWorklogFilterInput{
		ProjectID: in.ProjectID, UserID: in.UserID, TaskID: in.TaskID, FromDate: in.FromDate, ToDate: in.ToDate,
	})
	if err != nil { return nil, err }
	total := len(rows)
	start := (in.Page-1)*in.PerPage
	if start > total { start = total }
	end := start+in.PerPage
	if end > total { end = total }
	return &dailyWorklogListBody{Body: NewPaginated(rows[start:end], int64(total), in.Page, in.PerPage)}, nil
}

func dailyWorklogsCreate(ctx context.Context, in *struct {
	Body models.Worklog
}) (*singleBody[models.Worklog], error) {
	a, err := authFromCtx(ctx)
	if err != nil { return nil, err }
	w := &in.Body
	w.ID, w.UserID = 0, a.GetID()
	w.Username = ""
	s := db.NewSession()
	defer s.Close()
	allowed, err := w.CanCreate(s, a)
	if err != nil { return nil, dailyWorklogError(err) }
	if !allowed { return nil, huma.Error403Forbidden("Project or task is not accessible") }
	if _, err := s.Insert(w); err != nil { return nil, translateDomainError(err) }
	if err := s.Commit(); err != nil { return nil, translateDomainError(err) }
	return &singleBody[models.Worklog]{Body: w}, nil
}

func dailyWorklogsUpdate(ctx context.Context, in *struct {
	ID   int64 `path:"id"`
	Body models.Worklog
}) (*singleBody[models.Worklog], error) {
	a, err := authFromCtx(ctx)
	if err != nil { return nil, err }
	s := db.NewSession()
	defer s.Close()
	old := &models.Worklog{}
	exists, err := s.ID(in.ID).Get(old)
	if err != nil { return nil, translateDomainError(err) }
	if !exists { return nil, huma.Error404NotFound("Worklog not found") }
	w := &in.Body
	w.ID, w.ProjectID, w.UserID = old.ID, old.ProjectID, old.UserID
	allowed, err := w.CanUpdate(s, a)
	if err != nil { return nil, dailyWorklogError(err) }
	if !allowed { return nil, huma.Error403Forbidden("Only the author may edit this worklog") }
	if _, err := s.ID(w.ID).Cols("task_id", "work_date", "minutes", "comment").Update(w); err != nil {
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil { return nil, translateDomainError(err) }
	return &singleBody[models.Worklog]{Body: w}, nil
}

func dailyWorklogsDelete(ctx context.Context, in *struct {
	ID int64 `path:"id"`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil { return nil, err }
	s := db.NewSession()
	defer s.Close()
	w := &models.Worklog{ID: in.ID}
	allowed, err := w.CanDelete(s, a)
	if err != nil { return nil, dailyWorklogError(err) }
	if !allowed { return nil, huma.Error403Forbidden("Only the author may remove this worklog") }
	if _, err := s.ID(in.ID).Delete(&models.Worklog{}); err != nil { return nil, translateDomainError(err) }
	if err := s.Commit(); err != nil { return nil, translateDomainError(err) }
	return &emptyBody{}, nil
}

func dailyWorklogsSummary(ctx context.Context, in *dailyWorklogFilterInput) (*dailyWorklogSummaryBody, error) {
	rows, err := dailyWorklogData(ctx, *in)
	if err != nil { return nil, err }
	summary := summarizeDailyWorklogs(rows)
	return &dailyWorklogSummaryBody{Body: summary}, nil
}

func summarizeDailyWorklogs(rows []*models.Worklog) dailyWorklogSummary {
	out := dailyWorklogSummary{Days: make([]dailyWorklogDay, 0), RecordCount: len(rows)}
	keys := make(map[string]int)
	for _, r := range rows {
		day := r.WorkDate.UTC().Format("2006-01-02")
		key := day + "/" + strconv.FormatInt(r.ProjectID, 10) + "/" + strconv.FormatInt(r.UserID, 10)
		index, ok := keys[key]
		if !ok {
			index = len(out.Days)
			keys[key] = index
			out.Days = append(out.Days, dailyWorklogDay{
				WorkDate: day, ProjectID: r.ProjectID, ProjectTitle: r.ProjectTitle, UserID: r.UserID, Username: r.Username,
			})
		}
		out.Days[index].Minutes += r.Minutes
		out.TotalMinutes += r.Minutes
	}
	sort.Slice(out.Days, func(i, j int) bool {
		if out.Days[i].WorkDate == out.Days[j].WorkDate {
			if out.Days[i].ProjectID != out.Days[j].ProjectID {
				return out.Days[i].ProjectID < out.Days[j].ProjectID
			}
			return out.Days[i].UserID < out.Days[j].UserID
		}
		return out.Days[i].WorkDate < out.Days[j].WorkDate
	})
	return out
}

// csvCell prefixes spreadsheet formula starters so exported user text cannot
// become an executable spreadsheet formula upon opening the CSV in Excel.
func dailyWorklogCSVCell(value string) string {
	trimmed := strings.TrimLeftFunc(value, unicode.IsSpace)
	if trimmed != "" && strings.ContainsRune("=+-@", []rune(trimmed)[0]) {
		return "'" + value
	}
	return value
}

func dailyWorklogsExport(ctx context.Context, in *dailyWorklogFilterInput) (*huma.StreamResponse, error) {
	rows, err := dailyWorklogData(ctx, *in)
	if err != nil { return nil, err }
	return &huma.StreamResponse{Body: func(hctx huma.Context) {
		c := humaecho.Unwrap(hctx)
		w := (*c).Response()
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=worklogs.csv")
		_, _ = w.Write([]byte{0xef, 0xbb, 0xbf}) // Excel-friendly UTF-8 BOM.
		writer := csv.NewWriter(w)
		_ = writer.Write([]string{"date", "project", "project_id", "username", "user_id", "task_id", "minutes", "hours", "comment"})
		for _, r := range rows {
			_ = writer.Write([]string{
				r.WorkDate.UTC().Format("2006-01-02"),
				dailyWorklogCSVCell(r.ProjectTitle),
				strconv.FormatInt(r.ProjectID, 10),
				dailyWorklogCSVCell(r.Username),
				strconv.FormatInt(r.UserID, 10),
				strconv.FormatInt(r.TaskID, 10),
				strconv.Itoa(r.Minutes),
				fmt.Sprintf("%.2f", float64(r.Minutes)/60),
				dailyWorklogCSVCell(r.Comment),
			})
		}
		writer.Flush()
	}}, nil
}


func dailyWorklogsExportSummary(ctx context.Context, in *dailyWorklogFilterInput) (*huma.StreamResponse, error) {
	rows, err := dailyWorklogData(ctx, *in)
	if err != nil { return nil, err }
	sum := summarizeDailyWorklogs(rows)
	return &huma.StreamResponse{Body: func(hctx huma.Context) {
		c := humaecho.Unwrap(hctx)
		w := (*c).Response()
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=worklogs-daily-summary.csv")
		_, _ = w.Write([]byte{0xef, 0xbb, 0xbf})
		writer := csv.NewWriter(w)
		_ = writer.Write([]string{"date", "project", "project_id", "username", "user_id", "minutes", "hours"})
		for _, day := range sum.Days {
			_ = writer.Write([]string{
				day.WorkDate, dailyWorklogCSVCell(day.ProjectTitle),
				strconv.FormatInt(day.ProjectID, 10), dailyWorklogCSVCell(day.Username),
				strconv.FormatInt(day.UserID, 10),
				strconv.Itoa(day.Minutes),
				fmt.Sprintf("%.2f", float64(day.Minutes)/60),
			})
		}
		writer.Flush()
	}}, nil
}
