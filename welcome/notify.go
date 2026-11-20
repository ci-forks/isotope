package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

// later says where to find the apps again.
const later = "Open it later from the Isotope menu (Super+Esc → Welcome)."

// CompletionNotice is the notification for an install that ended while
// the window was gone: its summary and body, from the apps' names.
func CompletionNotice(ok, failed []string) (summary, body string) {
	switch {
	case len(failed) == 0:
		return "Apps installed", "Installed: " + strings.Join(ok, ", ") + ". Find them with the app launcher."
	case len(ok) > 0:
		return "Some apps did not install", "Installed: " + strings.Join(ok, ", ") + ". Failed: " + strings.Join(failed, ", ") + ". " + later
	}
	return "Apps did not install", "Failed: " + strings.Join(failed, ", ") + ". " + later
}

// Notify sends a desktop notification with busctl, as the terminal
// checklist's notify did. The notification daemon may still be starting
// at login, so it tries up to tries times, wait apart.
func Notify(r Runner, summary, body string, tries int, wait time.Duration) error {
	var err error
	for i := 0; i < tries; i++ {
		if i > 0 {
			time.Sleep(wait)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err = r.Run(ctx, "busctl", "--user", "call", "org.freedesktop.Notifications", "/org/freedesktop/Notifications",
			"org.freedesktop.Notifications", "Notify", "susssasa{sv}i",
			"Isotope", "0", "", summary, body, "0", "0", "15000")
		cancel()
		if err == nil {
			return nil
		}
	}
	return err
}

// appsCommand is what a click on the reminder runs: this window, on the
// Apps page. (The same as `isotope welcome apps`.)
var appsCommand = []string{"/usr/bin/isotope-welcome", "apps"}

const notifications = "org.freedesktop.Notifications"

// Remind sends the reminder notification at login and waits for a click
// on it (its default action); a click calls open. It returns when the
// notification was clicked or closed, or after a while. It tries for
// half a minute while the notification daemon starts.
func Remind(summary, body string, open func()) error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	defer conn.Close()
	for _, m := range []string{"ActionInvoked", "NotificationClosed"} {
		if err := conn.AddMatchSignal(dbus.WithMatchObjectPath("/org/freedesktop/Notifications"),
			dbus.WithMatchInterface(notifications), dbus.WithMatchMember(m)); err != nil {
			return err
		}
	}
	sigs := make(chan *dbus.Signal, 16)
	conn.Signal(sigs)
	obj := conn.Object(notifications, "/org/freedesktop/Notifications")
	var id uint32
	for i := 0; ; i++ {
		call := obj.Call(notifications+".Notify", 0, "Isotope", uint32(0), "", summary, body,
			[]string{"default", "Open"}, map[string]dbus.Variant{}, int32(15000))
		if err = call.Store(&id); err == nil {
			break
		}
		if i >= 14 {
			return fmt.Errorf("notify: %w", err)
		}
		time.Sleep(2 * time.Second)
	}
	if waitAction(sigs, id, 2*time.Minute) {
		open()
	}
	return nil
}

// waitAction waits for the default action of notification id; false when
// the notification closes first, or on timeout.
func waitAction(c <-chan *dbus.Signal, id uint32, timeout time.Duration) bool {
	t := time.NewTimer(timeout)
	defer t.Stop()
	for {
		select {
		case s := <-c:
			if s == nil || len(s.Body) == 0 {
				continue
			}
			if got, _ := s.Body[0].(uint32); got != id {
				continue
			}
			switch s.Name {
			case notifications + ".ActionInvoked":
				if a, _ := s.Body[1].(string); a == "default" {
					return true
				}
			case notifications + ".NotificationClosed":
				return false
			}
		case <-t.C:
			return false
		}
	}
}

// remindApps is the --login step for a user who skipped Apps offline: once
// there is a network and a usable Flathub, send the reminder, whose click
// opens the Apps page. It then writes the "skipped" stamp: one reminder.
func remindApps(r Runner, stateDir string) {
	if !Online(r) || !FlathubReady(r) {
		return // still pending: the next login tries again
	}
	err := Remind("Install a browser and essential apps",
		"Isotope has no browser yet. Pick one and a few essentials from Flathub.",
		func() {
			if err := start(appsCommand...); err != nil {
				fmt.Fprintf(os.Stderr, "isotope-welcome: %v\n", err)
			}
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "isotope-welcome: %v\n", err)
		return
	}
	if err := WriteStamp(stateDir, StampSkipped); err != nil {
		fmt.Fprintf(os.Stderr, "isotope-welcome: %v\n", err)
	}
}
