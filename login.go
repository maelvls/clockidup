package main

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"

	"github.com/maelvls/clockidup/logutil"
)

// Should only be called on a non-empty token.
func tokenWorks(token string) bool {
	logutil.Debugf("checking whether the token '%s' works", token)
	clockify := NewClockify(token, nil)
	_, err := clockify.Workspaces()
	return err == nil
}

func askToken(existing Config) (new Config, err error) {
	logutil.Infof("the API token is available at %s", logutil.Green("https://clockify.me/user/settings"))

	// Check whether the existing token already works or not and ask the
	// user if it already works.
	if existing.Token != "" && tokenWorks(existing.Token) {
		var override bool
		err = survey.Ask([]*survey.Question{{
			Name: "override",
			Prompt: &survey.Confirm{
				Message: "Existing token seems to be valid. Override it?",
			}}}, &override,
		)
		if err != nil {
			return Config{}, err
		}
		if !override {
			return existing, nil
		}
	}

	token := existing.Token
	err = survey.Ask([]*survey.Question{{
		Name:   "token",
		Prompt: &survey.Password{Message: "Clockify API token"}, Validate: func(ans interface{}) error {
			if ans == "" {
				return fmt.Errorf("the token cannot be empty")
			}
			return nil
		},
	}}, &token)
	if err != nil {
		return Config{}, err
	}

	if !tokenWorks(token) {
		return Config{}, fmt.Errorf("token seems to be invalid")
	}

	return Config{
		Token: token,
	}, nil
}

// The existing is the configuration loaded from ~/.config/clockidup.yaml.
func askWorkspace(existing Config) (new Config, err error) {
	logutil.Debugf("existing workspace: %s", existing.Workspace)
	clockify := NewClockify(existing.Token, nil)
	workspaces, err := clockify.Workspaces()
	if err != nil {
		return Config{}, fmt.Errorf("Failed to list workspaces: %s", err)
	}

	var names []string
	var selected string
	for _, w := range workspaces {
		names = append(names, w.Name)
	}

	err = survey.AskOne(&survey.Select{
		Options: names,
		Default: existing.Workspace,
		Message: "Choose a workspace",
	}, &selected)
	if err != nil {
		return Config{}, err
	}

	existing.Workspace = selected
	return existing, nil
}

func selectWorkspace(workspaces []Workspace) (string, error) {
	var workspaceNames []string
	var workspace string
	for _, workspace := range workspaces {
		workspaceNames = append(workspaceNames, workspace.Name)
	}
	err := survey.Ask([]*survey.Question{{
		Name: "workspace",
		Prompt: &survey.Select{
			Message: "Select a Workspace:",
			Options: workspaceNames,
		}}}, &workspace,
	)

	if err != nil {
		return "", err
	}
	return workspace, nil
}
