package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/mgutz/ansi"
	"github.com/tj/go-naturaldate"

	"github.com/maelvls/clockidup/clockify"
	"github.com/maelvls/clockidup/logutil"
)

const (
	confPath  = ".config/clockidup.yml"
	layoutISO = "2006-01-02"
)

var (
	debugFlag     = flag.Bool("debug", false, "Show debug output, including the HTTP requests.")
	onlyBillable  = flag.Bool("billable", false, "Only print the entries that are billable.")
	tokenFlag     = flag.String("token", "", "The Clockify API token.")
	workspaceFlag = flag.String("workspace", "", "Workspace Name to use.")
	serverFlag    = flag.String("server", "https://api.clockify.me", "(For testing purposes) Override the Clockidup API endpoint.")

	// The 'version' var is set during build, using something like:
	//  go build  -ldflags "-X main.version=$(git describe --tags)".
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
    clockidup select
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

Note that you can at any time change the Clockify workspace:

    {{ cmd "clockidup select" }}

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

	logutil.Debugf("config loaded from ~/.config/clockidup.yml: %#s", conf)

	var day time.Time
	switch flag.Arg(0) {
	case "login":
		conf.Token, err = promptToken(conf.Token, checkToken(*serverFlag))
		if err != nil {
			return fmt.Errorf("login failed: %s", err)
		}
		logutil.Infof("you are logged in!")

		client := clockify.NewClient(conf.Token, clockify.WithServer(*serverFlag))
		conf, err = askWorkspace(client, conf)
		if err != nil {
			return fmt.Errorf("unable to set workspace: %s", err)
		}
		logutil.Infof("Set workspace to: %s", conf.Workspace)

		err = saveConfig(confPath, conf)
		if err != nil {
			return fmt.Errorf("saving configuration: %s", err)
		}
		logutil.Debugf("config: %+v", conf)
		return nil
	case "select":
		client := clockify.NewClient(conf.Token, clockify.WithServer(*serverFlag))
		conf, err = askWorkspace(client, conf)
		if err != nil {
			return fmt.Errorf("unable to set workspace: %s", err)
		}
		logutil.Infof("set workspace to: %s", conf.Workspace)

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
		logutil.Errorf("not logged in, run 'clockidup login' first or use --token")
		os.Exit(1)
	}
	works, err := checkToken(*serverFlag)(token)
	if err != nil {
		logutil.Errorf("while checking that your token is still valid: %s", err)
		os.Exit(1)
	}
	if !works {
		logutil.Errorf("existing token does not work, run the 'login' command first or use --token")
		os.Exit(1)
	}

	workspaceName := workspaceFlag
	if workspaceName == "" {
		workspaceName = conf.Workspace
	}
	if workspaceName == "" {
		logutil.Errorf("no workspace selected, use 'clockidup select' or use --workspace")
		os.Exit(1)
	}

	clockify := clockify.NewClient(token, clockify.WithServer(*serverFlag))

	mergedEntries, err := timeEntriesForDay(clockify, time.Now, workspaceName, day)
	if err != nil {
		return fmt.Errorf("while merging entries: %w", err)
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
		// decimal to the closest neighbor. We also remove the leading zero to
		// distinguish "small" amounts (e.g. 0.5) from larger amounts (e.g.
		// 2.0). For example:
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
