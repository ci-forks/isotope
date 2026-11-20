package main

import (
	"errors"
	"testing"

	"gioui.org/app"

	"github.com/kairos-community/isotope/guikit"
)

// recorder is a Hooks whose Finish and Logout calls are kept.
type recorder struct {
	stamps  []Stamp
	logouts int
	sheets  int
}

func (r *recorder) hooks() Hooks {
	return Hooks{
		Invalidate:        func() {},
		Logout:            func() error { r.logouts++; return nil },
		OpenShortcutSheet: func() error { r.sheets++; return nil },
		Finish:            func(s Stamp) { r.stamps = append(r.stamps, s) },
	}
}

func to(u *UI, p PageID) {
	for u.flow.Current() != p {
		u.flow.Next()
	}
}

func TestCloseBeforeFinishIsASkip(t *testing.T) {
	var r recorder
	u := newUI(NewFlow(Sway), Sway, guikit.NewShaper(), r.hooks())
	to(u, PNetwork)
	u.Closed()
	if len(r.stamps) != 1 || r.stamps[0] != StampSkipped {
		t.Fatalf("stamps %v, want [skipped]", r.stamps)
	}
}

func TestFinishOnReady(t *testing.T) {
	var r recorder
	u := newUI(NewFlow(Cosmic), Cosmic, guikit.NewShaper(), r.hooks())
	to(u, PReady)
	u.next()
	// The window then closes: that is not a second stamp.
	u.Closed()
	if len(r.stamps) != 1 || r.stamps[0] != StampDone {
		t.Fatalf("stamps %v, want [completed]", r.stamps)
	}
}

func TestFinishAppsPending(t *testing.T) {
	var r recorder
	u := newUI(NewFlow(Sway), Sway, guikit.NewShaper(), r.hooks())
	u.appsPending = true
	to(u, PReady)
	u.next()
	if len(r.stamps) != 1 || r.stamps[0] != StampAppsPending {
		t.Fatalf("stamps %v, want [apps-pending]", r.stamps)
	}
}

func TestSkipSetupFromWelcome(t *testing.T) {
	var r recorder
	u := newUI(NewFlow(Sway), Sway, guikit.NewShaper(), r.hooks())
	u.skipSetup()
	if len(r.stamps) != 1 || r.stamps[0] != StampSkipped {
		t.Fatalf("stamps %v, want [skipped]", r.stamps)
	}
}

func TestLogOutWritesTheStampFirst(t *testing.T) {
	var r recorder
	h := r.hooks()
	h.Logout = func() error {
		if len(r.stamps) != 1 || r.stamps[0] != StampDone {
			t.Errorf("logout before the stamp: %v", r.stamps)
		}
		r.logouts++
		return nil
	}
	u := newUI(NewFlow(Sway), Sway, guikit.NewShaper(), h)
	u.flow.Outcome.LanguageChanged = "Deutsch (Deutschland)"
	to(u, PReady)
	u.page().(*readyPage).logOut()
	if r.logouts != 1 {
		t.Fatalf("logouts %d", r.logouts)
	}
}

func TestLogOutFailureIsShown(t *testing.T) {
	var r recorder
	h := r.hooks()
	h.Logout = func() error { return errors.New("swaymsg: not found") }
	u := newUI(NewFlow(Sway), Sway, guikit.NewShaper(), h)
	to(u, PReady)
	rp := u.page().(*readyPage)
	rp.logOut()
	if rp.logoutErr == "" {
		t.Fatal("no error shown")
	}
}

func TestBackAndContinue(t *testing.T) {
	var r recorder
	u := newUI(NewFlow(Sway), Sway, guikit.NewShaper(), r.hooks())
	u.next()
	if u.flow.Current() != PLanguage {
		t.Fatalf("Continue on Welcome went to %s", u.flow.Current())
	}
	u.prev()
	u.prev() // Esc on the first page stays there.
	if u.flow.Current() != PWelcome {
		t.Fatalf("Go back went to %s", u.flow.Current())
	}
	to(u, PShortcuts)
	u.next()
	if u.flow.Current() != PReady || len(r.stamps) != 0 {
		t.Fatalf("Continue on Shortcuts: %s, stamps %v", u.flow.Current(), r.stamps)
	}
}

func TestSinglePageFlowFinishes(t *testing.T) {
	var r recorder
	u := newUI(Single(PShortcuts), Sway, guikit.NewShaper(), r.hooks())
	u.next()
	if len(r.stamps) != 1 || r.stamps[0] != StampDone {
		t.Fatalf("stamps %v", r.stamps)
	}
}

func TestShowAllShortcuts(t *testing.T) {
	var r recorder
	u := newUI(NewFlow(Sway), Sway, guikit.NewShaper(), r.hooks())
	to(u, PShortcuts)
	u.page().(*shortcutsPage).showAll()
	if r.sheets != 1 {
		t.Fatalf("sheet opened %d times", r.sheets)
	}
}

// The window's Wayland app_id and X11 class, which the window rules of
// sway, i3 and niri float.
func TestAppID(t *testing.T) {
	if app.ID != "isotope-welcome" {
		t.Fatalf("app.ID = %q", app.ID)
	}
}
