package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadApps(t *testing.T) []App {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "rootfs", "usr", "share", "isotope", "first-run-apps.list"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	apps, err := ParseAppList(f)
	if err != nil {
		t.Fatal(err)
	}
	return apps
}

// TestParseAppListMatchesScript: the data file has the nine rows the
// terminal checklist (isotope-first-run-apps) had.
func TestParseAppListMatchesScript(t *testing.T) {
	want := []App{
		{"browser", "org.mozilla.firefox", "Firefox", true},
		{"browser", "org.chromium.Chromium", "Chromium", false},
		{"browser", "com.brave.Browser", "Brave", false},
		{"browser", "io.gitlab.librewolf-community", "LibreWolf", false},
		{"extra", "org.gnome.Loupe", "Image viewer (Loupe)", true},
		{"extra", "org.videolan.VLC", "Media player (VLC)", true},
		{"extra", "org.gnome.TextEditor", "Text editor (GNOME Text Editor)", true},
		{"extra", "org.gnome.Papers", "PDF viewer (Papers)", true},
		{"extra", "com.github.tchx84.Flatseal", "Flatpak permissions (Flatseal)", false},
	}
	got := loadApps(t)
	if len(got) != len(want) {
		t.Fatalf("%d rows, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestAppListIsTheScriptsList keeps the two lists equal while the script
// still ships, so the list is not defined twice with different rows.
func TestAppListIsTheScriptsList(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "rootfs", "usr", "bin", "isotope-first-run-apps"))
	if err != nil {
		t.Skip("the script is gone")
	}
	s := string(b)
	for _, a := range loadApps(t) {
		on := "off"
		if a.Preselected {
			on = "on"
		}
		row := a.Group + "|" + a.ID + "|" + a.Label + "|" + on
		if !strings.Contains(s, row) {
			t.Errorf("row %q is not in the script's APPS", row)
		}
	}
}

func TestParseAppListErrors(t *testing.T) {
	for _, in := range []string{"browser|a|b", "other|a|b|on", "extra|a|b|maybe", "extra||b|on"} {
		if _, err := ParseAppList(strings.NewReader(in)); err == nil {
			t.Errorf("%q: want an error", in)
		}
	}
	got, err := ParseAppList(strings.NewReader("# c\n\nextra|a.b|Ab|on\n"))
	if err != nil || len(got) != 1 {
		t.Fatalf("%v %v", got, err)
	}
}

func TestShort(t *testing.T) {
	for in, want := range map[string]string{
		"Firefox":                         "Firefox",
		"Image viewer (Loupe)":            "Loupe",
		"Text editor (GNOME Text Editor)": "GNOME Text Editor",
	} {
		if got := (App{Label: in}).Short(); got != want {
			t.Errorf("Short(%q) = %q, want %q", in, got, want)
		}
	}
}
