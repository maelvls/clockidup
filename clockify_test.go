package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/dnaeon/go-vcr/cassette"
	"github.com/dnaeon/go-vcr/recorder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var record = os.Getenv("RECORD") != ""

func TestClockify_Workspaces(t *testing.T) {
	tr := withReplayTransport(t)

	t.Run("when authenticated", func(t *testing.T) {
		clockify := NewClockify(withToken(t), &http.Client{Transport: tr})

		got, gotErr := clockify.Workspaces()

		require.NoError(t, gotErr)
		assert.Equal(t, []Workspace{
			workspaceWith("workspace-1", "60e086c24f27a949c058082e", "60e086c24f27a949c058082d"),
			workspaceWith("workspace-2", "60e08781bf81bd307230c097", "60e086c24f27a949c058082d"),
		}, got)
	})

	t.Run("when not authenticated", func(t *testing.T) {
		clockify := NewClockify("invalid-token", &http.Client{Transport: tr})

		got, gotErr := clockify.Workspaces()

		assert.Equal(t, []Workspace(nil), got)
		require.EqualError(t, gotErr, "Full authentication is required to access this resource")
	})
}

func TestFindWorkspace(t *testing.T) {
	tests := map[string]struct {
		givenWorkspaces []Workspace
		givenName       string
		want            Workspace
		wantFound       bool
	}{
		"workspace exists": {
			givenWorkspaces: []Workspace{
				workspaceWith("workspace-1", "60e086c24f27a949c058082e", "60e086c24f27a949c058082d"),
				workspaceWith("workspace-2", "60e08781bf81bd307230c097", "60e086c24f27a949c058082d"),
			},
			givenName: "workspace-2",
			wantFound: true,
			want:      workspaceWith("workspace-2", "60e08781bf81bd307230c097", "60e086c24f27a949c058082d"),
		},
		"workspace does not exist": {
			givenWorkspaces: []Workspace{
				workspaceWith("workspace-1", "60e086c24f27a949c058082e", "60e086c24f27a949c058082d"),
				workspaceWith("workspace-2", "60e08781bf81bd307230c097", "60e086c24f27a949c058082d"),
			},
			givenName: "workspace-3",
			wantFound: false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, gotFound := FindWorkspace(tt.givenWorkspaces, tt.givenName)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantFound, gotFound)
		})
	}
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
			IsProjectPublicByDefault:   true},
		ImageURL: "",
	}
}

func withToken(t *testing.T) (token string) {
	if record {
		return MustGetenv(t, "CLOCKIFY_TOKEN")
	}
	return "redacted-token"
}

// Must only be created once.
func withReplayTransport(t *testing.T) *recorder.Recorder {
	mode := recorder.ModeReplaying
	if record {
		mode = recorder.ModeRecording
	}

	rec, err := recorder.NewAsMode("fixtures/clockify", mode, http.DefaultTransport)
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

func MustGetenv(t *testing.T, v string) string {
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
