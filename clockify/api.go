package clockify

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/maelvls/clockidup/logutil"
	"github.com/sethgrid/gencurl"
)

type Clockify struct {
	client *http.Client
	server string
}

// NewClient creates a Clockify HTTP client. This function does not do any
// network call and does not check the validity of the token.
//
// This function is not thread-safe when giving it an existing client. If you do
// given a client with the WithClient option, only call NewClient function once,
// since it modifies the passed http.Client.
func NewClient(token string, opts ...Option) *Clockify {
	clockify := &Clockify{
		server: "https://api.clockify.me",
	}

	for _, option := range opts {
		option(clockify)
	}

	if clockify.client == nil {
		clockify.client = &http.Client{}
	}
	if clockify.client.Transport == nil {
		clockify.client.Transport = http.DefaultTransport
	}
	clockify.client.Transport = transport{
		trWrapped: clockify.client.Transport,
		token:     token,
	}

	return clockify
}

type Option func(*Clockify)

// WithServer allows you to set your own Clockify API server. By default, the
// server is https://api.clockify.me.
func WithServer(server string) Option {
	return func(c *Clockify) {
		c.server = server
	}
}

// WithClient allows you to set your own base client. The given client will be
// mutated in order to set the X-Api-Key header. You can use a nil client to use
// the default http.Client. By default, an empty &http.Client{} is used.
func WithClient(client *http.Client) Option {
	return func(c *Clockify) {
		c.client = client
	}
}

// Whenever an error occurs, Clockify responds with a JSON body that looks like
// this:
//
//  {
//    "message": "Full authentication is required to access this resource",
//    "code": 1000,
//  }
//
// The 'status' corresponds to the HTTP status code.
type ErrClockify struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Status  int
}

func (c ErrClockify) Error() string {
	return fmt.Sprintf("%d %s: %s", c.Status, http.StatusText(c.Status), c.Message)
}

// Some other errors look different, and they seem to only occur on
// routing-related errors. In the following example, the error appears due to
// the missing path segment 'projectID'. Since we assume that these "special"
// errors only occur at routing time, we decided to treat them as "unexpected
// errors".
//
//  {
//      "error": "Not Found",
//      "message": "",
//      "path": "/v1/workspaces/60e086c24f27a949c058082e/projects//tasks/60e0a9f00afa073620eade56",
//      "status": 404,
//      "timestamp": "2021-07-03T18:33:41.532+00:00"
//  }
//
// We also have errors have no body (and even no error code header). We also
// treat these errors as "unexpected errors".
type ErrUnexpect struct {
	RawResponseBody string
	Status          int
}

func (c ErrUnexpect) Error() string {
	if len(c.RawResponseBody) > 0 {
		return fmt.Sprintf("%d %s: (raw response body) %s", c.Status, http.StatusText(c.Status), c.RawResponseBody)
	}
	return fmt.Sprintf("%d %s (empty response body)", c.Status, http.StatusText(c.Status))
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
	req, err := http.NewRequest("GET", c.server+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request for GET %s: %w", path, err)
	}

	httpResp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("while calling GET %s: %w", path, err)
	}
	defer httpResp.Body.Close()

	bytes, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("while reading HTTP response for GET %s: %w", path, err)
	}

	switch httpResp.StatusCode {
	case 200:
		// continue below
	default:
		errClockify := ErrClockify{Status: httpResp.StatusCode}
		err = json.Unmarshal(bytes, &errClockify)
		if err != nil {
			return nil, ErrUnexpect{RawResponseBody: string(bytes), Status: httpResp.StatusCode}
		}
		return nil, errClockify
	}

	var workspaces []Workspace
	err = json.Unmarshal(bytes, &workspaces)
	if err != nil {
		logutil.Debugf("body: %s", bytes)
		return nil, fmt.Errorf("while parsing JSON from the HTTP response for GET %s: %w", path, err)
	}

	return workspaces, nil
}

type Project struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	HourlyRate     HourlyRate          `json:"hourlyRate"`
	ClientID       string              `json:"clientId"`
	WorkspaceID    string              `json:"workspaceId"`
	Billable       bool                `json:"billable"`
	Memberships    []Memberships       `json:"memberships"`
	Color          string              `json:"color"`
	Archived       bool                `json:"archived"`
	Duration       string              `json:"duration"`
	ClientName     string              `json:"clientName"`
	Note           string              `json:"note"`
	CostRate       interface{}         `json:"costRate"`
	TimeEstimate   ProjectTimeEstimate `json:"timeEstimate"`
	BudgetEstimate interface{}         `json:"budgetEstimate"`
	Public         bool                `json:"public"`
	Template       bool                `json:"template"`
}
type ProjectHourlyRate struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}
type ProjectMemberships struct {
	UserID           string      `json:"userId"`
	HourlyRate       interface{} `json:"hourlyRate"`
	CostRate         interface{} `json:"costRate"`
	TargetID         string      `json:"targetId"`
	MembershipType   string      `json:"membershipType"`
	MembershipStatus string      `json:"membershipStatus"`
}
type ProjectTimeEstimate struct {
	Estimate    string      `json:"estimate"`
	Type        string      `json:"type"`
	ResetOption interface{} `json:"resetOption"`
	Active      bool        `json:"active"`
}

var ErrEmptyWorkspaceID = fmt.Errorf("workspaceID is empty")
var ErrEmptyUserID = fmt.Errorf("userID is empty")
var ErrEmptyTaskID = fmt.Errorf("taskID is empty")
var ErrEmptyProjectID = fmt.Errorf("projectID is empty")

