package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/maelvls/standup/logutil"
)

const confPath = ".config/standup.yml"

var (
	tokenFlag = flag.String("token", "", "the Clockify API token")
	debugFlag = flag.Bool("debug", false, "show debug output")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] (yesterday|today|login)\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	logutil.EnableDebug = *debugFlag

	conf, err := loadConfig(confPath)
	if err != nil {
		logutil.Errorf("could not load config: %s", err)
		os.Exit(1)
	}

	var startT, endT time.Time
	switch flag.Arg(0) {
	case "yesterday":
		startT = time.Now().AddDate(0, 0, -2)
		endT = time.Now().AddDate(0, 0, -1)
	case "today":
		startT = time.Now().AddDate(0, 0, -1)
		endT = time.Now().AddDate(0, 0, 0)
	case "login":
		conf, err := login(conf)
		if err != nil {
			logutil.Errorf("login failed: %s", err)
			os.Exit(1)
		}

		logutil.Infof("you are logged in!")

		err = saveConfig(confPath, conf)
		if err != nil {
			logutil.Errorf("saving configuration: %s", err)
			os.Exit(1)
		}

		logutil.Debugf("config: %v")

		os.Exit(0)
	case "":
		flag.Usage()
		os.Exit(1)
	}

	token := conf.Token
	if *tokenFlag != "" {
		token = *tokenFlag
	}
	if token == "" {
		logutil.Errorf("not logged in, run the 'login' command first or use --token")
		os.Exit(1)
	}

	start := time.Date(startT.Year(), startT.Month(), startT.Day(), 0, 0, 0, 0, startT.Location())
	end := time.Date(endT.Year(), endT.Month(), endT.Day(), 0, 0, 0, 0, endT.Location())

	workspaces, err := clockifyWorkspaces(token)
	if err != nil {
		logutil.Errorf("%s", err)
		os.Exit(1)
	}
	if len(workspaces) == 0 {
		logutil.Errorf("no workspace found")
		os.Exit(1)
	}

	workspace := workspaces[0]
	userID := workspace.Memberships[0].UserID

	timeEntries, err := clockifyTimeEntries(token, workspaces[0].ID, userID, start, end)
	if err != nil {
		logutil.Errorf("%s", err)
		os.Exit(1)
	}

	projects, err := clockifyProjects(token, workspaces[0].ID)
	if err != nil {
		logutil.Errorf("%s", err)
		os.Exit(1)
	}
	projectMap := make(map[string]Project)
	for _, p := range projects {
		projectMap[p.ID] = p
	}

	// Deduplicate activities: when two activities have the same
	// description, I merge them by summing up their duration. The key of
	// the entriesSeen map is the description string.
	type MergedEntry struct {
		Project     string
		Description string
		Duration    time.Duration
	}
	entriesSeen := make(map[string]*MergedEntry)
	var mergedEntries []*MergedEntry
	for _, entry := range timeEntries {
		existing, found := entriesSeen[entry.Description]
		if found {
			existing.Duration += entry.TimeInterval.End.Sub(entry.TimeInterval.Start)
			continue
		}

		new := MergedEntry{
			Project:     projectMap[entry.ProjectID].Name,
			Description: entry.Description,
			Duration:    entry.TimeInterval.End.Sub(entry.TimeInterval.Start),
		}
		mergedEntries = append(mergedEntries, &new)
		entriesSeen[entry.Description] = &new
	}

	for i := range mergedEntries {
		entry := mergedEntries[len(mergedEntries)-i-1]
		fmt.Printf("- [%.2f] %s: %s\n", entry.Duration.Hours(), entry.Project, entry.Description)
	}
}
