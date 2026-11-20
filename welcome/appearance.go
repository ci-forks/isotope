package main

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Accent is one of COSMIC's accent colours, as the hex string
// cosmic-theme stores a colour as (#RRGGBBAA).
type Accent struct {
	Name string
	Hex  string
}

// Accents are the accent colours of COSMIC's own palette (the accent_*
// entries of cosmic-theme's dark.ron). cosmic-initial-setup's appearance
// page offers none of its own: it picks a whole theme, and the accent is the
// `accent` entry of the theme builder, which COSMIC Settings fills from
// these same colours.
func Accents() []Accent {
	return []Accent{
		{"Blue", "#63D0DFFF"},
		{"Indigo", "#A1C0EBFF"},
		{"Purple", "#E79CFEFF"},
		{"Pink", "#FF9CB1FF"},
		{"Red", "#FDA1A0FF"},
		{"Orange", "#FFAD00FF"},
		{"Yellow", "#F7E062FF"},
		{"Green", "#92CF9CFF"},
		{"Warm grey", "#CABAB4FF"},
	}
}

const gsettingsSchema = "org.gnome.desktop.interface"

func gsettings(r Runner, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.Run(ctx, "gsettings", args...)
}

// SetColorScheme makes apps dark or light: the portal's colour scheme
// (gsettings, which needs the user's session bus), GTK 3's setting file,
// and on COSMIC the theme mode COSMIC itself follows.
func SetColorScheme(r Runner, home string, d Desktop, dark bool) error {
	scheme := "prefer-light"
	if dark {
		scheme = "prefer-dark"
	}
	if _, err := gsettings(r, "set", gsettingsSchema, "color-scheme", scheme); err != nil {
		return fmt.Errorf("gsettings: %w", err)
	}
	ini := filepath.Join(home, ".config", "gtk-3.0", "settings.ini")
	if err := setIniKey(ini, "Settings", "gtk-application-prefer-dark-theme", fmt.Sprint(dark)); err != nil {
		return err
	}
	if d == Cosmic {
		return writeCosmic(home, "com.system76.CosmicTheme.Mode", 1, "is_dark", fmt.Sprint(dark))
	}
	return nil
}

// ReadColorScheme is what is set now. Dark is the answer when nothing says
// otherwise: it is Isotope's look.
func ReadColorScheme(r Runner, home string, d Desktop) (dark bool) {
	if d == Cosmic {
		b, err := os.ReadFile(cosmicKey(home, "com.system76.CosmicTheme.Mode", 1, "is_dark"))
		if err == nil {
			return strings.TrimSpace(string(b)) != "false"
		}
	}
	if r != nil {
		if out, err := gsettings(r, "get", gsettingsSchema, "color-scheme"); err == nil {
			return !strings.Contains(string(out), "prefer-light")
		}
	}
	return true
}

// SetCosmicAccent sets the accent colour of the dark and the light theme, as
// the `accent` entry of their builders.
func SetCosmicAccent(home string, accent Accent) error {
	for _, id := range cosmicBuilders {
		if err := writeCosmic(home, id, cosmicThemeVersion, "accent", fmt.Sprintf("Some(%q)", accent.Hex)); err != nil {
			return err
		}
	}
	return nil
}

// SetCosmicFrosted turns COSMIC's blurred transparency on or off for its
// four surfaces, the entries cosmic-initial-setup's appearance page sets.
func SetCosmicFrosted(home string, on bool) error {
	for _, id := range cosmicBuilders {
		for _, k := range []string{"frosted_applets", "frosted_panel", "frosted_system_interface", "frosted_windows"} {
			if err := writeCosmic(home, id, cosmicThemeVersion, k, fmt.Sprint(on)); err != nil {
				return err
			}
		}
	}
	return nil
}

// cosmicThemeVersion is the config version of cosmic-theme's ThemeBuilder at
// libcosmic 430f3d4 (what cosmic-initial-setup epoch-1.11.0 builds).
const cosmicThemeVersion = 2

var cosmicBuilders = []string{"com.system76.CosmicTheme.Dark.Builder", "com.system76.CosmicTheme.Light.Builder"}

// cosmicKey is the file cosmic-config keeps one key in:
// ~/.config/cosmic/<id>/v<version>/<key>, holding the key's RON.
func cosmicKey(home, id string, version int, key string) string {
	return filepath.Join(home, ".config", "cosmic", id, fmt.Sprintf("v%d", version), key)
}

func writeCosmic(home, id string, version int, key, ron string) error {
	p := cosmicKey(home, id, version, key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(ron), 0o644)
}

// setIniKey sets key=value in [section] of an INI file, keeping every other
// line, and creates the file or the section when it is missing.
func setIniKey(path, section, key, value string) error {
	b, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	var lines []string
	if s := strings.TrimRight(string(b), "\n"); s != "" {
		lines = strings.Split(s, "\n")
	}
	line := key + "=" + value
	in, done, secEnd := false, false, -1
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "[") {
			if in {
				secEnd = i
			}
			in = t == "["+section+"]"
			if in {
				secEnd = len(lines)
			}
			continue
		}
		if in {
			k, _, ok := strings.Cut(t, "=")
			if ok && strings.TrimSpace(k) == key {
				lines[i], done = line, true
			}
		}
	}
	if !done {
		switch {
		case secEnd >= 0:
			lines = append(lines[:secEnd], append([]string{line}, lines[secEnd:]...)...)
		default:
			lines = append(lines, "["+section+"]", line)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

// hexColor reads "#RRGGBBAA".
func hexColor(s string) color.NRGBA {
	var r, g, b, a uint8
	fmt.Sscanf(s, "#%02x%02x%02x%02x", &r, &g, &b, &a)
	return color.NRGBA{R: r, G: g, B: b, A: a}
}
