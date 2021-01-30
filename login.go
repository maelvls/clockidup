package main

import (
	"fmt"
	"net/http"

	"github.com/AlecAivazis/survey/v2"

	"github.com/maelvls/clockidup/logutil"
)

func tokenWorks(token string) bool {
	clockify := NewClockify(token, http.DefaultClient)
	_, err := clockify.Workspaces()
	return err != nil
}

func askToken(existing Config) (new Config, err error) {
	logutil.Infof("the API token is available at %s", logutil.Green("https://clockify.me/user/settings"))
	token := existing.Token

	// Check whether the existing token already works or not and ask the
	// user if it already works.

	if tokenWorks(existing.Token) {
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
