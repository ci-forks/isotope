package main

import (
	"fmt"
	"os"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kairos-community/isotope/install-ui/core"
)

// runInstall runs the installer as a subprocess, rendering the branded progress
// UI from its output. kairos-agent has no progress API, so its stdout is the
// only signal available (see core/steps.go). post, when set, runs after a
// successful install and its failure is an install failure.
func runInstall(manualConfig string, post core.PostInstall) error {
	in, err := core.NewInstall(manualConfig, post)
	if err != nil {
		return err
	}
	defer in.Close()

	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	go in.Run(func(e core.Event) { p.Send(eventMsg(e)) })

	final, err := p.Run()
	if err != nil {
		return err
	}
	if m, ok := final.(model); ok && m.action == actionShell {
		// Replace this process so the rescue shell owns tty1 outright.
		return syscall.Exec("/bin/sh", []string{"sh"}, os.Environ())
	}
	if m, ok := final.(model); ok && m.failed {
		return fmt.Errorf("install failed")
	}
	return nil
}

func main() {
	progressOnly := false
	for _, a := range os.Args[1:] {
		switch a {
		case "--progress-only":
			progressOnly = true
		default:
			fmt.Fprintf(os.Stderr, "isotope-install-ui: unknown argument: %s\n", a)
			os.Exit(2)
		}
	}

	// --progress-only (the agent appliance's installer) never runs the
	// wizard, so it never offers encryption.
	// The desktop ISOs ask in the graphical installer, which also runs the
	// install. The unattended paths keep the text progress screen.
	if !progressOnly && guiWanted() {
		if auto, err := unattended(); err == nil && !auto {
			fallback, note, err := runGUI()
			if err != nil {
				fmt.Fprintf(os.Stderr, "isotope-install-ui: %v\n", err)
				os.Exit(1)
			}
			if !fallback {
				return
			}
			fmt.Fprintln(os.Stderr, note)
		}
	}

	var choice *core.Choice
	if !progressOnly {
		c, err := runWizard()
		if err != nil {
			fmt.Fprintf(os.Stderr, "isotope-install-ui: %v\n", err)
			os.Exit(1)
		}
		choice = c
	}

	pre, cleanup, post := core.EncryptionSteps(choice)
	manualConfig := ""
	if pre != nil {
		manualConfig = core.ConfigOut()
		if err := pre(); err != nil {
			fmt.Fprintf(os.Stderr, "isotope-install-ui: %v\n", err)
			os.Exit(1)
		}
	}
	err := runInstall(manualConfig, post)
	if cleanup != nil {
		// post already deleted it after a successful install; this covers
		// a failed one.
		if cerr := cleanup(); cerr != nil && err == nil {
			err = cerr
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "isotope-install-ui: %v\n", err)
		os.Exit(1)
	}
	if post != nil {
		// The encrypted install told kairos-agent not to reboot.
		if err := core.RebootCommand().Run(); err != nil {
			fmt.Fprintf(os.Stderr, "isotope-install-ui: reboot: %v\n", err)
			os.Exit(1)
		}
	}
}
