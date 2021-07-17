package main

import (
	"fmt"
	"time"

	"github.com/maelvls/clockidup/clockify"
)

// timeEntry is similar to clockify.TimeEntry except it only contains what we
// need in clockidup e.g. names instead of IDs, and duration instead of start
// and end dates.
//
// Note that some time entries in Clockify may still be going on, in which case
// their end date is empty. In that case, we use the current UTC time (all times
// are in UTC in Clockify) to determine its duration.
type timeEntry struct {
	Project     string
	Description string
	Task        string
	Duration    time.Duration
	Billable    bool
}

// Testing purposes.
type clockifyClient interface {
	Workspaces() ([]clockify.Workspace, error)
	Projects(workspaceID string) ([]clockify.Project, error)
	TimeEntries(workspaceID, userID string, start, end time.Time) ([]clockify.TimeEntry, error)
	Task(workspaceID, projectID, taskID string) (clockify.Task, error)
}

// Times are UTC.
func timeEntriesForDay(client clockifyClient, now func() time.Time, workspaceName string, day time.Time) ([]timeEntry, error) {
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	end := time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 0, day.Location())

	workspaces, err := client.Workspaces()
	if err != nil {
		return nil, fmt.Errorf("%s", err)
	}
	if len(workspaces) == 0 {
		return nil, fmt.Errorf("no workspaces found, check your token and re-login via 'clockidup login'")
	}

	workspace, workspaceFound := findWorkspace(workspaces, workspaceName)
	if !workspaceFound {
		return nil, fmt.Errorf("unable to find workspace '%s'. Use 'clockidup select' or pass a workspace name with '--workspace'", workspaceName)
	}
	userID := workspace.Memberships[0].UserID

	timeEntries, err := client.TimeEntries(workspace.ID, userID, start, end)
	if err != nil {
		return nil, fmt.Errorf("%s", err)
	}

	projects, err := client.Projects(workspace.ID)
	if err != nil {
		return nil, fmt.Errorf("%s", err)
	}
	projectMap := make(map[string]*clockify.Project)
	for i := range projects {
		proj := &projects[i]
		projectMap[proj.ID] = proj
	}

	// Find the corresponding task when the taskId is set.
	var shortEntries []timeEntry
	for _, entry := range timeEntries {
		taskName := ""
		if entry.TaskID != "" {
			task, err := client.Task(entry.WorkspaceID, entry.ProjectID, entry.TaskID)
			if err != nil {
				return nil, fmt.Errorf("while fetching task for time entry '%s: %s': %s", projectMap[entry.ProjectID].Name, entry.Description, err)
			}
			taskName = task.Name
		}

		// When the time entry is still "ticking" i.e., the user has not
		// stopped the timer yet, the "end" date is null. In this case, we
		// still want to have an estimation of how long this entry has been
		// going on for.
		duration := entry.TimeInterval.End.Sub(entry.TimeInterval.Start)
		if entry.TimeInterval.End.IsZero() {
			duration = now().UTC().Sub(entry.TimeInterval.Start)
		}

		projectName := ""
		if entry.ProjectID != "" {
			p, ok := projectMap[entry.ProjectID]
			if !ok {
				return nil, fmt.Errorf("programmer mistake: projectID '%s' was supposed to exist in the projectMap!", entry.ProjectID)
			}

			projectName = p.Name
		}

		shortEntries = append(shortEntries, timeEntry{
			Project:     projectName,
			Description: entry.Description,
			Task:        taskName,
			Duration:    duration,
			Billable:    entry.Billable,
		})
	}

	return shortEntries, nil
}

// When onlyBillable is enabled, we leave out the non-billable entries.
func selectBillable(entries []timeEntry) []timeEntry {
	var selected []timeEntry
	for _, entry := range entries {
		if entry.Billable {
			selected = append(selected, entry)
		}
	}
	return selected
}

// mergeSimilarEntries merges similar time entries by summing up their
// durations. Similar entries have the same project, task and description
// simultaneously. For example, given the following time entries:
//
//   | Project   | Task   | Description                | Duration |
//   |-----------|--------|----------------------------|----------|
//   |           |        | "Review my emails"         | 1h       |
//   | project-2 |        | "Review PR"                | 40min    | ← merge 1
//   | project-1 |        | "Standup"                  | 30min    |
//   | project-1 |task-1  | "Implement user login"     | 30min    |
//   | project-2 |        | "Review PR"                | 10min    | ← merge 1
//   | project-1 |task-1  | "Deal with unit-testing"   | 30min    | ← merge 2
//   | project-1 |        | "Project meeting"          | 1h       |
//   | project-1 |task-1  | "Deal with unit-testing"   | 1h30     | ← merge 2
//
// Two pairs of entries get merged. The resulting time entries returned are:
//
//   | Project   | Task   | Description                | Duration |
//   |-----------|--------|----------------------------|----------|
//   |           |        | "Review my emails"         | 1h       |
//   | project-2 |        | "Review PR"                | 50min    | ← 1
//   | project-1 |        | "Standup"                  | 30min    |
//   | project-1 |task-1  | "Implement user login"     | 30min    |
//   | project-1 |task-1  | "Deal with unit-testing"   | 2h       | ← 2
//   | project-1 |        | "Project meeting"          | 1h       |
//
// Note that the order of the merged entries corresponds to the order of first
// appearance of the similar entries.
func mergeSimilarEntries(entries []timeEntry) ([]timeEntry, error) {
	type key struct{ project, task, descr string }
	entriesSeen := make(map[key]*timeEntry)
	var mergedEntries []timeEntry
	for _, entry := range entries {
		existing, alreadySeen := entriesSeen[key{entry.Project, entry.Task, entry.Description}]
		if alreadySeen {
			existing.Duration += entry.Duration
			continue
		}

		mergedEntries = append(mergedEntries, entry)
		entriesSeen[key{entry.Project, entry.Task, entry.Description}] = &mergedEntries[len(mergedEntries)-1]
	}

	return mergedEntries, nil
}

// Now, we want to mash the project name and task name into the description
// of the time entry. The format is "project 1: task 1: work on clockidup".
func String(entry timeEntry) string {
	str := entry.Description
	if entry.Task != "" {
		str = entry.Task + ":" + str
	}
	if entry.Project != "" {
		str = entry.Project + ":" + str
	}

	return str
}

func findWorkspace(workspaces []clockify.Workspace, name string) (clockify.Workspace, bool) {
	// If no workspace is selected or name provided, we return that it is not
	// found You must now select a workspace during login or via the select
	// subcommand.
	if name == "" {
		return clockify.Workspace{}, false
	}

	for _, workspace := range workspaces {
		if workspace.Name == name {
			return workspace, true
		}
	}

	return clockify.Workspace{}, false
}
