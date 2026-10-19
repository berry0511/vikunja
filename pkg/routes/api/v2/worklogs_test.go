// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package apiv2

import (
	"testing"
	"time"

	"code.vikunja.io/api/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDailyWorklogsDateRange(t *testing.T) {
	from, exclusiveTo, err := dailyWorklogDates("2026-10-18", "2026-10-19")
	require.NoError(t, err)
	assert.Equal(t, "2026-10-18", from.Format("2006-01-02"))
	assert.Equal(t, "2026-10-20", exclusiveTo.Format("2006-01-02"))
	_, _, err = dailyWorklogDates("2026-10-20", "2026-10-19")
	require.Error(t, err)
	_, _, err = dailyWorklogDates("10/18/2026", "2026-10-19")
	require.Error(t, err)
}

func TestDailyWorklogsSummaryAndSpreadsheetEscaping(t *testing.T) {
	day := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	rows := []*models.Worklog{
		{WorkDate: day, UserID: 1, Username: "alice", Minutes: 60},
		{WorkDate: day, UserID: 1, Username: "alice", Minutes: 30},
		{WorkDate: day, UserID: 2, Username: "bob", Minutes: 120},
		{WorkDate: day.Add(-24*time.Hour), UserID: 1, Username: "alice", Minutes: 75},
	}
	out := summarizeDailyWorklogs(rows)
	assert.Equal(t, 4, out.RecordCount)
	assert.Equal(t, 285, out.TotalMinutes)
	require.Len(t, out.Days, 3)
	assert.Equal(t, "2026-10-18", out.Days[0].WorkDate)
	assert.Equal(t, 75, out.Days[0].Minutes)
	assert.Equal(t, 90, out.Days[1].Minutes)
	assert.Equal(t, 120, out.Days[2].Minutes)

	for _, raw := range []string{"=cmd()", "+1+2", "-3+5", "@sum(1)", "\t=cmd()", "\r=cmd()", "  =cmd()", "\n=cmd()"} {
		assert.Equal(t, "'"+raw, dailyWorklogCSVCell(raw))
	}
	assert.Equal(t, "regular text", dailyWorklogCSVCell("regular text"))
}
