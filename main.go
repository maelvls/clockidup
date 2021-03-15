package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/tj/go-naturaldate"

	"github.com/maelvls/clockidup/logutil"
)

const (
	confPath  = ".config/standup.yml"
	layoutISO = "2006-01-02"
)

var (
	tokenFlag    = flag.String("token", "", "The Clockify API token.")
	debugFlag    = flag.Bool("debug", false, "Show debug output, including the HTTP requests.")
	onlyBillable = flag.Bool("billable", false, "Only print the entries that are billable.")

	showVersion = flag.Bool("version", false, "Print version. Note that it returns 'n/a (commit none, built on unknown)' when built with 'go get'.")
	// The 'version' var is set during build, using something like:
	//  go build  -ldflags"-X main.version=$(git describe --tags)".
	// Note: "version", "commit" and "date" are set automatically by
	// goreleaser.
	version = "n/a"
	commit  = "none"
	date    = "unknown"
)

func main() {
	flag.Parse()
	logutil.EnableDebug = *debugFlag

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, heredoc.Doc(`
            Usage:
              clockidup [options] (login | DATE)

            Examples:
              clockidup login
              clockidup yesterday
              clockidup today
              clockidup thursday
              clockidup "2 days ago"
              clockidup 2021-01-28
              clockidup --billable yesterday

            Options:
		`))
		flag.PrintDefaults()
	}

	err := Run(*tokenFlag)
	if err != nil {
		logutil.Errorf(err.Error())
		os.Exit(1)
	}
}

func Run(tokenFlag string) error {
	conf, err := loadConfig(confPath)
	if err != nil {
		return fmt.Errorf("could not load config: %s", err)
	}

	var day time.Time
	switch flag.Arg(0) {
	case "login":
		conf, err := askToken(conf)
		if err != nil {
			return fmt.Errorf("login failed: %s", err)
		}
		logutil.Infof("you are logged in!")

		err = saveConfig(confPath, conf)
		if err != nil {
			return fmt.Errorf("saving configuration: %s", err)
		}
		logutil.Debugf("config: %+v", conf)
		return nil
	case "":
		flag.Usage()
		return fmt.Errorf("a command is required, e.g. 'login' or 'yesterday'")
	default:
		day, err = time.Parse(layoutISO, flag.Arg(0))
		if err != nil {
			day, err = naturaldate.Parse(flag.Arg(0), time.Now(),
				naturaldate.WithDirection(naturaldate.Past),
			)
		}
		logutil.Debugf("day parsed: %s", day.String())
		if err != nil {
			logutil.Debugf("error parsing: %s", err)
			return fmt.Errorf(heredoc.Doc(`
				'%s' is not a valid date. The date must of the form:

				    2021-12-31
				    today
				    yesterday
				    three days ago
				    3 days ago
				    wednesday
				    monday
				    last tuesday

				See the documentation at https://github.com/tj/go-naturaldate#examples.`),
				flag.Arg(0))
		}

		if day.After(time.Now()) {
			return fmt.Errorf("cannot give a future date, %s is in the future", day)
		}
	}

	token := conf.Token
	if tokenFlag != "" {
		token = tokenFlag
	}
	if token == "" || !tokenWorks(token) {
		logutil.Errorf("not logged in, run the 'login' command first or use --token")
		os.Exit(1)
	}

	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	end := time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 0, day.Location())

	clockify := NewClockify(token, http.DefaultClient)

	workspaces, err := clockify.Workspaces()
	if err != nil {
		return fmt.Errorf("%s", err)
	}
	if len(workspaces) == 0 {
		return fmt.Errorf("no workspace found")
	}

	workspace := workspaces[0]
	userID := workspace.Memberships[0].UserID

	timeEntries, err := clockify.TimeEntries(workspaces[0].ID, userID, start, end)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	projects, err := clockify.Projects(workspaces[0].ID)
	if err != nil {
		return fmt.Errorf("%s", err)
	}
	projectMap := make(map[string]*Project)
	for i := range projects {
		proj := &projects[i]
		projectMap[proj.ID] = proj
	}

	// Find the corresponding task when the taskId is set.
	for i := range timeEntries {
		entry := &timeEntries[i]
		if entry.TaskID == "" {
			continue
		}
		task, err := clockify.Task(entry.WorkspaceID, entry.ProjectID, entry.TaskID)
		if err != nil {
			return fmt.Errorf("while fetching task for time entry '%s: %s': %s", projectMap[entry.ProjectID].Name, entry.Description, err)
		}
		entry.Description = task.Name + ": " + entry.Description
	}

	// When onlyBillable is enabled, we leave out the non-billable entries.
	selectBillable := func(entries []TimeEntry) []TimeEntry {
		var selected []TimeEntry
		for _, entry := range entries {
			if entry.Billable {
				selected = append(selected, entry)
			}
		}
		return selected
	}
	if *onlyBillable {
		timeEntries = selectBillable(timeEntries)
	}

	// Deduplicate activities: when two activities have the same
	// description, I merge them by summing up their duration. The key of
	// the entriesSeen map is the description string.
	type MergedEntry struct {
		Project     string
		Description string
		Task        string
		Duration    time.Duration
		Billable    bool
	}
	entriesSeen := make(map[string]*MergedEntry)
	var mergedEntries []*MergedEntry
	for _, entry := range timeEntries {
		existing, found := entriesSeen[entry.Description]
		if found && !entry.TimeInterval.End.IsZero() {
			existing.Duration += entry.TimeInterval.End.Sub(entry.TimeInterval.Start)
			continue
		}

		projectName := "no-project"
		if entry.ProjectID != "" {
			projectName = projectMap[entry.ProjectID].Name
		}

		// When the time entry is still "ticking" i.e., the user has not
		// stopped the timer yet, the "end" date is null. In this case, we
		// still want to have an estimation of how long this entry has been
		// going on for.
		duration := entry.TimeInterval.End.Sub(entry.TimeInterval.Start)
		if entry.TimeInterval.End.IsZero() {
			duration = time.Now().UTC().Sub(entry.TimeInterval.Start)
		}
		new := MergedEntry{
			Project:     projectName,
			Description: entry.Description,
			Duration:    duration,
		}
		mergedEntries = append(mergedEntries, &new)
		entriesSeen[entry.Description] = &new
	}

	// Print the current day e.g., "Monday" if the date is within a week in
	// the past; otherwise, print "2021-01-28".
	if day.After(time.Now().AddDate(0, 0, -6)) {
		fmt.Printf("%s:\n", day.Format("Monday"))
	} else {
		fmt.Printf("%s:\n", day.Format("2006-01-02"))
	}
	for i := range mergedEntries {
		entry := mergedEntries[len(mergedEntries)-i-1]

		// The format "%.1f" (precision = 1) rounds the 2nd digit after the
		// decimal to the closest neightbor. We also remove the leading
		// zero to distinguish "small" amounts (e.g. 0.5) from larger
		// amounts (e.g. 2.0). For example:
		//
		//  0.55 becomes ".5"
		//  0.56 becomes ".6"
		//  0.98 becomes "1.0"
		//  1.85 becomes "1.8"
		//  1.86 becomes "1.9"
		hours := fmt.Sprintf("%.1f", entry.Duration.Hours())
		hours = strings.TrimPrefix(hours, "0")
		fmt.Printf("- [%s] %s: %s\n", hours, entry.Project, entry.Description)
	}

	return nil
}
