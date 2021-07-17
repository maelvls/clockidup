package e2e

import (
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/dnaeon/go-vcr/cassette"
	"github.com/dnaeon/go-vcr/recorder"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withReplayTransport(t *testing.T) *recorder.Recorder {
	mode := recorder.ModeReplaying
	if record {
		mode = recorder.ModeRecording
	}

	rec, err := recorder.NewAsMode("fixtures/"+strings.ToLower(t.Name()), mode, http.DefaultTransport)
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, rec.Stop())
	})

	// The response's token is only filtered in recording mode (RECORD=1). The
	// filter does nothing in replay mode.
	rec.AddFilter(func(i *cassette.Interaction) error {
		if i.Request.Headers.Get("X-Api-Key") == withToken(t) {
			i.Request.Headers.Set("X-Api-Key", "redacted-token")
		}
		return nil
	})

	return rec
}

func withToken(t *testing.T) (token string) {
	if record {
		return mustGetenv(t, "CLOCKIFY_TOKEN")
	}
	return "redacted-token"
}

func mustGetenv(t *testing.T, v string) string {
	res := os.Getenv(v)
	if res == "" {
		t.Errorf("The env var %s is not set or is empty. Did you forget to 'export %s=value' or to add it to your '.envrc'?", v, v)
		t.FailNow()
	}
	return res
}

// Returns the path to the built CLI. Better call it only once since it
// needs to recompile.
func withBinary(t *testing.T) string {
	start := time.Now()

	bincli, err := gexec.Build("github.com/maelvls/clockidup")
	require.NoError(t, err)

	t.Cleanup(func() {
		gexec.Terminate()
		gexec.CleanupBuildArtifacts()
	})

	t.Logf("compiling binary took %v, path: %s", time.Since(start).Truncate(time.Second), bincli)
	return bincli
}

type e2ecmd struct {
	*exec.Cmd
	Output *gbytes.Buffer // Both stdout and stderr.
	T      *testing.T
}

func (cmd *e2ecmd) Wait() *e2ecmd {
	_ = cmd.Cmd.Wait()
	return cmd
}

// Runs the passed command and make sure SIGTERM is called on cleanup. Also
// dumps stderr and stdout using log.Printf.
func startWith(t *testing.T, cmd *exec.Cmd) *e2ecmd {
	buff := gbytes.NewBuffer()
	cmd.Stdout = createWriterLoggerStr("stdout", buff)
	cmd.Stderr = createWriterLoggerStr("stderr", buff)

	err := cmd.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = cmd.Process.Signal(syscall.SIGTERM)
	})

	return &e2ecmd{Cmd: cmd, Output: buff, T: t}
}

func contents(f io.Reader) string {
	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

// createWriterLoggerStr returns a writer that behaves like w except that it
// logs (using log.Printf) each write to standard error, printing the
// prefix and the data written as a string.
//
// Pretty much the same as iotest.NewWriterLogger except it logs strings,
// not hexadecimal jibberish.
func createWriterLoggerStr(prefix string, w io.Writer) io.Writer {
	return &writeLogger{prefix, w}
}

type writeLogger struct {
	prefix string
	w      io.Writer
}

func (l *writeLogger) Write(p []byte) (n int, err error) {
	n, err = l.w.Write(p)
	if err != nil {
		log.Printf("%s %s: %v", l.prefix, string(p[0:n]), err)
	} else {
		log.Printf("%s %s", l.prefix, string(p[0:n]))
	}
	return
}

func freePort() string {
	port, err := freeport.GetFreePort()
	if err != nil {
		panic(err)
	}
	return strconv.Itoa(port)
}

// When given a io.Reader, checks that the given string eventuall appears.
// A bit like Testify's require.Eventually but works directly on a
// io.Reader.
//
// Commented out since I'm not using it anymore.

func eventuallyEqual(t *testing.T, expected string, got *gbytes.Buffer, msgsAndArgs ...interface{}) {
	expectedBuffer := gbytes.Say(expected)

	match := func() func() bool {
		return func() bool {
			ok, err := expectedBuffer.Match(got)
			assert.NoError(t, err)

			return ok
		}
	}

	if !assert.Eventually(t, match(), 2*time.Second, 100*time.Millisecond, msgsAndArgs...) {
		t.Errorf(expectedBuffer.FailureMessage(expected))
	}
}

// Addr is of the form ip:port. Only supports IPs, not hostnames. We check
// that we can connect() to this ip:port by running the TCP handshake
// SYN-SYNACK-ACK until the the connection is ESTABLISHED.
func canConnect(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	return true
}
