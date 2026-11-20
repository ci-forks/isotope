package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestColorSchemeSway(t *testing.T) {
	home := t.TempDir()
	ini := filepath.Join(home, ".config", "gtk-3.0", "settings.ini")
	os.MkdirAll(filepath.Dir(ini), 0o755)
	os.WriteFile(ini, []byte("[Settings]\ngtk-theme-name=Adwaita\ngtk-application-prefer-dark-theme=true\n\n[Other]\nx=1\n"), 0o644)
	r := &fakeRunner{}
	if err := SetColorScheme(r, home, Sway, false); err != nil {
		t.Fatal(err)
	}
	want := []string{"gsettings set org.gnome.desktop.interface color-scheme prefer-light"}
	if !reflect.DeepEqual(r.called(), want) {
		t.Fatalf("calls = %v", r.called())
	}
	got := readFile(t, ini)
	if got != "[Settings]\ngtk-theme-name=Adwaita\ngtk-application-prefer-dark-theme=false\n\n[Other]\nx=1\n" {
		t.Fatalf("settings.ini:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "cosmic")); err == nil {
		t.Fatal("sway wrote COSMIC files")
	}
}

func TestColorSchemeNewIni(t *testing.T) {
	home := t.TempDir()
	if err := SetColorScheme(&fakeRunner{}, home, I3, true); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(home, ".config", "gtk-3.0", "settings.ini")); got != "[Settings]\ngtk-application-prefer-dark-theme=true\n" {
		t.Fatalf("settings.ini:\n%s", got)
	}
}

func TestColorSchemeCosmic(t *testing.T) {
	home := t.TempDir()
	r := &fakeRunner{}
	if err := SetColorScheme(r, home, Cosmic, false); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(home, ".config/cosmic/com.system76.CosmicTheme.Mode/v1/is_dark")); got != "false" {
		t.Fatalf("is_dark = %q", got)
	}
	if ReadColorScheme(r, home, Cosmic) {
		t.Fatal("ReadColorScheme says dark after light")
	}
}

func TestColorSchemeGsettingsFails(t *testing.T) {
	r := &fakeRunner{fail: []string{"gsettings"}}
	err := SetColorScheme(r, t.TempDir(), Sway, true)
	if err == nil || !strings.Contains(err.Error(), "gsettings") {
		t.Fatalf("err = %v", err)
	}
}

func TestReadColorScheme(t *testing.T) {
	get := "gsettings get org.gnome.desktop.interface color-scheme"
	for out, want := range map[string]bool{"'prefer-light'\n": false, "'prefer-dark'\n": true, "'default'\n": true} {
		if got := ReadColorScheme(&fakeRunner{out: map[string]string{get: out}}, t.TempDir(), Sway); got != want {
			t.Errorf("%q: dark = %v", out, got)
		}
	}
	if !ReadColorScheme(&fakeRunner{fail: []string{"gsettings"}}, t.TempDir(), Sway) {
		t.Error("an unreadable scheme must default to dark")
	}
}

func TestCosmicAccent(t *testing.T) {
	home := t.TempDir()
	if err := SetCosmicAccent(home, Accents()[0]); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"Dark", "Light"} {
		p := filepath.Join(home, ".config/cosmic/com.system76.CosmicTheme."+id+".Builder/v2/accent")
		if got := readFile(t, p); got != `Some("#63D0DFFF")` {
			t.Errorf("%s accent = %q", id, got)
		}
	}
}

func TestCosmicFrosted(t *testing.T) {
	home := t.TempDir()
	if err := SetCosmicFrosted(home, true); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"frosted_applets", "frosted_panel", "frosted_system_interface", "frosted_windows"} {
		p := filepath.Join(home, ".config/cosmic/com.system76.CosmicTheme.Dark.Builder/v2", k)
		if got := readFile(t, p); got != "true" {
			t.Errorf("%s = %q", k, got)
		}
	}
}
