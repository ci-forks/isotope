// Command isotope-welcome is the graphical first-login flow.
package main

import (
	"fmt"
	"os"
)

// PageArg names where the flow starts.
type PageArg string

const (
	PageAll           PageArg = ""
	PageLogin         PageArg = "--login"
	PageApps          PageArg = "apps"
	PageBackup        PageArg = "backup"
	PageLanguage      PageArg = "language"
	PageNetwork       PageArg = "network"
	PageAppearance    PageArg = "appearance"
	PageAccessibility PageArg = "accessibility"
	PageShortcuts     PageArg = "shortcuts"
)

// ParseArgs maps the command line to a PageArg; ok is false for anything else.
func ParseArgs(args []string) (PageArg, bool) {
	if len(args) == 0 {
		return PageAll, true
	}
	if len(args) > 1 {
		return "", false
	}
	switch p := PageArg(args[0]); p {
	case PageLogin, PageApps, PageBackup, PageLanguage, PageNetwork,
		PageAppearance, PageAccessibility, PageShortcuts:
		return p, true
	}
	return "", false
}

func main() {
	page, ok := ParseArgs(os.Args[1:])
	if !ok {
		fmt.Fprintln(os.Stderr, "usage: isotope-welcome [--login | apps | language | network | appearance | accessibility | shortcuts]")
		os.Exit(2)
	}
	if page == PageBackup {
		fmt.Fprintln(os.Stderr, "isotope-welcome: backup is not available yet")
		os.Exit(2)
	}
	if page == PageLogin {
		home, _ := os.UserHomeDir()
		if !ShouldOpenAtLogin(StateDir(home, os.Getenv)) {
			os.Exit(0)
		}
	}
	// The window arrives with the pages.
}
