package main

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"

	"github.com/maelvls/clockidup/clockify"
	"github.com/maelvls/clockidup/logutil"
)

func checkToken(token string, client func(token string) clockifyClient) (bool, error) {
	_, err := client(token).Workspaces()
	if clockify.Is(err, 401) || clockify.Is(err, 403) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func promptToken(existingToken string, client func(token string) clockifyClient) (newToken string, _ error) {
	logutil.Infof("the API token is available at %s", logutil.Green("https://clockify.me/user/settings"))

	// Check whether the existing token already works or not and ask the
	// user if it already works.
	works, err := checkToken(existingToken, client)
	if err != nil {
		return "", err
	}
	if works {
		var override bool
		err = survey.Ask([]*survey.Question{{
			Name: "override",
			Prompt: &survey.Confirm{
				Message: "Existing token seems to be valid. Override it?",
			}}}, &override,
		)
		if err != nil {
			return "", err
		}
		if !override {
			return existingToken, nil
		}
	}

	token := existingToken
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
		return "", err
	}

	works, err = checkToken(token, client)
	if err != nil {
		return "", err
	}
	if !works {
		return "", fmt.Errorf("token seems to be invalid")
	}

	return token, nil
}

// The existing is the configuration loaded from ~/.config/clockidup.yaml.
func promptWorkspace(client clockifyClient, existing Config) (new Config, err error) {
	logutil.Debugf("existing workspace: %s", existing.Workspace)

	workspaces, err := client.Workspaces()
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

func promptWorkspaceSelect(workspaces []clockify.Workspace) (string, error) {
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
