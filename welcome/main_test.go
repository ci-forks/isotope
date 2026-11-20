package main

import "testing"

func TestParseArgs(t *testing.T) {
	for _, a := range []string{"", "--login", "apps", "backup", "language", "network", "appearance", "accessibility", "shortcuts"} {
		args := []string{a}
		if a == "" {
			args = nil
		}
		if p, ok := ParseArgs(args); !ok || string(p) != a {
			t.Errorf("%q: %q %v", a, p, ok)
		}
	}
	for _, bad := range [][]string{{"--nonsense"}, {"apps", "x"}} {
		if _, ok := ParseArgs(bad); ok {
			t.Errorf("%v accepted", bad)
		}
	}
}
