package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kairos-community/isotope/install-ui/core"
)

// stubGUI points cage and the GUI at scripts in a temporary directory and
// the markers at another; it returns the cage script's path.
func stubGUI(t *testing.T, cage, cmdline string) string {
	t.Helper()
	dir := t.TempDir()
	run := filepath.Join(dir, "run")
	t.Setenv("ISOTOPE_INSTALL_RUN_DIR", run)
	oldCage, oldGUI, oldCmd := cagePath, guiPath, cmdlinePath
	t.Cleanup(func() { cagePath, guiPath, cmdlinePath = oldCage, oldGUI, oldCmd })
	cagePath = filepath.Join(dir, "cage")
	guiPath = filepath.Join(dir, "isotope-install-gui")
	cmdlinePath = filepath.Join(dir, "cmdline")
	for p, body := range map[string]string{cagePath: cage, guiPath: "exit 0"} {
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(cmdlinePath, []byte(cmdline), 0o644); err != nil {
		t.Fatal(err)
	}
	return cagePath
}

func TestGUIWanted(t *testing.T) {
	stubGUI(t, "exit 0", "quiet splash install-mode-interactive\n")
	if !guiWanted() {
		t.Fatal("not wanted with cage and the GUI present")
	}
	if err := os.WriteFile(cmdlinePath, []byte("quiet isotope.installer=text splash\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if guiWanted() {
		t.Fatal("wanted with isotope.installer=text")
	}
	stubGUI(t, "exit 0", "")
	os.Remove(cagePath)
	if guiWanted() {
		t.Fatal("wanted without cage (the server)")
	}
}

func TestRunGUIFallsBackBeforeTheFirstFrame(t *testing.T) {
	stubGUI(t, "exit 1", "")
	fallback, note, err := runGUI()
	if err != nil || !fallback || !strings.Contains(note, "could not start") {
		t.Fatalf("fallback %v, note %q, err %v", fallback, note, err)
	}
}

func TestRunGUIDone(t *testing.T) {
	stubGUI(t, `touch "$ISOTOPE_INSTALL_RUN_DIR/gui-started"; exit 0`, "")
	fallback, _, err := runGUI()
	if err != nil || fallback {
		t.Fatalf("fallback %v, err %v", fallback, err)
	}
}

// The GUI asks for a relaunch with a new layout: cage starts again with it
// as its keymap.
func TestRunGUIRelaunchesWithTheLayout(t *testing.T) {
	stubGUI(t, `
r="$ISOTOPE_INSTALL_RUN_DIR"
touch "$r/gui-started"
if [ ! -f "$r/second" ]; then
	touch "$r/second"
	echo '{"next":"account","layout":"it"}' > "$r/state.json"
	exit 75
fi
echo "$XKB_DEFAULT_LAYOUT $ISOTOPE_INSTALL_GUI_RELAUNCH $LIBSEAT_BACKEND" > "$r/env"
exit 0`, "")
	fallback, _, err := runGUI()
	if err != nil || fallback {
		t.Fatalf("fallback %v, err %v", fallback, err)
	}
	b, err := os.ReadFile(filepath.Join(core.RunDir(), "env"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(b)); got != "it 1 builtin" {
		t.Fatalf("second cage's environment: %q", got)
	}
}

func TestRunGUICrashAfterStartShowsText(t *testing.T) {
	stubGUI(t, `touch "$ISOTOPE_INSTALL_RUN_DIR/gui-started"; exit 2`, "")
	fallback, note, err := runGUI()
	if err != nil || !fallback || !strings.Contains(note, "stopped") {
		t.Fatalf("fallback %v, note %q, err %v", fallback, note, err)
	}
}

func TestRunGUICrashDuringTheInstallStartsNoSecond(t *testing.T) {
	stubGUI(t, `r="$ISOTOPE_INSTALL_RUN_DIR"; touch "$r/gui-started" "$r/install-running"; exit 2`, "")
	fallback, _, err := runGUI()
	if err != errInstallRunning || fallback {
		t.Fatalf("fallback %v, err %v", fallback, err)
	}
}

func TestGUIStateHasNoSecrets(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.json")
	if err := core.WriteGUIState(p, core.GUIState{Next: "account", Layout: "de", Variant: "nodeadkeys", Username: "alex"}); err != nil {
		t.Fatal(err)
	}
	s, err := core.ReadGUIState(p)
	if err != nil || s.Layout != "de" || s.Variant != "nodeadkeys" || s.Username != "alex" {
		t.Fatalf("%+v, %v", s, err)
	}
	b, _ := os.ReadFile(p)
	for _, w := range []string{"pass", "key_value", "recovery_key_value"} {
		if strings.Contains(string(b), w) {
			t.Fatalf("state has %q: %s", w, b)
		}
	}
	if err := os.WriteFile(p, []byte(`{"layout":"de;rm -rf /"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if s, _ := core.ReadGUIState(p); s.Layout != "" {
		t.Fatalf("an invalid layout survived: %q", s.Layout)
	}
}

func TestRunGUIRescueShell(t *testing.T) {
	stubGUI(t, `touch "$ISOTOPE_INSTALL_RUN_DIR/gui-started"; exit 76`, "")
	fallback, _, err := runGUI()
	if err != errRescueShell || fallback {
		t.Fatalf("fallback %v, err %v", fallback, err)
	}
}
