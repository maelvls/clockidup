package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/maelvls/clockidup/logutil"
	"github.com/sethgrid/gencurl"
)

type Clockify struct {
	*http.Client
}

// The client can be left nil to use the default client. The given client
// will be mutated in order to set the X-Api-Key header. You can use a nil
// client to use the default http.Client.
//
// This function is not thread-safe when giving it an existing client. If
// you do, only call this function once, since it modifies the passed
// http.Client. This function does not do any network call and does not
// check the validity of the token.
func NewClockify(token string, cl *http.Client) *Clockify {
	if cl == nil {
		cl = &http.Client{}
	}

	if cl.Transport == nil {
		cl.Transport = http.DefaultTransport
	}

	cl.Transport = transport{
		trWrapped: cl.Transport,
		token:     token,
	}
	return &Clockify{Client: cl}
}

type Workspace struct {
	ID                      string            `json:"id"`
	Name                    string            `json:"name"`
	HourlyRate              HourlyRate        `json:"hourlyRate"`
	Memberships             []Memberships     `json:"memberships"`
	WorkspaceSettings       WorkspaceSettings `json:"workspaceSettings"`
	ImageURL                string            `json:"imageUrl"`
	FeatureSubscriptionType interface{}       `json:"featureSubscriptionType"`
}
type HourlyRate struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}
type Memberships struct {
	UserID           string      `json:"userId"`
	HourlyRate       interface{} `json:"hourlyRate"`
	CostRate         interface{} `json:"costRate"`
	TargetID         string      `json:"targetId"`
	MembershipType   string      `json:"membershipType"`
	MembershipStatus string      `json:"membershipStatus"`
}
type Round struct {
	Round   string `json:"round"`
	Minutes string `json:"minutes"`
}
type WorkspaceSettings struct {
	TimeRoundingInReports              bool          `json:"timeRoundingInReports"`
	OnlyAdminsSeeBillableRates         bool          `json:"onlyAdminsSeeBillableRates"`
	OnlyAdminsCreateProject            bool          `json:"onlyAdminsCreateProject"`
	OnlyAdminsSeeDashboard             bool          `json:"onlyAdminsSeeDashboard"`
	DefaultBillableProjects            bool          `json:"defaultBillableProjects"`
	LockTimeEntries                    interface{}   `json:"lockTimeEntries"`
	Round                              Round         `json:"round"`
	ProjectFavorites                   bool          `json:"projectFavorites"`
	CanSeeTimeSheet                    bool          `json:"canSeeTimeSheet"`
	CanSeeTracker                      bool          `json:"canSeeTracker"`
	ProjectPickerSpecialFilter         bool          `json:"projectPickerSpecialFilter"`
	ForceProjects                      bool          `json:"forceProjects"`
	ForceTasks                         bool          `json:"forceTasks"`
	ForceTags                          bool          `json:"forceTags"`
	ForceDescription                   bool          `json:"forceDescription"`
	OnlyAdminsSeeAllTimeEntries        bool          `json:"onlyAdminsSeeAllTimeEntries"`
	OnlyAdminsSeePublicProjectsEntries bool          `json:"onlyAdminsSeePublicProjectsEntries"`
	TrackTimeDownToSecond              bool          `json:"trackTimeDownToSecond"`
	ProjectGroupingLabel               string        `json:"projectGroupingLabel"`
	AdminOnlyPages                     []interface{} `json:"adminOnlyPages"`
	AutomaticLock                      interface{}   `json:"automaticLock"`
	OnlyAdminsCreateTag                bool          `json:"onlyAdminsCreateTag"`
	OnlyAdminsCreateTask               bool          `json:"onlyAdminsCreateTask"`
	TimeTrackingMode                   string        `json:"timeTrackingMode"`
	IsProjectPublicByDefault           bool          `json:"isProjectPublicByDefault"`
}

