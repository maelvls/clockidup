package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/dnaeon/go-vcr/cassette"
	"github.com/dnaeon/go-vcr/recorder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClockify_Workspaces(t *testing.T) {
	withLiveSlack := os.Getenv("LIVE") != ""
	token := "redacted-token"
	mode := recorder.ModeReplaying
	if withLiveSlack {
		token = MustGetenv("CLOCKIFY_TOKEN")
		mode = recorder.ModeRecording
	}

	rec, err := recorder.NewAsMode("fixtures/clockify", mode, http.DefaultTransport)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = rec.Stop()
	}()

	rec.AddFilter(func(i *cassette.Interaction) error {
		i.Request.Body = RedactJSON(i.Request.Body, "userId", "fake-user-id", "id", "fake-id", "targetId", "fake-target-id")
		i.Response.Body = RedactJSON(i.Response.Body, "userId", "fake-user-id", "id", "fake-id", "targetId", "fake-target-id")
		if i.Request.Headers.Get("X-Api-Key") != "" {
			i.Request.Headers.Set("X-Api-Key", "redacted-token")
		}
		return nil
	})

	t.Run("when authentified", func(t *testing.T) {
		clockify := NewClockify(token, &http.Client{Transport: rec})
		got, gotErr := clockify.Workspaces()
		require.NoError(t, gotErr)
		assert.Equal(t, []Workspace{{
			ID:   "fake-id",
			Name: "Maël Valais's workspace",
			HourlyRate: HourlyRate{Amount: 0,
				Currency: "USD"},
			Memberships: []Memberships{{UserID: "fake-user-id",
				HourlyRate:       interface{}(nil),
				CostRate:         interface{}(nil),
				TargetID:         "fake-target-id",
				MembershipType:   "WORKSPACE",
				MembershipStatus: "ACTIVE"}},
			WorkspaceSettings: WorkspaceSettings{TimeRoundingInReports: false,
				OnlyAdminsSeeBillableRates: true,
				OnlyAdminsCreateProject:    true,
				OnlyAdminsSeeDashboard:     false,
				DefaultBillableProjects:    true,
				LockTimeEntries:            interface{}(nil),
				Round: Round{Round: "Round to nearest",
					Minutes: "15"},
				ProjectFavorites:                   true,
				CanSeeTimeSheet:                    false,
				CanSeeTracker:                      true,
				ProjectPickerSpecialFilter:         false,
				ForceProjects:                      false,
				ForceTasks:                         false,
				ForceTags:                          false,
				ForceDescription:                   false,
				OnlyAdminsSeeAllTimeEntries:        false,
				OnlyAdminsSeePublicProjectsEntries: false,
				TrackTimeDownToSecond:              true,
				ProjectGroupingLabel:               "client",
				AdminOnlyPages:                     []interface{}{},
				AutomaticLock:                      interface{}(nil),
				OnlyAdminsCreateTag:                false,
				OnlyAdminsCreateTask:               false,
				TimeTrackingMode:                   "DEFAULT",
				IsProjectPublicByDefault:           true},
			ImageURL:                "",
			FeatureSubscriptionType: interface{}(nil)}},
			got)
	})

	t.Run("when not authentified", func(t *testing.T) {
		clockify := NewClockify("wrong-token", &http.Client{Transport: rec})
		got, gotErr := clockify.Workspaces()

		require.EqualError(t, gotErr, "Full authentication is required to access this resource")
		assert.Equal(t, []Workspace(nil), got)
	})
}

func MustGetenv(v string) string {
	res := os.Getenv(v)
	if res == "" {
		panic(fmt.Errorf("The env var %s is not set or is empty. Did you forget to 'source .envrc'?", v))
	}
	return res
}

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
