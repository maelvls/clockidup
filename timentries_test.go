package main

import (
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maelvls/clockidup/clockify"
	"github.com/maelvls/clockidup/mocks"
)

var record = os.Getenv("RECORD") != ""

func Test_findWorkspace(t *testing.T) {
	tests := map[string]struct {
		givenWorkspaces []clockify.Workspace
		givenName       string
		want            clockify.Workspace
		wantFound       bool
	}{
		"workspace exists": {
			givenWorkspaces: []clockify.Workspace{
				workspaceWith("workspace-1", "workspace-1-uid", "60e086c24f27a949c058082d"),
				workspaceWith("workspace-2", "60e08781bf81bd307230c097", "60e086c24f27a949c058082d"),
			},
			givenName: "workspace-2",
			wantFound: true,
			want:      workspaceWith("workspace-2", "60e08781bf81bd307230c097", "60e086c24f27a949c058082d"),
		},
		"workspace does not exist": {
			givenWorkspaces: []clockify.Workspace{
				workspaceWith("workspace-1", "workspace-1-uid", "60e086c24f27a949c058082d"),
				workspaceWith("workspace-2", "60e08781bf81bd307230c097", "60e086c24f27a949c058082d"),
			},
			givenName: "workspace-3",
			wantFound: false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, gotFound := findWorkspace(tt.givenWorkspaces, tt.givenName)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantFound, gotFound)
		})
	}
}

func workspaceWith(name, id, uid string) clockify.Workspace {
	return clockify.Workspace{
		ID:         id,
		Name:       name,
		HourlyRate: clockify.HourlyRate{Currency: "USD"},
		Memberships: []clockify.Memberships{{
			UserID:           uid,
			TargetID:         id,
			MembershipType:   "WORKSPACE",
			MembershipStatus: "ACTIVE",
		}},
		WorkspaceSettings: clockify.WorkspaceSettings{
			OnlyAdminsSeeBillableRates: true,
			OnlyAdminsCreateProject:    true,
			DefaultBillableProjects:    true,
			Round:                      clockify.Round{Round: "Round to nearest", Minutes: "15"},
			ProjectFavorites:           true,
			CanSeeTracker:              true,
			TrackTimeDownToSecond:      true,
			ProjectGroupingLabel:       "client",
			AdminOnlyPages:             []interface{}{},
			TimeTrackingMode:           "DEFAULT",
			IsProjectPublicByDefault:   true,
		},
		ImageURL: "",
	}
}

