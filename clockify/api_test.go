package clockify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/dnaeon/go-vcr/cassette"
	"github.com/dnaeon/go-vcr/recorder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var record = os.Getenv("RECORD") != ""

func TestClockify_Workspaces(t *testing.T) {
	tr := withReplayTransport(t)

	t.Run("two workspaces are returned", func(t *testing.T) {
		clockify := NewClient(withToken(t), WithClient(&http.Client{Transport: tr}))

		got, gotErr := clockify.Workspaces()

		require.NoError(t, gotErr)
		assert.Equal(t, []Workspace{
			workspaceWith("workspace-1", "60e086c24f27a949c058082e", "60e086c24f27a949c058082d"),
			workspaceWith("workspace-2", "60e08781bf81bd307230c097", "60e086c24f27a949c058082d"),
		}, got)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		clockify := NewClient("invalid-token", WithClient(&http.Client{Transport: tr}))

		got, gotErr := clockify.Workspaces()

		assert.Equal(t, []Workspace(nil), got)
		require.EqualError(t, gotErr, "401 Unauthorized: Full authentication is required to access this resource")
	})
}

func workspaceWith(name, id, uid string) Workspace {
	return Workspace{
		ID:         id,
		Name:       name,
		HourlyRate: HourlyRate{Currency: "USD"},
		Memberships: []Memberships{{
			UserID:           uid,
			TargetID:         id,
			MembershipType:   "WORKSPACE",
			MembershipStatus: "ACTIVE",
		}},
		WorkspaceSettings: WorkspaceSettings{
			OnlyAdminsSeeBillableRates: true,
			OnlyAdminsCreateProject:    true,
			DefaultBillableProjects:    true,
			Round:                      Round{Round: "Round to nearest", Minutes: "15"},
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

func TestClockify_Projects(t *testing.T) {
	tr := withReplayTransport(t)

	t.Run("the requested project id exists", func(t *testing.T) {
		clockify := NewClient(withToken(t), WithClient(&http.Client{Transport: tr}))

		got, gotErr := clockify.Projects("60e086c24f27a949c058082e") // "workspace-1"

		require.NoError(t, gotErr)
		assert.Equal(t, []Project{{
			ID:          "60e0a9cf5f596c5a7d10d821",
			Name:        "project-1",
			HourlyRate:  HourlyRate{Amount: 0, Currency: "USD"},
			ClientID:    "",
			WorkspaceID: "60e086c24f27a949c058082e",
			Billable:    true,
			Memberships: []Memberships{{
				UserID:           "60e086c24f27a949c058082d",
				HourlyRate:       interface{}(nil),
				CostRate:         interface{}(nil),
				TargetID:         "60e0a9cf5f596c5a7d10d821",
				MembershipType:   "PROJECT",
				MembershipStatus: "ACTIVE",
			}},
			Color:      "#795548",
			Archived:   false,
			Duration:   "PT1H",
			ClientName: "",
			Note:       "",
			CostRate:   interface{}(nil),
			TimeEstimate: ProjectTimeEstimate{
				Estimate: "PT0S",
				Type:     "AUTO",
				Active:   false,
			},
			Public:   true,
			Template: false,
		}}, got)
	})

	t.Run("the requested project id does not exist", func(t *testing.T) {
		clockify := NewClient(withToken(t), WithClient(&http.Client{Transport: tr}))

		got, gotErr := clockify.Projects("some-dummy-id")

		require.EqualError(t, gotErr, "403 Forbidden (empty response body)")
		assert.Equal(t, []Project(nil), got)
	})

	t.Run("empty project id", func(t *testing.T) {
		clockify := NewClient(withToken(t), WithClient(&http.Client{Transport: tr}))

		got, gotErr := clockify.Projects("")

		require.Equal(t, ErrEmptyWorkspaceID, gotErr)
		assert.Equal(t, []Project(nil), got)
	})
}

func TestClockify_Task(t *testing.T) {
	tr := withReplayTransport(t)

	t.Run("the requested task id exists", func(t *testing.T) {
		clockify := NewClient(withToken(t), WithClient(&http.Client{Transport: tr}))

		got, gotErr := clockify.Task(
			"60e086c24f27a949c058082e", // "workspace-1"
			"60e0a9cf5f596c5a7d10d821", // "project-1"
			"60e0a9f00afa073620eade56", // "task-1"
		)

		require.NoError(t, gotErr)
		assert.Equal(t, Task{
			ID:          "60e0a9f00afa073620eade56",
			Name:        "task-1",
			ProjectID:   "60e0a9cf5f596c5a7d10d821",
			AssigneeIds: []string{},
			Estimate:    "PT0S",
			Status:      "ACTIVE",
			Duration:    "PT0S",
		}, got)
	})

	t.Run("the requested task id does not exist", func(t *testing.T) {
		clockify := NewClient(withToken(t), WithClient(&http.Client{Transport: tr}))

		got, gotErr := clockify.Task(
			"60e086c24f27a949c058082e", // "workspace-1"
			"60e0a9cf5f596c5a7d10d821", // "project-1"
			"dummy",
		)

		require.EqualError(t, gotErr, "404 Not Found: TASK with ID 'dummy' not found.")
		assert.Equal(t, Task{}, got)
	})

	t.Run("empty workspace id", func(t *testing.T) {
		clockify := NewClient(withToken(t), WithClient(&http.Client{Transport: tr}))

		got, gotErr := clockify.Task("", "60e0a9cf5f596c5a7d10d821", "60e0a9f00afa073620eade56")

		require.Equal(t, ErrEmptyWorkspaceID, gotErr)
		assert.Equal(t, Task{}, got)
	})

	t.Run("empty project id", func(t *testing.T) {
		clockify := NewClient(withToken(t), WithClient(&http.Client{Transport: tr}))

		got, gotErr := clockify.Task("60e086c24f27a949c058082e", "", "60e0a9f00afa073620eade56")

		require.Equal(t, ErrEmptyProjectID, gotErr)
		assert.Equal(t, Task{}, got)
	})

	t.Run("empty task id", func(t *testing.T) {
		clockify := NewClient(withToken(t), WithClient(&http.Client{Transport: tr}))

		got, gotErr := clockify.Task("60e086c24f27a949c058082e", "60e0a9cf5f596c5a7d10d821", "")

		require.Equal(t, ErrEmptyTaskID, gotErr)
		assert.Equal(t, Task{}, got)
	})
}

func TestClockify_TimeEntries(t *testing.T) {
	tr := withReplayTransport(t)

	t.Run("the requested workspace and user exist", func(t *testing.T) {
		clockify := NewClient(withToken(t), WithClient(&http.Client{Transport: tr}))

		got, gotErr := clockify.TimeEntries(
			"60e086c24f27a949c058082e", // "workspace-1"
			"60e086c24f27a949c058082d", // "user-1"
			mustParse("2021-07-03 00:00:00"),
			mustParse("2021-07-03 23:59:00"),
		)

		require.NoError(t, gotErr)
		assert.Equal(t, []TimeEntry([]TimeEntry{{
			ID:          "60e0ccf4909afe51901a154c",
			Description: "Work with no project",
			TagIds:      []interface{}{},
			UserID:      "60e086c24f27a949c058082d",
			Billable:    false,
			TaskID:      "",
			ProjectID:   "",
			TimeInterval: TimeInterval{
				Start:    mustParse("2021-07-03 13:30:00"),
				End:      mustParse("2021-07-03 14:00:00"),
				Duration: "PT30M",
			},
			WorkspaceID:       "60e086c24f27a949c058082e",
			IsLocked:          false,
			CustomFieldValues: interface{}(nil),
		}, {
			ID:          "60e0cccf909afe51901a151c",
			Description: "Some work with project but no task",
			TagIds:      []interface{}{},
			UserID:      "60e086c24f27a949c058082d",
			Billable:    true,
			TaskID:      "",
			ProjectID:   "60e0a9cf5f596c5a7d10d821",
			TimeInterval: TimeInterval{
				Start:    mustParse("2021-07-03 13:00:00"),
				End:      mustParse("2021-07-03 13:30:00"),
				Duration: "PT30M",
			},
			WorkspaceID:       "60e086c24f27a949c058082e",
			IsLocked:          false,
			CustomFieldValues: interface{}(nil),
		}, {
			ID:          "60e0ccbc4f27a949c058498b",
			Description: "Unit-test of clockidup, work with project and task",
			TagIds:      []interface{}{},
			UserID:      "60e086c24f27a949c058082d",
			Billable:    true,
			TaskID:      "60e0a9f00afa073620eade56",
			ProjectID:   "60e0a9cf5f596c5a7d10d821",
			TimeInterval: TimeInterval{
				Start:    mustParse("2021-07-03 12:30:00"),
				End:      mustParse("2021-07-03 13:00:00"),
				Duration: "PT30M",
			},
			WorkspaceID:       "60e086c24f27a949c058082e",
			IsLocked:          false,
			CustomFieldValues: interface{}(nil),
		}}), got)
	})
	t.Run("the requested workspace does not exist", func(t *testing.T) {
		clockify := NewClient(withToken(t), WithClient(&http.Client{Transport: tr}))

		got, gotErr := clockify.TimeEntries(
			"dummy-workspace",
			"60e086c24f27a949c058082d", // "user-1"
			mustParse("2021-07-03 00:00:00"),
			mustParse("2021-07-03 23:59:00"),
		)

		require.EqualError(t, gotErr, "403 Forbidden (empty response body)")
		assert.Equal(t, []TimeEntry(nil), got)
	})
	t.Run("the requested user does not exist", func(t *testing.T) {
		clockify := NewClient(withToken(t), WithClient(&http.Client{Transport: tr}))

		got, gotErr := clockify.TimeEntries(
			"60e086c24f27a949c058082e", // "workspace-1"
			"dummy-user-id",
			mustParse("2021-07-03 00:00:00"),
			mustParse("2021-07-03 23:59:00"),
		)

		require.EqualError(t, gotErr, "404 Not Found: USER with ID 'dummy-user-id' not found.")
		assert.Equal(t, []TimeEntry(nil), got)
	})
	t.Run("the requested workspace is empty", func(t *testing.T) {
		clockify := NewClient(withToken(t), WithClient(&http.Client{Transport: tr}))

		got, gotErr := clockify.TimeEntries(
			"",
			"60e086c24f27a949c058082d",
			mustParse("2021-07-03 00:00:00"),
			mustParse("2021-07-03 23:59:00"),
		)

		require.Equal(t, gotErr, ErrEmptyWorkspaceID)
		assert.Equal(t, []TimeEntry(nil), got)
	})
	t.Run("the requested user is empty", func(t *testing.T) {
		clockify := NewClient(withToken(t), WithClient(&http.Client{Transport: tr}))

		got, gotErr := clockify.TimeEntries(
			"60e086c24f27a949c058082e", // "workspace-1"
			"",
			mustParse("2021-07-03 00:00:00"),
			mustParse("2021-07-03 23:59:00"),
		)

		require.Equal(t, gotErr, ErrEmptyUserID)
		assert.Equal(t, []TimeEntry(nil), got)
	})

}

func withToken(t *testing.T) (token string) {
	if record {
		return mustGetenv(t, "CLOCKIFY_TOKEN")
	}
	return "redacted-token"
}

func withReplayTransport(t *testing.T) *recorder.Recorder {
	mode := recorder.ModeReplaying
	if record {
		mode = recorder.ModeRecording
	}

	rec, err := recorder.NewAsMode("fixtures/"+strings.ToLower(t.Name()), mode, http.DefaultTransport)
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = rec.Stop()
	})

	// The response's token is only filtered in recording mode (RECORD=1). The
	// filter does nothing in replay mode.
	rec.AddFilter(func(i *cassette.Interaction) error {
		if i.Request.Headers.Get("X-Api-Key") == withToken(t) {
			i.Request.Headers.Set("X-Api-Key", "redacted-token")
		}
		return nil
	})

	return rec
}

func mustGetenv(t *testing.T, v string) string {
	res := os.Getenv(v)
	if res == "" {
		t.Errorf("The env var %s is not set or is empty. Did you forget to 'export %s=value' or to add it to your '.envrc'?", v, v)
		t.FailNow()
	}
	return res
}

// For example:
//   req.Body = RedactJSON(req.Body, "userId", "fake-user-id", "id", "fake-id", "targetId", "fake-target-id")
//   resp.Body = RedactJSON(resp.Body, "userId", "fake-user-id", "id", "fake-id", "targetId", "fake-target-id")
func RedactJSON(body string, replaceKeyWith ...string) string {
	replaceMap := make(map[string]string)
	for i := 0; i < len(replaceKeyWith)-1; i = i + 2 {
		replaceMap[replaceKeyWith[i]] = replaceKeyWith[i+1]
	}

	if len(body) == 0 {
		return ""
	}

	var jsonBlob interface{}
	err := json.Unmarshal([]byte(body), &jsonBlob)
	if err != nil {
		panic(fmt.Errorf("while redacting the JSON body: %w", err))
	}
	redactValue(replaceMap, &jsonBlob)

	bodyBytes, err := json.Marshal(jsonBlob)
	if err != nil {
		panic(fmt.Errorf("while marshalling the redacted JSON body: %w", err))
	}

	return string(bodyBytes)
}

// Copied from https://github.com/cloudfoundry/lager/blob/master/json_redacter.go#L45
func redactValue(replaceMap map[string]string, data *interface{}) interface{} {
	if data == nil {
		return data
	}

	if a, ok := (*data).([]interface{}); ok {
		redactArray(replaceMap, &a)
	} else if m, ok := (*data).(map[string]interface{}); ok {
		redactObject(replaceMap, &m)
	} else if s, ok := (*data).(string); ok {
		if replaceValue, found := replaceMap[s]; found {
			(*data) = replaceValue
		}
	}
	return (*data)
}

func redactArray(replaceMap map[string]string, data *[]interface{}) {
	for i := range *data {
		redactValue(replaceMap, &((*data)[i]))
	}
}

func redactObject(replaceMap map[string]string, data *map[string]interface{}) {
	for k, v := range *data {
		replaceValue, found := replaceMap[k]
		if found {
			(*data)[k] = replaceValue
		}

		if (*data)[k] != replaceValue {
			(*data)[k] = redactValue(replaceMap, &v)
		}
	}
}

func callerFnName() string {
	pc, _, _, _ := runtime.Caller(2)
	parts := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	n := len(parts)
	funcName := parts[n-1]
	if parts[n-2][0] == '(' {
		funcName = parts[n-2] + "." + funcName
	}
	return funcName
}

// Of the form "2006-01-02 15:04:05".
func mustParse(s string) time.Time {
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestNewClient(t *testing.T) {
	t.Run("with WithClient, the client is overridden", func(t *testing.T) {
		expect := &http.Client{}
		got := NewClient("token", WithClient(expect))
		assert.Same(t, expect, got.client)
	})
	t.Run("with WithClient and no transport, the given client has its transport changed", func(t *testing.T) {
		c := &http.Client{}
		assert.Nil(t, c.Transport)

		got := NewClient("token", WithClient(c))
		assert.IsType(t, transport{}, got.client.Transport)
	})
	t.Run("with WithClient and some transport, the given client has its transport changed", func(t *testing.T) {
		c := &http.Client{}
		c.Transport = http.DefaultTransport.(*http.Transport).Clone()
		assert.NotNil(t, c.Transport)

		got := NewClient("token", WithClient(c))
		assert.IsType(t, transport{}, got.client.Transport)
	})
	t.Run("without WithClient, the X-Api-Key is set on requests", func(t *testing.T) {
		c := NewClient("token")
		req, _ := http.NewRequest("GET", "foo://foo", http.NoBody)
		_, _ = c.client.Do(req)
		assert.Equal(t, "token", req.Header.Get("X-Api-Key"), "the header X-Api-Key is expected to be set")
	})
	t.Run("with WithClient, the X-Api-Key is set on requests", func(t *testing.T) {
		c := &http.Client{}
		c.Transport = http.DefaultTransport.(*http.Transport).Clone()
		cc := NewClient("token", WithClient(c))
		req, _ := http.NewRequest("GET", "foo://foo", http.NoBody)
		_, _ = cc.client.Do(req)
		assert.Equal(t, "token", req.Header.Get("X-Api-Key"), "the header X-Api-Key is expected to be set")
	})
	t.Run("WithServer overrides the server", func(t *testing.T) {
		expect := &http.Client{}
		got := NewClient("token", WithClient(expect))
		assert.Same(t, expect, got.client)
	})
}

func TestIs(t *testing.T) {
	tests := []struct {
		givenErr    error
		givenStatus int
		want        bool
	}{
		{
			givenErr: nil,
			want:     false,
		},
		{
			givenErr:    nil,
			givenStatus: 200,
			want:        false,
		},
		{
			givenErr: fmt.Errorf("some err"),
			want:     false,
		},
		{
			givenErr: ErrClockify{Message: "Full authentication is required to access this resource", Code: 1000, Status: 401},
			want:     false,
		},
		{
			givenErr:    ErrClockify{Message: "Full authentication is required to access this resource", Code: 1000, Status: 401},
			givenStatus: 401,
			want:        true,
		},
		{
			givenErr: ErrClockify{Message: "", Status: 403},
			want:     false,
		},
		{
			givenErr:    ErrClockify{Message: "", Status: 403},
			givenStatus: 403,
			want:        true,
		},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := Is(tt.givenErr, tt.givenStatus)
			assert.Equal(t, tt.want, got)
		})
	}
}
