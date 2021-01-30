package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/dnaeon/go-vcr/recorder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClockify_Workspaces(t *testing.T) {
	tests := map[string]struct {
		want    []Workspace
		wantErr string
	}{}
	for name, _ := range tests {
		t.Run(name, func(t *testing.T) {

		})
	}

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
	defer assert.NoError(t, rec.Stop())

	t.Run("when unauthentified", func(t *testing.T) {
		clockify := NewClockify(token, &http.Client{Transport: rec})
		got, gotErr := clockify.Workspaces()

		require.EqualError(t, gotErr, "Full authentication is required to access this resource")
		assert.Equal(t, nil, got)
	})

	t.Run("when authentified", func(t *testing.T) {
		clockify := NewClockify(token, &http.Client{Transport: rec})
		got, gotErr := clockify.Workspaces()

		require.EqualError(t, gotErr, "Full authentication is required to access this resource")
		assert.Equal(t, []Workspace{{ID: "a"}}, got)
	})
}

func MustGetenv(v string) string {
	res := os.Getenv(v)
	if res == "" {
		panic(fmt.Errorf("The env var %s is not set or is empty. Did you forget to 'source .envrc'?", v))
	}
	return res
}
