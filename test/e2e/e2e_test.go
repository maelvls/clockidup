package e2e

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/sethgrid/gencurl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var record = os.Getenv("RECORD") != ""

func Test_CLI(t *testing.T) {
	cli := withBinary(t)

	// For these end-to-end tests, we don't want to always rely on the "live"
	// Clockidup API since it is only available by one person. We thus test
	// clockidup using a proxy that records all the interations and replays
	// them.
	tr := withReplayTransport(t)
	port := freePort()
	server := &http.Server{
		Addr: ":" + port,
		Handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			r.Host = "api.clockify.me"
			r.URL.Scheme = "https"
			r.URL.Host = "api.clockify.me"

			resp, err := tr.RoundTrip(r)
			require.NoError(t, err)
			defer resp.Body.Close()

			for key, value := range resp.Header {
				for _, v := range value {
					rw.Header().Set(key, v)
				}
			}
			rw.WriteHeader(resp.StatusCode)

			t.Logf("proxy forwarded request: %s [%d]", gencurl.FromRequest(r), resp.StatusCode)

			io.Copy(rw, resp.Body)
		}),
	}
	go func() {
		err := server.ListenAndServe()
		assert.NoError(t, err)
	}()
	for !canConnect(":" + port) {
		t.Log("waiting for proxy to start")
		time.Sleep(10 * time.Millisecond)
	}

	t.Run("--help", func(t *testing.T) {
		cmd := exec.Command(cli, "--help")
		cli := startWith(t, cmd).Wait()

		output := contents(cli.Output)
		assert.Contains(t, output, "SYNOPSYS")
		assert.Equal(t, 0, cli.ProcessState.ExitCode())
	})

	t.Run("no token available", func(t *testing.T) {
		home := withConfigInFakeHome(t, "")
		cmd := exec.Command(cli, "--server=http://localhost:"+port, "2021-07-03")
		cmd.Env = append(cmd.Env, "HOME="+home)
		cli := startWith(t, cmd).Wait()

		output := contents(cli.Output)
		assert.Equal(t, 1, strings.Count(output, "\n"))
		assert.Contains(t, output, heredoc.Doc(`
			not logged in, run 'clockidup login' first or use --token
		`))

		assert.Equal(t, 1, cli.ProcessState.ExitCode())
	})

	t.Run("configuration file is not a correct YAML file", func(t *testing.T) {
		home := withConfigInFakeHome(t, "\n")
		cmd := exec.Command(cli, "--server=http://localhost:"+port, "2021-07-03")
		cmd.Env = append(cmd.Env, "HOME="+home)
		cli := startWith(t, cmd).Wait()

		output := contents(cli.Output)
		assert.Equal(t, 1, strings.Count(output, "\n"))
		assert.Regexp(t, regexp.MustCompile(heredoc.Doc(`
			could not load config: decoding '.*/.config/clockidup.yml' from YAML: EOF
		`)), output)

		assert.Equal(t, 1, cli.ProcessState.ExitCode())
	})

	t.Run("valid token present in .config/clockidup.yml", func(t *testing.T) {
		home := withConfigInFakeHome(t, heredoc.Docf(`
			# This is the YAML file in ~/.config/clockidup.yml.
			token: "%s"
			workspace: workspace-1
		`, withToken(t)))

		cmd := exec.Command(cli, "--server=http://localhost:"+port, "2021-07-03")
		cmd.Env = append(cmd.Env, "HOME="+home)
		cli := startWith(t, cmd).Wait()

		output := contents(cli.Output)
		assert.Equal(t, heredoc.Doc(`
			2021-07-03:
			- [.5] project-1: Unit-test of clockidup, work with project and task
			- [.5] project-1: Some work with project but no task
			- [.5] : Work with no project
		`), output)

		assert.Equal(t, 0, cli.ProcessState.ExitCode())
	})

	t.Run("--token overrides the token present in .config/clockidup.yml", func(t *testing.T) {
		home := withConfigInFakeHome(t, heredoc.Docf(`
			# This is the YAML file in ~/.config/clockidup.yml.
			token: "%s"
			workspace: workspace-1
		`, withToken(t)))

		cmd := exec.Command(cli, "--server=http://localhost:"+port, "--token=foo", "2021-07-03")
		cmd.Env = append(cmd.Env, "HOME="+home)
		cli := startWith(t, cmd).Wait()

		output := contents(cli.Output)
		assert.Contains(t, output, heredoc.Doc(`
			existing token does not work, run the 'login' command first or use --token
		`))

		assert.Equal(t, 1, cli.ProcessState.ExitCode())
	})

	t.Run("--workspace overrides the workspace present in .config/clockidup.yml", func(t *testing.T) {
		home := withConfigInFakeHome(t, heredoc.Docf(`
			# This is the YAML file in ~/.config/clockidup.yml.
			token: "%s"
			workspace: workspace-1
		`, withToken(t)))

		cmd := exec.Command(cli, "--server=http://localhost:"+port, "--workspace=workspace-2", "2021-07-03")
		cmd.Env = append(cmd.Env, "HOME="+home)
		cli := startWith(t, cmd).Wait()

		output := contents(cli.Output)
		assert.Equal(t, heredoc.Doc(`
			2021-07-03:
		`), output)

		assert.Equal(t, 0, cli.ProcessState.ExitCode())
	})

	t.Run("wrong day format", func(t *testing.T) {
		cmd := exec.Command(cli, "--server=http://localhost:"+port, "--token="+withToken(t), "--workspace=workspace-1", "NOT_A_DAY")
		home := withConfigInFakeHome(t, "")
		cmd.Env = append(cmd.Env, "HOME="+home)
		cli := startWith(t, cmd).Wait()

		output := contents(cli.Output)
		assert.Equal(t, "\x1b[0;31merror\x1b[0m: "+heredoc.Doc(`
		'NOT_A_DAY' is not a valid date. The date must of the form:

		    2021-12-31
		    today
		    yesterday
		    three days ago
		    3 days ago
		    wednesday
		    monday
		    last tuesday

		See the documentation at https://github.com/tj/go-naturaldate#examples.
		`), output)
		assert.Equal(t, 1, cli.ProcessState.ExitCode())
	})
}

func withConfigInFakeHome(t *testing.T, clockidupConfigYAML string) string {
	home, err := ioutil.TempDir("", "clockidup-e2e-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		// os.RemoveAll(home)
	})
	os.MkdirAll(home+"/.config", 0755)

	if len(clockidupConfigYAML) > 0 {
		require.NoError(t, ioutil.WriteFile(home+"/.config/clockidup.yml", []byte(clockidupConfigYAML), 0755))
	}

	return home
}
