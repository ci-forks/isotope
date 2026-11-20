package main

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// TestWelcomeExecsTheApp: `isotope welcome ARGS` replaces itself with
// /usr/bin/isotope-welcome ARGS, so the app's exit code (2 for a bad page)
// is the command's.
func TestWelcomeExecsTheApp(t *testing.T) {
	for _, args := range [][]string{nil, {"apps"}, {"--login"}, {"--nonsense"}} {
		var gotPath string
		var gotArgv []string
		old := execWelcome
		execWelcome = func(path string, argv, env []string) error {
			gotPath, gotArgv = path, argv
			return nil
		}
		var out, errb bytes.Buffer
		code := run(append([]string{"welcome"}, args...), strings.NewReader(""), &out, &errb)
		execWelcome = old
		if code != exitOK {
			t.Errorf("%v: exit %d, stderr %q", args, code, errb.String())
		}
		if gotPath != "/usr/bin/isotope-welcome" {
			t.Errorf("%v: exec %q", args, gotPath)
		}
		want := append([]string{"isotope-welcome"}, args...)
		if !reflect.DeepEqual(gotArgv, want) {
			t.Errorf("%v: argv %q, want %q", args, gotArgv, want)
		}
	}
}

// TestWelcomeMissing: on an image without it (the agent, the server) the
// command says so and fails.
func TestWelcomeMissing(t *testing.T) {
	old := execWelcome
	defer func() { execWelcome = old }()
	execWelcome = func(string, []string, []string) error { return errors.New("no such file or directory") }
	var out, errb bytes.Buffer
	if code := run([]string{"welcome"}, strings.NewReader(""), &out, &errb); code != exitError {
		t.Errorf("exit %d, want %d", code, exitError)
	}
	if !strings.Contains(errb.String(), "isotope-welcome") {
		t.Errorf("stderr %q does not name isotope-welcome", errb.String())
	}
}

func TestWelcomeInUsage(t *testing.T) {
	if !strings.Contains(usage, "\n  welcome ") {
		t.Error("the usage text does not list welcome")
	}
}
