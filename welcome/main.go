// Command isotope-welcome is the graphical first-login flow.
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/kairos-community/isotope/guikit"
)

// appID is the window's Wayland app_id and X11 class: the window rules of
// sway, i3 and niri float it by this name.
const appID = "isotope-welcome"

// Gio reads app.ID when it makes the window, and defaults it to the name the
// binary was started as; `isotope welcome` may start it under another one.
func init() { app.ID = appID }

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
	d, err := ReadDesktop(SessionFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "isotope-welcome: %v\n", err)
		os.Exit(1)
	}
	home, _ := os.UserHomeDir()
	stateDir := StateDir(home, os.Getenv)
	// Only the whole flow writes the stamp: a page opened from the menu
	// says nothing about the first login.
	stamps := page == PageAll || page == PageLogin

	win := new(app.Window)
	win.Option(app.Title("Welcome to Isotope"), app.Size(unit.Dp(1100), unit.Dp(720)), app.Decorated(true))
	u := newUI(flowFor(page, d), d, guikit.NewShaper(), Hooks{
		Invalidate:        win.Invalidate,
		Logout:            func() error { return start(logoutCommand(d)...) },
		OpenShortcutSheet: func() error { return start("isotope-menu", "Help") },
		Finish: func(s Stamp) {
			if stamps {
				if err := WriteStamp(stateDir, s); err != nil {
					fmt.Fprintf(os.Stderr, "isotope-welcome: %v\n", err)
				}
			}
			win.Perform(system.ActionClose)
		},
	})
	u.version = versionLine("/etc/isotope/release")
	if f, err := os.Open(ShortcutsFile); err == nil {
		if s, err := ParseShortcuts(f); err == nil {
			u.shortcuts = s
		} else {
			fmt.Fprintf(os.Stderr, "isotope-welcome: %v\n", err)
		}
		f.Close()
	}

	go func() {
		var ops op.Ops
		for {
			switch e := win.Event().(type) {
			case app.DestroyEvent:
				// The window manager's close button lands here too: before
				// Finish, it is a skip.
				u.Closed()
				if e.Err != nil {
					fmt.Fprintf(os.Stderr, "isotope-welcome: %v\n", e.Err)
					os.Exit(1)
				}
				os.Exit(0)
			case app.FrameEvent:
				gtx := app.NewContext(&ops, e)
				u.Layout(gtx)
				e.Frame(gtx.Ops)
			}
		}
	}()
	app.Main()
}

// flowFor is the flow a command line opens: the whole flow, or one page.
func flowFor(p PageArg, d Desktop) *Flow {
	if p == PageAll || p == PageLogin {
		return NewFlow(d)
	}
	return Single(PageID(p))
}

// logoutCommand ends the session of desktop d, as the power menus do.
func logoutCommand(d Desktop) []string {
	switch d {
	case Sway:
		return []string{"swaymsg", "exit"}
	case I3:
		return []string{"i3-msg", "exit"}
	case Niri:
		return []string{"niri", "msg", "action", "quit", "--skip-confirmation"}
	}
	return []string{"cosmic-osd", "log-out"}
}

// start runs a command in the background and does not wait for it.
func start(argv ...string) error {
	cmd := exec.Command(argv[0], argv[1:]...)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// versionLine is the edition and version, "Isotope v0.0.25 · sway", from
// the release file the image build writes (as install-gui's).
func versionLine(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return "Isotope"
	}
	defer f.Close()
	var version, flavor string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, _ := strings.Cut(sc.Text(), "=")
		switch k {
		case "VERSION":
			version = v
		case "FLAVOR":
			flavor = v
		}
	}
	s := "Isotope"
	if version != "" {
		s += " " + version
	}
	if flavor != "" {
		s += " · " + flavor
	}
	return s
}