// May return ErrClockify, ErrUnexpect, or ErrEmptyWorkspaceID.
func (c *Clockify) Projects(workspaceID string) ([]Project, error) {
	if workspaceID == "" {
		return nil, ErrEmptyWorkspaceID
	}

	path := fmt.Sprintf("/api/v1/workspaces/%s/projects", workspaceID)
	req, err := http.NewRequest("GET", c.server+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request for GET %s: %w", path, err)
	}

	httpResp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("while calling GET %s: %w", path, err)
	}
	defer httpResp.Body.Close()

	bytes, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("while reading HTTP response for GET %s: %w", path, err)
	}

	switch httpResp.StatusCode {
	case 200:
		// continue below
	default:
		errClockify := ErrClockify{Status: httpResp.StatusCode}
		err = json.Unmarshal(bytes, &errClockify)
		if err != nil {
			return nil, ErrUnexpect{RawResponseBody: string(bytes), Status: httpResp.StatusCode}
		}
		return nil, errClockify
	}

	var projects []Project
	err = json.Unmarshal(bytes, &projects)
	if err != nil {
		return nil, fmt.Errorf("while parsing JSON from the HTTP response for GET %s: %w", path, err)
	}

	return projects, nil
}

type TimeEntry struct {
	ID                string        `json:"id"`
	Description       string        `json:"description"`
	TagIds            []interface{} `json:"tagIds"`
	UserID            string        `json:"userId"`
	Billable          bool          `json:"billable"`
	TaskID            string        `json:"taskId"`
	ProjectID         string        `json:"projectId"`
	TimeInterval      TimeInterval  `json:"timeInterval"`
	WorkspaceID       string        `json:"workspaceId"`
	IsLocked          bool          `json:"isLocked"`
	CustomFieldValues interface{}   `json:"customFieldValues"`
}
type TimeInterval struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Duration string    `json:"duration"`
}

// May return ErrClockify, ErrUnexpect, ErrEmptyWorkspaceID or ErrEmptyUserID.
func (c *Clockify) TimeEntries(workspaceID, userID string, start, end time.Time) ([]TimeEntry, error) {
	if workspaceID == "" {
		return nil, ErrEmptyWorkspaceID
	}
	if userID == "" {
		return nil, ErrEmptyUserID
	}

	path := fmt.Sprintf("/api/v1/workspaces/%s/user/%s/time-entries?start=%s&end=%s",
		workspaceID,
		userID,
		// the expected format is "2021-01-26T06:02:00Z" (ISO 8601) but
		// since RFC 3339 is a stricter version of ISO 8601, I use that
		// instead.
		start.UTC().Format(time.RFC3339),
		end.UTC().Format(time.RFC3339),
	)

	req, err := http.NewRequest("GET", c.server+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request for GET %s: %w", path, err)
	}

	httpResp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("while doing GET %s: %w", path, err)
	}
	defer httpResp.Body.Close()

	bytes, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("while reading HTTP response for %s: %w", path, err)
	}

	switch httpResp.StatusCode {
	case 200:
		// continue below
	default:
		errClockify := ErrClockify{Status: httpResp.StatusCode}
		err = json.Unmarshal(bytes, &errClockify)
		if err != nil {
			return nil, ErrUnexpect{RawResponseBody: string(bytes), Status: httpResp.StatusCode}
		}
		return nil, errClockify
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

// May return ErrClockify, ErrUnexpect, ErrEmptyWorkspaceID, ErrProjectID or
// ErrEmptyTaskID.
func (c *Clockify) Task(workspaceID, projectID, taskID string) (Task, error) {
	if workspaceID == "" {
		return Task{}, ErrEmptyWorkspaceID
	}
	if projectID == "" {
		return Task{}, ErrEmptyProjectID
	}
	if taskID == "" {
		return Task{}, ErrEmptyTaskID
	}

	path := fmt.Sprintf("/api/v1/workspaces/%s/projects/%s/tasks/%s",
		workspaceID,
		projectID,
		taskID,
	)

	req, err := http.NewRequest("GET", c.server+path, nil)
	if err != nil {
		return Task{}, fmt.Errorf("creating HTTP request for GET %s: %w", path, err)
	}

	httpResp, err := c.client.Do(req)
	if err != nil {
		return Task{}, fmt.Errorf("while doing GET %s: %w", path, err)
	}
	defer httpResp.Body.Close()

	bytes, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return Task{}, fmt.Errorf("while reading HTTP response for %s: %w", path, err)
	}

	switch httpResp.StatusCode {
	case 200:
		// continue below
	default:
		errClockify := ErrClockify{Status: httpResp.StatusCode}
		err = json.Unmarshal(bytes, &errClockify)
		if err != nil {
			return Task{}, ErrUnexpect{RawResponseBody: string(bytes), Status: httpResp.StatusCode}
		}
		return Task{}, errClockify
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
	if err != nil {
		return nil, err
	}

	// We won't show the body here since the io.Reader might already be
	// read somewhere else and it can only be read once. We could use a
	// buffer for that though...
	if logutil.EnableDebug {
		logutil.Debugf("%s [%d]", gencurl.FromRequest(r), resp.StatusCode)
	}

	return resp, nil
}

func Is(err error, status int) bool {
	var clErr ErrClockify
	switch {
	case errors.As(err, &clErr) && clErr.Status == status:
		return true
	default:
		return false
	}
}
