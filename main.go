package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/mgutz/ansi"
	"github.com/tj/go-naturaldate"

	"github.com/maelvls/clockidup/logutil"
)

const (
	confPath  = ".config/clockidup.yml"
	layoutISO = "2006-01-02"
)

var (
	tokenFlag     = flag.String("token", "", "The Clockify API token.")
	workspaceFlag = flag.String("workspace", "", "Workspace Name to use.")
	debugFlag     = flag.Bool("debug", false, "Show debug output, including the HTTP requests.")
	onlyBillable  = flag.Bool("billable", false, "Only print the entries that are billable.")

	// The 'version' var is set during build, using something like:
	//  go build  -ldflags"-X main.version=$(git describe --tags)".
	// Note: "version", is set automatically by goreleaser.
	version = ""
)

const help = `{{- if .Extended -}}
{{ section "NAME" }}

clockidup helps you generating your standup entry using the time entries from
{{ url "https://clockify.me" }}.

{{ end -}}
{{ section "SYNOPSYS" }}

    clockidup login
    clockidup [--billable] {{ url "DAY" }}
    clockidup version

where {{ url "DAY" }} is of the form:

    today
    yesterday
    thursday
    "2 days ago"
    2021-01-28

{{- if .Extended }}

{{ section "HOW TO USE IT" }}

To start, log in:

    {{ cmd "clockidup login" }}

Each "project" in Clockify will show up as a prefix of your time entries. For
example, imagining that you have an entry that you created with clockify-cli:

    clockify-cli in prod/cert-manager --when=now "#3444: continue dataforcertificate unit test"
    clockify-cli out

where {{ url "prod/cert-manager" }} is a Clockify project. clockidup will show:

    {{ cmd "clockidup today" }}
    {{ out "Friday:" }}
    {{ out "- [1.2] prod/cert-manager: #3444: continue dataforcertificate unit test" }}
      <---> <--------------->  <------------------------------------------>
    duration     project                        entry text

You may only want the "billable" entries to be displayed by clockidup:

    {{ cmd "clockidup --billable today" }}

You can also use Clockify tasks. Like projects, there are displayed as a prefix
to the entry text:

    {{ out "- [1.2] prod/cert-manager: big refactoring for the v2: rm large defers" }}
      <---> <--------------->  <------------------------> <-------------->
     duration     project                 task               entry text

{{ section "CONFIG FILE" }}

The auth token is saved to {{ url "~/.config/clockidup.yml" }}. The file looks
like this:

    token: your-clockify-auth-token

{{- end }}
{{- if not .Extended }}

More help is available with the command {{ yel "clockidup help" }}.

{{- end }}

{{ section "OPTIONS" }}

`

func main() {
	printHelp := func(extended bool) func() {
		return func() {
			t, err := template.New("help").Funcs(map[string]interface{}{
				"section": ansi.ColorFunc("black+hb"),
				"url":     ansi.ColorFunc("white+u"),
				"grey":    ansi.ColorFunc("white+d"),
				"yel":     ansi.ColorFunc("yellow"),
				"cmd": func(cmd string) string {
					return ansi.ColorFunc("white+d")("% ") + ansi.ColorFunc("yellow+b")(cmd)
				},
				"out": ansi.ColorFunc("white+d"),
			}).Parse(help)
			if err != nil {
				log.Fatal(err)
			}

			err = t.Execute(flag.CommandLine.Output(), struct{ Extended bool }{
				Extended: extended,
			})
			if err != nil {
				log.Fatal(err)
			}
			flag.PrintDefaults()
		}
	}
	flag.Usage = printHelp(false)
	flag.Parse()

	if *debugFlag {
		logutil.EnableDebug = true
	}

	err := Run(*tokenFlag, *workspaceFlag, printHelp)
	if err != nil {
		logutil.Errorf(err.Error())
		os.Exit(1)
	}
}

func Run(tokenFlag string, workspaceFlag string, printHelp func(bool) func()) error {
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
	case "version":
		if version == "" {
			version, err = versionUsingGo()
			if err != nil {
				return fmt.Errorf("binary not built with a version, and error while fetching the error using 'go version -m': %w", err)
			}
		}

		fmt.Printf("%s\n", version)
		return nil
	case "help":
		printHelp(true)()
		os.Exit(0)
	case "":
		printHelp(false)()
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

	if token == "" {
		logutil.Errorf("no configuration found in ~/.config/clockidup.yml, run 'clockidup login' first or use --token")
		os.Exit(1)
	}
	if !tokenWorks(token) {
		logutil.Errorf("existing token does not work, run the 'login' command first or use --token")
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
		return fmt.Errorf("no workspaces found")
	}

	workspaceName := workspaceFlag
	if workspaceName == "" {
		workspaceName = conf.Workspace
	}

	workspace, err := clockify.FindWorkspace(workspaces, workspaceName)
	if err != nil {
		return fmt.Errorf("%s", err)
	}
	userID := workspace.Memberships[0].UserID

	timeEntries, err := clockify.TimeEntries(workspace.ID, userID, start, end)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	projects, err := clockify.Projects(workspace.ID)
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

// Use Go installed on the system to get the git tag of the running Go
// executable by running:
//
//   go version -m /path/to/binary
//
// and by parsing the "mod" line. For example, we want to show "v0.3.0"
// from the following:
//
//   /home/mvalais/go/bin/clockidup: go1.16.3
//      path    github.com/maelvls/clockidup
//      mod     github.com/maelvls/clockidup      v0.3.0  h1:84sL4RRZKsyJgSs8KFyE6ykSjtNk79bBVa0ZgC48Kpw=
//      dep     github.com/AlecAivazis/survey/v2  v2.2.12 h1:5a07y93zA6SZ09gOa9wLVLznF5zTJMQ+pJ3cZK4IuO8=
func versionUsingGo() (string, error) {
	bin, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("while trying to find the clockidup binary path: %s", err)
	}
	cmd := exec.Command("go", "version", "-m", bin)

	bytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("while slurping stdout from 'go version -m %s': %s", bin, err)
	}

	if cmd.ProcessState.ExitCode() != 0 {
		return "", fmt.Errorf("running 'go version -m %s': %s", bin, err)
	}

	// We want to be parsing the following line:
	//
	//       mod     github.com/maelvls/clockidup     v0.3.0  h1:84sL4RRZKsyJgSs8KFyE6ykSjtNk79bBVa0ZgC48Kpw=
	//   <-->   <-->                             <---><---->
	//   tab    tab                               tab  m[0][1]

	regStr := `mod\s*[^\s]*\s*([^\s]*)`
	//                        <------>
	//                         m[0][1]

	reg := regexp.MustCompile(regStr)

	m := reg.FindAllStringSubmatch(string(bytes), 1)
	if len(m) < 1 || len(m[0]) < 2 {
		return "", fmt.Errorf("'go version -m %s' did not return a string of the form '%s':\nmatches: %v\nstdout: %s", bin, regStr, m, string(bytes))
	}

	return m[0][1], nil
}