func (c *Clockify) Workspaces() ([]Workspace, error) {
	path := "/api/v1/workspaces"
	req, err := http.NewRequest("GET", "https://api.clockify.me"+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request for GET %s: %w", path, err)
	}

	httpResp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("while calling GET %s: %w", path, err)
	}
	defer httpResp.Body.Close()

	bytes, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("while reading HTTP response for GET %s: %w", path, err)
	}

	switch httpResp.StatusCode {
	case 400, 401, 403, 500:
		var errResp ClockifyError

		msg := "(raw body) " + string(bytes)
		err = json.Unmarshal(bytes, &errResp)
		if err == nil {
			msg = errResp.Message
		}
		return nil, fmt.Errorf("%s", msg)
	case 200:
		// continue below
	default:
		return nil, fmt.Errorf("unexpected HTTP status code %d for GET %s: %s", httpResp.StatusCode, path, bytes)
	}

	var workspaces []Workspace
	err = json.Unmarshal(bytes, &workspaces)
	if err != nil {
		return nil, fmt.Errorf("while parsing JSON from the HTTP response for GET %s: %w", path, err)
	}

	return workspaces, nil
}

func FindWorkspace(workspaces []Workspace, name string) (Workspace, bool) {
	// If no workspace is selected or name provided, we return that it is not found
	// You must now select a workspace during login or via the select subcommand
	if name == "" {
		return Workspace{}, false
	}

	for _, workspace := range workspaces {
		if workspace.Name == name {
			return workspace, true
		}
	}

	return Workspace{}, false
}

type Project struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	HourlyRate struct {
		Amount   int    `json:"amount"`
		Currency string `json:"currency"`
	} `json:"hourlyRate"`
	ClientID    string `json:"clientId"`
	WorkspaceID string `json:"workspaceId"`
	Billable    bool   `json:"billable"`
	Memberships []struct {
		UserID           string      `json:"userId"`
		HourlyRate       interface{} `json:"hourlyRate"`
		CostRate         interface{} `json:"costRate"`
		TargetID         string      `json:"targetId"`
		MembershipType   string      `json:"membershipType"`
		MembershipStatus string      `json:"membershipStatus"`
	} `json:"memberships"`
	Color    string `json:"color"`
	Estimate struct {
		Estimate string `json:"estimate"`
		Type     string `json:"type"`
	} `json:"estimate"`
	Archived     bool        `json:"archived"`
	Duration     string      `json:"duration"`
	ClientName   string      `json:"clientName"`
	Note         string      `json:"note"`
	CostRate     interface{} `json:"costRate"`
	TimeEstimate struct {
		Estimate    string      `json:"estimate"`
		Type        string      `json:"type"`
		ResetOption interface{} `json:"resetOption"`
		Active      bool        `json:"active"`
	} `json:"timeEstimate"`
	BudgetEstimate interface{} `json:"budgetEstimate"`
	Template       bool        `json:"template"`
	Public         bool        `json:"public"`
}

func (c *Clockify) Projects(workspaceID string) ([]Project, error) {
	path := fmt.Sprintf("/api/v1/workspaces/%s/projects", workspaceID)
	req, err := http.NewRequest("GET", "https://api.clockify.me"+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request for GET %s: %w", path, err)
	}

	httpResp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("while calling GET %s: %w", path, err)
	}
	defer httpResp.Body.Close()

	bytes, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("while reading HTTP response for GET %s: %w", path, err)
	}

	switch httpResp.StatusCode {
	case 400, 401, 403, 500:
		var errResp ClockifyError

		msg := "(raw body) " + string(bytes)
		err = json.Unmarshal(bytes, &errResp)
		if err == nil {
			msg = errResp.Message
		}
		return nil, fmt.Errorf("%s", msg)
	case 200:
		// continue below
	default:
		return nil, fmt.Errorf("unxpected HTTP status code %d for GET %s: %s", httpResp.StatusCode, path, bytes)
	}

	var projects []Project
	err = json.Unmarshal(bytes, &projects)
	if err != nil {
		return nil, fmt.Errorf("while parsing JSON from the HTTP response for GET %s: %w", path, err)
	}

	return projects, nil
}

type TimeEntry struct {
	ID           string   `json:"id"`
	Description  string   `json:"description"`
	TagIds       []string `json:"tagIds"`
	UserID       string   `json:"userId"`
	Billable     bool     `json:"billable"`
	TaskID       string   `json:"taskId"`
	ProjectID    string   `json:"projectId"`
	TimeInterval struct {
		Start    time.Time `json:"start"`
		End      time.Time `json:"end"`
		Duration string    `json:"duration"`
	} `json:"timeInterval"`
	WorkspaceID       string      `json:"workspaceId"`
	IsLocked          bool        `json:"isLocked"`
	CustomFieldValues interface{} `json:"customFieldValues"`
}