func Test_timeEntriesForDay(t *testing.T) {
	nowFixed := mustParse("2021-07-03T14:00:00Z")
	tests := []struct {
		name          string
		workspaceName string
		day           time.Time
		givenMock     func(m *mocks.MockclockifyClientMockRecorder)
		want          []timeEntry
		wantErr       error
	}{
		{
			name:          "three time entries returned",
			workspaceName: "workspace-1",
			day:           mustParse("2021-07-03T00:00:00Z"),
			givenMock: func(m *mocks.MockclockifyClientMockRecorder) {
				m.Workspaces().Return([]clockify.Workspace{
					{ID: "workspace-1-uid", Name: "workspace-1", Memberships: []clockify.Memberships{{UserID: "user-1-uid"}}},
					{ID: "workspace-2-uid", Name: "workspace-2", Memberships: []clockify.Memberships{{UserID: "user-2-uid", TargetID: "workspace-2-uid"}}},
				}, nil)
				m.TimeEntries("workspace-1-uid", "user-1-uid", mustParse("2021-07-03T00:00:00Z"), mustParse("2021-07-03T23:59:59Z")).Return([]clockify.TimeEntry{{
					ID: "entry-1-uid", WorkspaceID: "workspace-1-uid", UserID: "user-1-uid",
					Description:  "work with no project",
					Billable:     false,
					TimeInterval: clockify.TimeInterval{Start: mustParse("2021-07-03T13:30:00Z"), End: mustParse("2021-07-03T14:00:00Z"), Duration: "PT30M"},
				}, {
					ID: "entry-2-uid", WorkspaceID: "workspace-1-uid", UserID: "user-1-uid",
					ProjectID:    "project-1-uid",
					Description:  "some work with project but no task",
					Billable:     true,
					TimeInterval: clockify.TimeInterval{Start: mustParse("2021-07-03T13:00:00Z"), End: mustParse("2021-07-03T13:30:00Z"), Duration: "PT30M"},
				}, {
					ID: "entry-3-uid", WorkspaceID: "workspace-1-uid", UserID: "user-1-uid",
					Description:  "unit-test of clockidup, work with project and task",
					ProjectID:    "project-1-uid",
					TaskID:       "task-1-uid",
					Billable:     true,
					TimeInterval: clockify.TimeInterval{Start: mustParse("2021-07-03T12:30:00Z"), End: mustParse("2021-07-03T13:00:00Z"), Duration: "PT30M"},
				}}, nil)
				m.Projects("workspace-1-uid").Return([]clockify.Project{
					{ID: "project-1-uid", Name: "project-1"},
				}, nil)
				m.Task("workspace-1-uid", "project-1-uid", "task-1-uid").Return(clockify.Task{ID: "task-1-uid", Name: "task-1"}, nil)
			},
			want: []timeEntry{
				{Project: "", Description: "work with no project", Duration: 30 * time.Minute, Billable: false},
				{Project: "project-1", Description: "some work with project but no task", Duration: 30 * time.Minute, Billable: true},
				{Project: "project-1", Description: "unit-test of clockidup, work with project and task", Task: "task-1", Duration: 30 * time.Minute, Billable: true},
			},
		},
		{
			name:          "time entry still running started at 13:30 should have a duration of 30 minutes",
			workspaceName: "workspace-1",
			day:           mustParse("2021-07-03T00:00:00Z"),
			givenMock: func(m *mocks.MockclockifyClientMockRecorder) {
				m.Workspaces().Return([]clockify.Workspace{{ID: "workspace-1-uid", Name: "workspace-1", Memberships: []clockify.Memberships{{UserID: "user-1-uid"}}}}, nil)
				m.TimeEntries("workspace-1-uid", "user-1-uid", mustParse("2021-07-03T00:00:00Z"), mustParse("2021-07-03T23:59:59Z")).Return([]clockify.TimeEntry{{
					ID: "entry-1-uid", WorkspaceID: "workspace-1-uid", UserID: "user-1-uid",
					Description:  "time entry that is still going on (no end time)",
					TimeInterval: clockify.TimeInterval{Start: mustParse("2021-07-03T13:30:00Z")},
				}}, nil)
				m.Projects("workspace-1-uid").Return(nil, nil)
			},
			want: []timeEntry{
				{Project: "", Description: "time entry that is still going on (no end time)", Duration: 30 * time.Minute},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			client := mocks.NewMockclockifyClient(ctrl)

			tt.givenMock(client.EXPECT())

			got, err := timeEntriesForDay(client, func() time.Time { return nowFixed }, tt.workspaceName, tt.day)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Of the form "2006-01-02T15:04:05Z". Parses as UTC time.
func mustParse(s string) time.Time {
	t, err := time.Parse("2006-01-02T15:04:05Z", s)
	if err != nil {
		panic(err)
	}
	return t
}

func Test_selectBillable(t *testing.T) {
	tests := []struct {
		name  string
		given []timeEntry
		want  []timeEntry
	}{
		{
			given: []timeEntry{
				{Description: "time entry billable", Billable: true},
				{Description: "time entry not billable", Billable: false},
			},
			want: []timeEntry{
				{Description: "time entry billable", Billable: true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectBillable(tt.given)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_mergeSimilarEntries(t *testing.T) {
	tests := []struct {
		name    string
		given   []timeEntry
		want    []timeEntry
		wantErr error
	}{
		{
			name: "no similar entries",
			given: []timeEntry{
				{Description: "time entry 1", Duration: 30 * time.Minute},
				{Description: "time entry 1", Project: "project 1", Duration: 30 * time.Minute},
				{Description: "time entry 1", Project: "project 1", Task: "task 1", Duration: 30 * time.Minute},
				{Description: "time entry 2", Duration: 30 * time.Minute},
				{Description: "time entry 2", Project: "project 1", Duration: 30 * time.Minute},
				{Description: "time entry 2", Project: "project 1", Task: "task 1", Duration: 30 * time.Minute},
			},
			want: []timeEntry{
				{Description: "time entry 1", Duration: 30 * time.Minute},
				{Description: "time entry 1", Project: "project 1", Duration: 30 * time.Minute},
				{Description: "time entry 1", Project: "project 1", Task: "task 1", Duration: 30 * time.Minute},
				{Description: "time entry 2", Duration: 30 * time.Minute},
				{Description: "time entry 2", Project: "project 1", Duration: 30 * time.Minute},
				{Description: "time entry 2", Project: "project 1", Task: "task 1", Duration: 30 * time.Minute},
			},
		},
		{
			name: "merge when descriptions are equal",
			given: []timeEntry{
				{Description: "time entry 1", Duration: 30 * time.Minute},
				{Description: "time entry 1", Duration: 30 * time.Minute},
			},
			want: []timeEntry{
				{Description: "time entry 1", Duration: 60 * time.Minute},
			},
		},
		{
			name: "merge when descriptions and projects are equal",
			given: []timeEntry{
				{Description: "time entry 1", Project: "project 1", Duration: 20 * time.Minute},
				{Description: "time entry 1", Project: "project 1", Duration: 40 * time.Minute},
				{Description: "time entry 1", Project: "project 1", Duration: 20 * time.Minute},
			},
			want: []timeEntry{
				{Description: "time entry 1", Project: "project 1", Duration: 80 * time.Minute},
			},
		},
		{
			name: "merge when descriptions and projects and tasks are equal",
			given: []timeEntry{
				{Description: "time entry 1", Project: "project 1", Task: "task 1", Duration: 30 * time.Minute},
				{Description: "time entry 1", Project: "project 1", Task: "task 1", Duration: 30 * time.Minute},
				{Description: "time entry 1", Project: "project 1", Task: "task 1", Duration: 10 * time.Minute},
			},
			want: []timeEntry{
				{Description: "time entry 1", Project: "project 1", Task: "task 1", Duration: 70 * time.Minute},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeSimilarEntries(tt.given)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
