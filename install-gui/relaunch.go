package main

import (
	"github.com/kairos-community/isotope/install-ui/core"
)

// Keyboard layouts and cage.
//
// The design note wanted the window to keep its own xkb_state and translate
// raw keycodes, so a layout applies at once under cage, whose keymap is fixed
// at start. Gio does not allow that: its Wayland backend keeps the xkb state
// in an internal package (gioui.org/app/internal/xkb), loads only the keymap
// the compositor sends, and hands the program translated key.Events and
// EditEvents, never keycodes. So this is the note's fallback: a layout other
// than the one cage was started with saves the answers (never a secret) to
// core.GUIStatePath and exits with core.GUIRelaunchExit;
// isotope-install-ui starts cage again with XKB_DEFAULT_LAYOUT and
// XKB_DEFAULT_VARIANT set, and the window opens where it was.

// saveState is the answers so far, to open next after the relaunch.
func (u *UI) saveState(next gpage) core.GUIState {
	s := core.GUIState{
		Next:        "keyboard",
		Search:      u.kbSearch.ed.Text(),
		Hostname:    u.host.ed.Text(),
		Username:    u.user.ed.Text(),
		Github:      u.gh.ed.Text(),
		Encrypt:     u.encOn,
		RecoveryKey: u.recKey,
	}
	if next == pAccount {
		s.Next = "account"
	}
	if u.kbSel >= 0 {
		k := u.w.Keyboards[u.kbSel]
		s.Layout, s.Variant = k.Layout, k.Variant
	}
	if u.tzSel >= 0 {
		s.Timezone = u.w.Zones[u.tzSel]
	}
	if u.diskSel < len(u.w.Disks) {
		s.Disk = u.w.Disks[u.diskSel].Path
	}
	return s
}

// restore applies a saved state to a fresh wizard: the keyboard and zone go
// through core, the other answers back into their fields for Continue to
// check again. The password fields stay empty.
func (u *UI) restore(s core.GUIState) {
	w := u.w
	for i, k := range w.Keyboards {
		if k.Layout == s.Layout && k.Variant == s.Variant {
			u.kbSel = i
		}
	}
	for i, z := range w.Zones {
		if z == s.Timezone {
			u.pickZone(i)
		}
	}
	u.kbSearch.ed.SetText(s.Search)
	u.kbSearch.ed.SetCaret(u.kbSearch.ed.Len(), u.kbSearch.ed.Len())
	if s.Hostname != "" {
		u.host.ed.SetText(s.Hostname)
	}
	u.user.ed.SetText(s.Username)
	u.gh.ed.SetText(s.Github)
	for i, d := range w.Disks {
		if d.Path == s.Disk {
			u.diskSel = i
		}
	}
	u.encOn, u.recKey = s.Encrypt, s.RecoveryKey
	u.scrollToSelection()
	if s.Next == "account" && u.kbSel >= 0 && u.tzSel >= 0 {
		w.SetKeyboard(u.kbSel)
		w.SetTimezone(u.tzSel)
		u.goTo(pAccount)
		return
	}
	u.kbApplied = true
	u.goTo(pKeyboard)
}