type ClockifyError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func (c *Clockify) TimeEntries(workspaceID, userID string, start, end time.Time) ([]TimeEntry, error) {
	path := fmt.Sprintf("/api/v1/workspaces/%s/user/%s/time-entries?start=%s&end=%s",
		workspaceID,
		userID,
		// the expected format is "2021-01-26T06:02:00Z" (ISO 8601) but
		// since RFC 3339 is a stricter version of ISO 8601, I use that
		// instead.
		start.UTC().Format(time.RFC3339),
		end.UTC().Format(time.RFC3339),
	)

	req, err := http.NewRequest("GET", "https://api.clockify.me"+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request for GET %s: %w", path, err)
	}

	httpResp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("while doing GET %s: %w", path, err)
	}
	defer httpResp.Body.Close()

	bytes, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("while reading HTTP response for %s: %w", path, err)
	}

	switch httpResp.StatusCode {
	case 400, 401, 403, 500:
		var errResp ClockifyError

		msg := "(raw body) " + string(bytes)
		err = json.Unmarshal(bytes, &errResp)
		if err == nil {
			msg = errResp.Message
		}
		return nil, fmt.Errorf("%s", msg)
	case 200:
		// continue below
	default:
		return nil, fmt.Errorf("unxpected HTTP status code %d for GET %s: %s", httpResp.StatusCode, path, bytes)
	}

	var timeEntry []TimeEntry
	err = json.Unmarshal(bytes, &timeEntry)
	if err != nil {
		return nil, fmt.Errorf("while parsing JSON from the HTTP response for GET %s: %w", path, err)
	}

	return timeEntry, nil
}

type Task struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	ProjectID   string   `json:"projectId"`
	AssigneeIds []string `json:"assigneeIds"`
	AssigneeID  string   `json:"assigneeId"`
	Estimate    string   `json:"estimate"`
	Status      string   `json:"status"`
	Duration    string   `json:"duration"`
}

func (c *Clockify) Task(workspaceID, projectID, taskID string) (Task, error) {
	path := fmt.Sprintf("/api/v1/workspaces/%s/projects/%s/tasks/%s",
		workspaceID,
		projectID,
		taskID,
	)

	req, err := http.NewRequest("GET", "https://api.clockify.me"+path, nil)
	if err != nil {
		return Task{}, fmt.Errorf("creating HTTP request for GET %s: %w", path, err)
	}

	httpResp, err := c.Client.Do(req)
	if err != nil {
		return Task{}, fmt.Errorf("while doing GET %s: %w", path, err)
	}
	defer httpResp.Body.Close()

	bytes, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return Task{}, fmt.Errorf("while reading HTTP response for %s: %w", path, err)
	}

	switch httpResp.StatusCode {
	case 400, 401, 403, 500:
		var errResp ClockifyError

		msg := "(raw body) " + string(bytes)
		err = json.Unmarshal(bytes, &errResp)
		if err == nil {
			msg = errResp.Message
		}
		return Task{}, fmt.Errorf("%s", msg)
	case 200:
		// continue below
	default:
		return Task{}, fmt.Errorf("unxpected HTTP status code %d for GET %s: %s", httpResp.StatusCode, path, bytes)
	}

	var task Task
	err = json.Unmarshal(bytes, &task)
	if err != nil {
		return Task{}, fmt.Errorf("while parsing JSON from the HTTP response for GET %s: %w", path, err)
	}

	return task, nil
}

type transport struct {
	trWrapped http.RoundTripper
	token     string
}

func (tr transport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("X-Api-Key", tr.token)
	resp, err := tr.trWrapped.RoundTrip(r)

	// We won't show the body here since the io.Reader might already be
	// read somewhere else and it can only be read once. We could use a
	// buffer for that though...
	if logutil.EnableDebug {
		logutil.Debugf("%s [%d]", gencurl.FromRequest(r), resp.StatusCode)
	}

	return resp, err
}
