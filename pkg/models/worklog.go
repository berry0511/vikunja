// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.

package models

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"code.vikunja.io/api/pkg/web"
	"xorm.io/xorm"
)

// Worklog is a self-contained daily manual work record. It deliberately does not
// use or change the separately licensed time_entries resource.
type Worklog struct {
	ID        int64     `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"Unique worklog id."`
	UserID    int64     `xorm:"bigint not null index" json:"user_id" readOnly:"true" doc:"Person who logged the work; assigned by the server."`
	ProjectID int64     `xorm:"bigint not null index" json:"project_id" doc:"Historical project attribution. Cannot be changed after creation."`
	TaskID    int64     `xorm:"bigint not null default 0 index" json:"task_id" minimum:"1" doc:"Required task id in this project; every new worklog must be attached to a real task."`
	WorkDate  time.Time `xorm:"not null index" json:"work_date" doc:"Work day, stored as midnight UTC. Clients should send a UTC midnight timestamp for the selected calendar day."`
	Minutes   int       `xorm:"int not null" json:"minutes" minimum:"1" maximum:"1440" doc:"Actual minutes spent on the selected day, from 1 to 1440 per record."`
	Comment   string    `xorm:"text" json:"comment" doc:"Optional note of what was done, up to 2000 Unicode characters."`
	Created   time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"Creation timestamp."`
	Updated   time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"Last modification timestamp."`
	Username  string    `xorm:"-" json:"username" readOnly:"true" doc:"Display username, resolved from the user table."`
	ProjectTitle string `xorm:"-" json:"project_title" readOnly:"true" doc:"Project title, resolved at report time."`
}

func (*Worklog) TableName() string { return "worklogs" }

var ErrWorklogInvalid = errors.New("invalid daily worklog")

func ValidateWorklog(w *Worklog) error {
	if w.ProjectID <= 0 || w.TaskID <= 0 || w.Minutes < 1 || w.Minutes > 1440 ||
		w.WorkDate.IsZero() || w.WorkDate.Location() != time.UTC ||
		w.WorkDate.Hour() != 0 || w.WorkDate.Minute() != 0 ||
		w.WorkDate.Second() != 0 || w.WorkDate.Nanosecond() != 0 ||
		utf8.RuneCountInString(w.Comment) > 2000 {
		return ErrWorklogInvalid
	}
	w.Comment = strings.TrimSpace(w.Comment)
	return nil
}

// WorklogScope checks read rights before any list, summary or export query.
// Project admins can see everyone's hours; members see only their own.
// A requested user id of 0 means all users only for a project admin.
func WorklogScope(s *xorm.Session, a web.Auth, projectID, requestedUserID int64) (int64, error) {
	if projectID <= 0 || requestedUserID < 0 {
		return 0, ErrWorklogInvalid
	}
	if _, isShare := a.(*LinkSharing); isShare {
		return 0, ErrGenericForbidden{}
	}
	p := &Project{ID: projectID}
	read, _, err := p.CanRead(s, a)
	if err != nil {
		return 0, err
	}
	if !read {
		return 0, ErrGenericForbidden{}
	}
	admin, err := p.IsAdmin(s, a)
	if err != nil {
		return 0, err
	}
	if !admin {
		if requestedUserID != 0 && requestedUserID != a.GetID() {
			return 0, ErrGenericForbidden{}
		}
		return a.GetID(), nil
	}
	return requestedUserID, nil
}

// CanCreate permits a logged-in project member to record their own time,
// including on a task that currently belongs to that project.
func (w *Worklog) CanCreate(s *xorm.Session, a web.Auth) (bool, error) {
	if _, isShare := a.(*LinkSharing); isShare {
		return false, nil
	}
	if err := ValidateWorklog(w); err != nil {
		return false, err
	}
	p := &Project{ID: w.ProjectID}
	ok, _, err := p.CanRead(s, a)
	if err != nil || !ok {
		return ok, err
	}
	task, err := GetTaskByIDSimple(s, w.TaskID)
	if err != nil {
		return false, err
	}
	return task.ProjectID == w.ProjectID, nil
}

// CanUpdate and CanDelete never allow another user's hours to be modified,
// even by a project administrator. The historical project remains immutable.
func (w *Worklog) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	old := &Worklog{}
	found, err := s.ID(w.ID).Get(old)
	if err != nil || !found {
		return false, err
	}
	if old.UserID != a.GetID() || old.ProjectID != w.ProjectID {
		return false, nil
	}
	// Moving a task later must not make its historical records impossible to
	// correct. Only a deliberate change to a *different* task needs a new
	// task-to-project membership check.
	if w.TaskID == old.TaskID {
		if err := ValidateWorklog(w); err != nil {
			return false, err
		}
		p := &Project{ID: old.ProjectID}
		canRead, _, err := p.CanRead(s, a)
		return canRead, err
	}
	return w.CanCreate(s, a)
}

func (w *Worklog) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	old := &Worklog{}
	found, err := s.ID(w.ID).Get(old)
	if err != nil || !found {
		return false, err
	}
	if old.UserID != a.GetID() {
		return false, nil
	}
	_, err = WorklogScope(s, a, old.ProjectID, a.GetID())
	return err == nil, err
}

// ListProjectWorklogs fetches a project/date-scoped data set. All callers must
// first resolve WorklogScope, and pass that resolved user id (0=all).
// Date bounds use [from, to) so reporting never depends on server timezone.
func ListProjectWorklogs(s *xorm.Session, projectID, userID, taskID int64, from, to time.Time) ([]*Worklog, error) {
	query := s.Where("project_id = ? AND work_date >= ? AND work_date < ?", projectID, from, to)
	if userID > 0 {
		query = query.And("user_id = ?", userID)
	}
	if taskID > 0 {
		query = query.And("task_id = ?", taskID)
	}
	rows := make([]*Worklog, 0)
	err := query.OrderBy("work_date ASC, id ASC").Find(&rows)
	return rows, err
}
