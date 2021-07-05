package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

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
