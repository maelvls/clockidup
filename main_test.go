package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			got, gotFound := findWorkspace(tt.givenWorkspaces, tt.givenName)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantFound, gotFound)
		})
	}
}
