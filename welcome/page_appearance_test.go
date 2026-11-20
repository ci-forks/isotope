package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func appearanceUI(t *testing.T, d Desktop, r *fakeRunner) (*UI, *recorder, *appearancePage) {
	t.Helper()
	u, rec := newRecorded(NewFlow(d), d)
	u.hooks.Runner, u.hooks.Home = r, t.TempDir()
	to(u, PAppearance)
	p := u.page().(*appearancePage)
	p.start()
	return u, rec, p
}

func TestAppearanceShowsWhatIsSet(t *testing.T) {
	get := "gsettings get org.gnome.desktop.interface color-scheme"
	_, _, p := appearanceUI(t, Sway, &fakeRunner{out: map[string]string{get: "'prefer-light'\n"}})
	if p.dark {
		t.Fatal("the page must show the light scheme that is set")
	}
	_, _, p = appearanceUI(t, Sway, &fakeRunner{fail: []string{"gsettings"}})
	if !p.dark {
		t.Fatal("unknown means dark")
	}
}

func TestAppearanceContinueApplies(t *testing.T) {
	r := &fakeRunner{}
	u, _, p := appearanceUI(t, Cosmic, r)
	p.dark, p.pick, p.frosted, p.frostedSet = false, 1, true, true
	p.Continue()
	if u.flow.Current() != PLayout {
		t.Fatalf("page %s", u.flow.Current())
	}
	h := u.hooks.Home
	for path, want := range map[string]string{
		".config/cosmic/com.system76.CosmicTheme.Mode/v1/is_dark":               "false",
		".config/cosmic/com.system76.CosmicTheme.Light.Builder/v2/accent":       `Some("#A1C0EBFF")`,
		".config/cosmic/com.system76.CosmicTheme.Dark.Builder/v2/frosted_panel": "true",
	} {
		if got := readFile(t, filepath.Join(h, path)); got != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
}

func TestAppearanceSwayLeavesCosmicAlone(t *testing.T) {
	u, _, p := appearanceUI(t, Sway, &fakeRunner{})
	p.pick, p.frostedSet = 2, true // not reachable on sway, but must not write
	p.Continue()
	if _, err := os.Stat(filepath.Join(u.hooks.Home, ".config", "cosmic")); err == nil {
		t.Fatal("sway wrote COSMIC files")
	}
}

func TestAppearanceFailureStaysWithAnError(t *testing.T) {
	u, _, p := appearanceUI(t, Sway, &fakeRunner{fail: []string{"gsettings set"}})
	p.Continue()
	if u.flow.Current() != PAppearance || !strings.Contains(p.err, "Could not change the appearance") {
		t.Fatalf("page %s, err %q", u.flow.Current(), p.err)
	}
}
