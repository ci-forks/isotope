# Isotope Welcome Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the terminal app checklist, the welcome notification and, on COSMIC, `cosmic-initial-setup` with one graphical first-login flow, `isotope-welcome`, in the installer's look, on sway, i3, COSMIC and niri.

**Architecture:** A new Go module `welcome/` (Gio, cgo) built on `guikit/` (PR #121). Pure logic (login gate, flow, appliers that write settings) sits behind small interfaces (`Runner` for commands, `FS` paths rooted at a `$HOME`, D-Bus clients) and is unit-tested against a temporary home. Pages are drawn with guikit and covered by headless screenshot tests, as `install-gui` is. The one root action, setting the system locale, is a new method on `isotope-updated` (`update/`). Two builds: `-tags nox11,novulkan` for sway, COSMIC and niri; `-tags novulkan` (X11) for i3.

**Tech Stack:** Go 1.25, Gio (the version `install-gui/go.mod` pins), godbus/dbus v5 (as `update/`), polkit, NetworkManager D-Bus, flatpak CLI, gsettings.

**Spec:** `docs/superpowers/specs/2026-11-20-isotope-welcome-design.md` (this plan covers "Order of work" steps 2 and 3; Backup and Restore have their own plans). Read the spec's "The flow", "Starting, and what it replaces", "Build" and "Tests" sections before any task.

## Global Constraints

- Base: `ci-forks/refactor/shared-gui-kit` (PR #121) until it merges; then rebase on `origin/main`.
- Same flow on all four desktop variants, landing in one release: no PR in this plan may be released alone with `cosmic-initial-setup` removed but the COSMIC pages missing, or the reverse. Tasks merge as one PR (or back-to-back PRs with no release between).
- Page order: Welcome → Language → Network → Appearance → [Layout → Workflow → Launcher, COSMIC only] → Accessibility → Apps → Shortcuts → Ready. In this plan the Welcome page has only "Set up as new"; Restore is the Restore plan's.
- Window: decorated, about 1100×720 dp, not fullscreen; app ID and X11 class `isotope-welcome`; floats on sway, i3 and niri by window rule. Esc = Go back, Ctrl+Plus/Minus/0 = scale (`guikit.WindowKeys`).
- Every widget comes from `guikit`; no local copies of guikit widgets or colors.
- Stamp: `$XDG_STATE_HOME/isotope/welcome` (default `~/.local/state/isotope/welcome`). Old stamps that also mean "done": `isotope/first-run-apps`, `isotope/welcomed`, `hadron/first-run-apps`, `hadron/welcomed`.
- Apps list file: `/usr/share/isotope/first-run-apps.list`, format `group|Flatpak ID|label|preselected`, same rows as today's `APPS` in `rootfs/usr/bin/isotope-first-run-apps`.
- Flatpak installs: `flatpak --user install --noninteractive flathub <ids>` from the per-user `flathub` remote.
- Locale: daemon method `SetLocale(lang string)` on `io.kairos.IsotopeUpdate1`, polkit action `io.kairos.isotope-update.settings`; writes `LANG=<lang>` to `/etc/locale.conf` and `/oem/91_isotope_locale.yaml`.
- Light mode on sway, i3, niri changes apps only (gsettings `color-scheme`, `~/.config/gtk-3.0/settings.ini`); the chrome stays dark. The light card's text: "Apps use a light theme; the bar and menus stay dark".
- COSMIC panel restart after a layout change: `pkill -x cosmic-panel` (Isotope has no `killall`).
- No terminal, sudo prompt or root shell anywhere in the flow.
- Not on the agent image (`Dockerfile.agent` removes it) and not on the server.
- Commits: `git commit -s`, conventional subjects (`feat(welcome): ...`), body ends with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

## Review Focus

1. The user closes the window with the window manager's close button mid-flow (Gio cannot veto a close): this counts as a skip. The stamp is written as `skipped`, and an Apps install already started keeps running and notifies when done (Task 1 gate test, Task 4 detach test).
2. The machine is offline at first login: Network shows its offline state, Apps shows "Try again" and "Skip"; a skip for that reason makes the next login send the reminder notification that opens `isotope welcome apps` (Task 4 test).
3. A user who upgraded from a release with the old checklist: any old stamp keeps the flow from opening at login (Task 1 test, one case per old stamp).
4. A non-admin user: the daemon refuses `SetLocale`; the Language page shows it read-only with "Ask an administrator to change the language" (Task 6 tests).
5. A small screen (1024×768) and a 2× HiDPI screen: every page fits without clipping its footer (screenshot tests at 1024×768, and one page at scale 2).

---

### Task 1: Module, session, login gate and entry point

**Files:**
- Create: `welcome/go.mod` (module `github.com/kairos-community/isotope/welcome`, `replace` for `../guikit`, as `install-gui/go.mod` does), `welcome/main.go`, `welcome/session.go`, `welcome/gate.go`, `welcome/gate_test.go`, `welcome/session_test.go`
- Modify: `.github/workflows/ci.yml` (a welcome test step beside the guikit/install-gui ones)

**Interfaces:**
- Produces:
  - `type Desktop string` with consts `Sway`, `I3`, `Cosmic`, `Niri`; `ReadDesktop(sessionFile string) (Desktop, error)` reading `DESKTOP_NAME` from `/etc/isotope/session` (`COSMIC` → `Cosmic`, `sway`, `i3`, `niri`).
  - `type Stamp string` consts `StampDone = "completed"`, `StampSkipped = "skipped"`, `StampAppsPending = "apps-pending"`.
  - `StateDir(home string, env func(string) string) string` (honors `XDG_STATE_HOME`).
  - `ShouldOpenAtLogin(stateDir string) bool`; `WriteStamp(stateDir string, s Stamp) error`; `ReadStamp(stateDir string) (Stamp, bool)`.
  - `PageArg`: `main` accepts no argument (the whole flow), `--login`, or one page name: `apps`, `backup` (later plan; until then exit 2 with "not available yet"), `language`, `network`, `appearance`, `accessibility`, `shortcuts`. Any other argument exits 2 (the Dockerfile's `--nonsense` check).

- [ ] **Step 1: Write the failing tests**
  - `TestShouldOpenAtLogin`: a fresh state dir → true; with `welcome` present → false; for each of `isotope/first-run-apps`, `isotope/welcomed`, `hadron/first-run-apps`, `hadron/welcomed` (relative to the parent of the state dir, as the scripts compute `${state%/*}/hadron/...`) → false.
  - `TestWriteStampRoundTrip`: `WriteStamp(dir, StampSkipped)` then `ReadStamp` → `StampSkipped, true`; the file is mode 0644 and its parent is created.
  - `TestStateDirHonorsXDG`: `XDG_STATE_HOME=/x` → `/x/isotope`; unset → `<home>/.local/state/isotope`.
  - `TestReadDesktop`: files with `DESKTOP_NAME=COSMIC`, `sway`, `i3`, `niri` map to the consts; a missing key → error.

- [ ] **Step 2: Run them, see them fail**

Run: `cd welcome && go test ./... -run 'Login|Stamp|StateDir|Desktop'`
Expected: FAIL (undefined functions).

- [ ] **Step 3: Implement `session.go` and `gate.go`, and `main.go`'s argument parsing** (no window yet: `main` parses, and for `--login` exits 0 when `ShouldOpenAtLogin` is false).

- [ ] **Step 4: Run the tests and `go vet -tags nox11,novulkan ./...`**, see them pass.

- [ ] **Step 5: Add the CI step and commit** (`feat(welcome): add the module, its login gate and stamps`).

### Task 2: The flow model

**Files:**
- Create: `welcome/flow.go`, `welcome/flow_test.go`

**Interfaces:**
- Consumes: `Desktop` (Task 1).
- Produces:
  - `type PageID string` consts `PWelcome, PLanguage, PNetwork, PAppearance, PLayout, PWorkflow, PLauncher, PAccessibility, PApps, PShortcuts, PReady`.
  - `Pages(d Desktop) []PageID` — the order in Global Constraints; the three COSMIC pages only for `Cosmic`.
  - `PageTitle(p PageID) string` — sidebar names: "Welcome", "Language", "Network", "Appearance", "Layout", "Workflow", "Launcher", "Accessibility", "Apps", "Shortcuts", "Ready".
  - `type Flow struct` with `Current() PageID`, `Next()`, `Back()`, `Skip()` (marks the page skipped, then Next), `Index() int`, `Single(p PageID) Flow` (a flow of one page, for `isotope welcome apps`).
  - `type Outcome struct { Applied, Skipped []PageID; LanguageChanged string; AppsInstalled, AppsFailed []string }` — what Ready shows.

- [ ] **Step 1: Write the failing tests** — `TestPagesCosmic` (11 pages, exact order), `TestPagesTiling` (8 pages for `Sway`, `I3`, `Niri`), `TestFlowBackAtStart` (stays at 0), `TestFlowSkipRecords` (Skip on Language puts `PLanguage` in `Outcome.Skipped`), `TestSingle` (one page; Next ends the flow).
- [ ] **Step 2: Run, see them fail.** `go test ./... -run 'Pages|Flow|Single'`
- [ ] **Step 3: Implement `flow.go`.**
- [ ] **Step 4: Run, see them pass.**
- [ ] **Step 5: Commit** (`feat(welcome): the pages of the flow, per desktop`).

### Task 3: The window, and the Welcome, Shortcuts and Ready pages

**Files:**
- Create: `welcome/ui.go` (the `UI` type: `Layout(gtx)`, frame, footer, page dispatch), `welcome/page_welcome.go`, `welcome/page_shortcuts.go`, `welcome/page_ready.go`, `welcome/shortcuts.go`, `welcome/shortcuts_test.go`, `welcome/screenshot_test.go`, `welcome/testdata/*.png`
- Modify: `welcome/main.go` (open the window)

**Interfaces:**
- Consumes: `guikit.Theme`, `Steps`, `Frame`, `Footer`, `Action`, `Slide`, `Scaled`, `WindowKeys`, `ChoiceCard`, `Heading`, `Mark`; `Flow`, `Outcome` (Task 2).
- Produces:
  - `type Hooks struct { Invalidate func(); Logout func() error; OpenShortcutSheet func() error; Finish func(Stamp) }` — every side effect the UI triggers goes through a hook, so screenshot tests pass fakes.
  - `newUI(flow Flow, d Desktop, sh *text.Shaper, h Hooks) *UI` and `(*UI).Layout(gtx layout.Context) layout.Dimensions`.
  - `ParseShortcuts(r io.Reader) ([]Shortcut, error)` with `type Shortcut struct { Keys []string; Action string }`, reading `/usr/share/isotope/shortcuts.txt` (generated by `tools/gen-shortcuts.py`; COSMIC's from `tools/shortcuts-cosmic.txt`). The page shows the rows for Super+Escape, Super+D, Super+F1, Super+Shift+E when present, as key caps.
  - The screenshot harness: copy `install-gui/screenshot_test.go`'s harness (headless GPU, tolerance, `-update`, `ISOTOPE_GUI_SKIP_SCREENSHOTS`), rendering `newUI` with fakes at 1440×900 and 1024×768.

- [ ] **Step 1: Failing tests** — `TestParseShortcuts` on each variant's real `rootfs-*/usr/share/isotope/shortcuts.txt` (non-empty; Super+Escape present on all four); screenshot cases `1-welcome`, `shortcuts-cosmic`, `shortcuts-sway`, `ready-all-applied`, `ready-language-changed` (shows "Log out to use Deutsch (Deutschland)" and a Log out button), `ready-apps-failed`, and `welcome-scale-2` (scale 2, 1440×900).
- [ ] **Step 2: Run, see them fail** (no testdata yet → the harness reports missing goldens).
- [ ] **Step 3: Implement** `ui.go` and the three pages. The Welcome page: the mark, "Welcome to Isotope", the edition and version from `/etc/isotope/release` (as `install-gui`'s `versionLine`), one paragraph, "Set up as new" selected. Window close (`app.DestroyEvent`) before Finish calls `Hooks.Finish(StampSkipped)`; Finish on Ready calls `Hooks.Finish(StampDone)` (or `StampAppsPending` when Apps was skipped for being offline, Task 4).
- [ ] **Step 4: Generate goldens with `-update`, inspect every PNG by eye** (open them with the Read tool), then run without `-update` and see them pass. Use the Docker recipe PR #121 used (Ubuntu 24.04, the Mesa packages from `ci.yml`, Go 1.25.0) if this box lacks EGL headers.
- [ ] **Step 5: Commit** (`feat(welcome): the window, with the Welcome, Shortcuts and Ready pages`).

### Task 4: Apps

**Files:**
- Create: `rootfs/usr/share/isotope/first-run-apps.list`, `welcome/applist.go`, `welcome/flatpak.go`, `welcome/dockpin.go`, `welcome/page_apps.go`, tests for each, `welcome/testdata/apps-*.png`
- Modify: `welcome/ui.go`, `welcome/main.go` (wire the real runner)

**Interfaces:**
- Consumes: `Hooks`, `Outcome`, `Desktop`.
- Produces:
  - `type App struct { Group, ID, Label string; Preselected bool }`; `ParseAppList(r io.Reader) ([]App, error)` (`browser` group = one choice or none; `extra` = any).
  - `type Runner interface { Run(ctx context.Context, name string, args ...string) ([]byte, error); Start(name string, args ...string) (Process, error) }` with `Process` exposing stdout lines and `Wait() error` — the one exec seam, reused by later tasks.
  - `InstalledApps(r Runner) (map[string]bool, error)` (`flatpak --user list --app --columns=application`); `FlathubReady(r Runner) bool` (the same rule as today's `flathub_ready`: remote `flathub` present and not `disabled`).
  - `type Installer struct` with `Start(ids []string)`, `Progress() (done, total int, current string)`, `Result() (ok, failed []string, finished bool)`. It starts `flatpak --user install --noninteractive flathub <id>` per app in a child with `Setpgid`/new session so it survives the window closing; when the window is gone it sends today's completion notification (via `busctl --user call org.freedesktop.Notifications ...`, as the script's `notify`).
  - `PinToCosmicDock(home, appID string) error`: appends `appID` to `~/.config/cosmic/com.system76.CosmicAppList/v1/favorites` (RON list), creating it from `/usr/share/cosmic/com.system76.CosmicAppList/v1/favorites` when missing; no duplicate.

- [ ] **Step 1: Failing tests** —
  - `TestParseAppListMatchesScript`: parsing `rootfs/usr/share/isotope/first-run-apps.list` gives exactly today's nine rows (Firefox on; Chromium, Brave, LibreWolf off; Loupe, VLC, TextEditor, Papers on; Flatseal off).
  - `TestFlathubReady`: fake runner output with `flathub\t` → true; `flathub\tdisabled` → false; no flathub → false.
  - `TestInstallerReportsFailures`: fake runner where `org.videolan.VLC` exits 1 → `Result()` = ok without VLC, failed `[org.videolan.VLC]`, finished.
  - `TestInstallerDetaches`: the started process attributes set a new process group/session (assert on the `SysProcAttr` the real `Start` builds, through a small exported constructor used by both).
  - `TestPinToCosmicDock`: missing user file + system default without firefox → user file = default + `org.mozilla.firefox`; pinning twice → one entry.
  - `TestOfflineSkipSetsAppsPending`: Apps page in its offline state, Skip → `Outcome` marks apps pending, and Finish writes `StampAppsPending`; `ShouldOpenAtLogin` stays false, and `ShouldRemindApps(stateDir) bool` is true only for `StampAppsPending`.
  - Screenshots: `apps-default`, `apps-installed-greyed` (Firefox installed), `apps-offline`, `apps-installing` (sidebar shows "Apps 2/5").
- [ ] **Step 2: Run, see them fail.**
- [ ] **Step 3: Implement.** `--login` with `StampAppsPending` and a network: send the reminder notification whose default action runs `isotope welcome apps` (as `isotope-welcome`'s script waits for `ActionInvoked` today; use godbus instead of `gdbus monitor`).
- [ ] **Step 4: Run, see them pass; goldens generated and inspected.**
- [ ] **Step 5: Commit** (`feat(welcome): the Apps page, with installs that outlive the window`).

### Task 5: Network

**Files:**
- Create: `welcome/nm.go` (godbus client), `welcome/nm_test.go`, `welcome/page_network.go`, `welcome/testdata/network-*.png`

**Interfaces:**
- Produces:
  - `type Network interface { State() (NetState, error); AccessPoints() ([]AccessPoint, error); Connect(ssid, password string, hidden bool) error }`; `type NetState struct { Connected bool; Name string; HasWifi bool }`; `type AccessPoint struct { SSID string; Strength uint8; Secured bool }`.
  - `nmNetwork` implements it over `org.freedesktop.NetworkManager` (Connected = `State` is `NM_STATE_CONNECTED_GLOBAL` (70); `AddAndActivateConnection` with `802-11-wireless-security.key-mgmt = wpa-psk` for secured APs).
  - `dedupeAPs([]AccessPoint) []AccessPoint` — one row per SSID (strongest), hidden (empty SSID) dropped, sorted by strength.

- [ ] **Step 1: Failing tests** — `TestDedupeAPs` (two "Home" at 40 and 80 → one at 80; empty SSID dropped; order by strength); screenshots `network-connected` ("Connected to Home"), `network-wifi-list`, `network-password` (password field for "Home", Connect), `network-error` (inline "Wrong password"), `network-no-device` ("Plug in a network cable, or continue without a network").
- [ ] **Step 2–4: Run fail, implement page + client, run pass, goldens inspected.** The page uses a fake `Network` in tests; `main` passes `nmNetwork`. Continue is always enabled (offline allowed).
- [ ] **Step 5: Commit** (`feat(welcome): the Network page over NetworkManager`).

### Task 6: Language, with the daemon's SetLocale

**Files:**
- Modify: `update/dbus.go` (introspection XML, method), `update/` new `settings_bus.go` and `settings_bus_test.go`, `rootfs/usr/share/polkit-1/actions/io.kairos.isotope-update.policy` (new action `io.kairos.isotope-update.settings`), `AGENTS.md`/`CLAUDE.md` (the "Hadron has no localed" line, per the spec; and the profile.d note if Step 5 changes it)
- Create: `welcome/locale.go`, `welcome/locale_test.go`, `welcome/page_language.go`, `welcome/testdata/language-*.png`; a `/etc/profile.d/` or login-manager change only if Step 5 finds `LANG` does not reach a session

**Interfaces:**
- Produces (daemon): `SetLocale(lang string) *dbus.Error` under `actionSettings = "io.kairos.isotope-update.settings"`; refuses a `lang` not in `locale -a` output (the daemon runs `locale -a` through its own exec seam) and anything with characters outside `[A-Za-z0-9_.@-]`; writes `/etc/locale.conf` (`LANG=<lang>\n`, 0644, atomic rename) and `/oem/91_isotope_locale.yaml`:

  ```yaml
  name: "Isotope system locale"
  stages:
    initramfs:
      - name: "Write /etc/locale.conf"
        files:
          - path: /etc/locale.conf
            permissions: 0644
            content: |
              LANG=<lang>
  ```

  (match the stage and file style of the installer's keyboard cloud-config in `install-ui/core`; if it uses a different stage, use that one).
- Produces (welcome): `Locales(r Runner) ([]Locale, error)` with `type Locale struct { Code, Name string }` — UTF-8 locales only (`.UTF-8`/`.utf8`), named from an embedded table of language and territory names in their own language (`de_DE` → "Deutsch (Deutschland)"), falling back to the code; `CurrentLocale(localeConf string) string`; `type Settings interface { SetLocale(lang string) error }` with a godbus implementation calling the daemon.

- [ ] **Step 1: Failing tests** —
  - daemon: `TestSetLocaleRefusedWithoutAction` (fake authorizer says no → error, no files written), `TestSetLocaleRefusesUnknown` (`xx_YY.UTF-8` not in fake `locale -a` → error), `TestSetLocaleRefusesBadChars` (`de_DE.UTF-8\nevil`), `TestSetLocaleWritesBoth` (temp root: both files, exact content), `TestSetLocaleRefusesFlatpakCaller` (as the existing tests do for extensions).
  - welcome: `TestLocalesFiltersUTF8` (fake `locale -a` with `C`, `POSIX`, `de_DE.UTF-8`, `en_US.utf8` → two, named), `TestCurrentLocale`.
  - screenshots: `language-list`, `language-search` ("deu" typed), `language-readonly` (non-admin: "Ask an administrator to change the language").
- [ ] **Step 2–4: Run fail, implement, run pass** (`cd update && go test ./...`, `cd welcome && go test ./...`).
- [ ] **Step 5: Check that `LANG` reaches the session** on each login manager: read how ly's and greetd's PAM stacks and `start-desktop`, `niri-session`, `start-cosmic` build the environment (grep `rootfs*/` and the Dockerfile for `locale.conf`, `pam_env`, `LANG`). If any path ignores `/etc/locale.conf`, add `/etc/profile.d/isotope-locale.sh` that exports `LANG` from it, as `isotope-keyboard.sh` does for the layout, and say which path needed it in the commit.
- [ ] **Step 6: Commit** (`feat(welcome): the Language page, and SetLocale in isotope-updated`).

### Task 7: Appearance

**Files:**
- Create: `welcome/appearance.go`, `welcome/appearance_test.go`, `welcome/page_appearance.go`, `welcome/testdata/appearance-*.png`

**Interfaces:**
- Produces:
  - `SetColorScheme(r Runner, home string, d Desktop, dark bool) error`: runs `gsettings set org.gnome.desktop.interface color-scheme prefer-dark|prefer-light`; writes `gtk-application-prefer-dark-theme=true|false` under `[Settings]` in `~/.config/gtk-3.0/settings.ini` (keeping other keys); on `Cosmic` also writes `~/.config/cosmic/com.system76.CosmicTheme.Mode/v1/is_dark` (`true`/`false`).
  - `SetCosmicAccent(home string, accent Accent) error` and `SetCosmicFrosted(home string, on bool) error` with the keys `cosmic-initial-setup`'s `appearance.rs` writes at the COSMIC version the Dockerfile pins (read it: `gh api "repos/pop-os/cosmic-initial-setup/contents/src/page/appearance.rs?ref=epoch-<COSMIC_VERSION>"`); `Accents() []Accent` = the same list it offers.

- [ ] **Step 1: Failing tests** — `TestColorSchemeSway` (gsettings call with `prefer-light`; settings.ini keeps an existing `gtk-theme-name` line), `TestColorSchemeCosmic` (also `is_dark` = `false`), `TestCosmicAccent` (the key file content for one accent); screenshots `appearance-tiling` (two cards; the light card reads "Apps use a light theme; the bar and menus stay dark"), `appearance-cosmic` (cards + accents + frosted).
- [ ] **Step 2–4: Run fail, implement, run pass, goldens inspected.**
- [ ] **Step 5: Commit** (`feat(welcome): the Appearance page`).

### Task 8: COSMIC Layout, Workflow and Launcher

**Files:**
- Create: `welcome/cosmic.go`, `welcome/cosmic_test.go`, `welcome/page_cosmic.go`, `welcome/testdata/cosmic-*.png`

**Interfaces:**
- Produces:
  - `Layouts(dir string) ([]Layout, error)` reading `/usr/share/cosmic-layouts/*/` (`type Layout struct { ID, Name, Dir, Icon string }`, name from `layout.kdl` as upstream reads it).
  - `ApplyLayout(r Runner, home string, l Layout) error`: for each config directory in the layout, remove `~/.config/cosmic/<name>` and copy the layout's (file contents only, no metadata, as upstream's `copy_dir`), then `pkill -x cosmic-panel`.
  - `SetWorkflow(home string, w Workflow) error`, `SetLauncher(home string, l LauncherChoice) error` — the keys upstream's `workflow.rs` and `launcher.rs` write at the pinned COSMIC version; `Workflows()`, `LauncherChoices()` = its options.

- [ ] **Step 1: Failing tests** — `TestApplyLayoutReplacesDirs` (temp home with a stale `com.system76.CosmicPanel.Dock` dir and the `bottom-panel` fixture copied from the image's layout files → the Dock dir is gone, `com.system76.CosmicPanel.Panel/v1/anchor` = `Bottom`, and the runner saw `pkill -x cosmic-panel`), `TestLayoutsReadsBoth` (both shipped layouts found, in id order), `TestSetWorkflow`, `TestSetLauncher`; screenshots `cosmic-layout`, `cosmic-workflow`, `cosmic-launcher`.
- [ ] **Step 2–4: Run fail, implement, run pass, goldens inspected.**
- [ ] **Step 5: Commit** (`feat(welcome): COSMIC's layout, workflow and launcher pages`).

### Task 9: Accessibility

**Files:**
- Create: `welcome/scale.go`, `welcome/scale_test.go`, `welcome/page_access.go`, `welcome/testdata/access-*.png`
- Modify: `rootfs-sway/etc/sway/config` (an `include` of `~/.config/sway/isotope-scale.conf` if none exists for user overrides), `rootfs-i3/` session start (load `~/.Xresources` with `xrdb -merge` if it does not already), niri: decide per Step 3

**Interfaces:**
- Produces:
  - `Scales = []float64{1, 1.25, 1.5, 1.75, 2}`.
  - `ApplyScale(r Runner, home string, d Desktop, s float64) error`:
    - `Sway`: `swaymsg output '*' scale <s>`; persist `output * scale <s>` in `~/.config/sway/isotope-scale.conf`.
    - `Niri`: `niri msg output <name> scale <s>` for each output of `niri msg --json outputs`; persist in the user's niri config the way niri supports per-user overrides (Step 3 decides: niri reads `~/.config/niri/config.kdl` instead of `/etc/niri/config.kdl` when it exists — if so, persist with an `output` block in a file the Isotope config `include`s, adding that include to `rootfs-niri/etc/niri/config.kdl`).
    - `I3`: `xrdb -merge` with `Xft.dpi: <96*s>`; persist the line in `~/.Xresources`.
    - `Cosmic`: COSMIC's output scale config, as COSMIC Settings writes it (read `cosmic-settings`' display page at the pinned version; `cosmic-randr` if that is what it uses).
  - `SetLargerText(r Runner, home string, d Desktop, on bool) error`: `gsettings set org.gnome.desktop.interface text-scaling-factor 1.25|1.0`; on `Cosmic` also its interface font-size/density key.
  - `CurrentScale(r Runner, d Desktop) float64`.

- [ ] **Step 1: Failing tests** — one per desktop for `ApplyScale(…, 1.5)`: the command the fake runner saw and the persisted file content (`Xft.dpi: 144` for i3); `TestApplyScaleKeepsXresources` (other lines kept); `TestLargerText`; screenshots `access-tiling`, `access-i3` (the note "Some X11 apps need a restart"), `access-cosmic`.
- [ ] **Step 2: Run, see them fail.**
- [ ] **Step 3: Decide niri's persistence** by reading niri's config loading (its version in the Dockerfile) and `rootfs-niri/etc/niri/config.kdl`; record the choice in the commit message.
- [ ] **Step 4: Implement, add the config includes, run pass, goldens inspected.**
- [ ] **Step 5: Commit** (`feat(welcome): the Accessibility page`).

### Task 10: Into the image

**Files:**
- Modify: `Dockerfile` (a `welcome-build` stage after `install-gui-build`: `go test` with `ISOTOPE_GUI_SKIP_SCREENSHOTS=1`, then two builds — `-tags nox11,novulkan` → `/out/wayland/isotope-welcome`, `-tags novulkan` → `/out/x11/isotope-welcome` — each with the `--nonsense` exit-2 check; copy the wayland one in the sway, cosmic and niri rootfs stages and the x11 one in i3's; COSMIC: stop installing `cosmic-initial-setup`'s binary and autostart file, keep `/usr/share/cosmic-layouts`), `Dockerfile.agent` (remove `/usr/bin/isotope-welcome`, `/usr/share/isotope/first-run-apps.list`, and the session lines), `update/main.go` (a `welcome` subcommand that `exec`s `/usr/bin/isotope-welcome` with its arguments, hidden from nothing: listed in the usage text), `rootfs/usr/bin/isotope-menu` ("Install apps" → `isotope welcome apps`; new "Welcome" entry → `isotope welcome`; "Backup" stays as it is until the Backup plan), `rootfs-sway/etc/sway/config`, `rootfs-i3/etc/i3/config`, `rootfs-niri/etc/niri/config.kdl`, `rootfs-cosmic/etc/xdg/autostart/` (start `isotope-welcome --login` in place of `isotope-welcome` and `isotope-first-run-apps --login`; float rules for app ID / class `isotope-welcome`), `AGENTS.md` (the feature table rows, per the spec's "Daemon API" last paragraph)
- Delete: `rootfs/usr/bin/isotope-welcome` (the script), `rootfs-cosmic/etc/xdg/autostart/isotope-first-run-apps.desktop`
- Replace: `rootfs/usr/bin/isotope-first-run-apps` with a two-line script that runs `exec isotope welcome apps` (and ignores `--login`, which the new `--login` covers)
- Create: `test/welcome/check-welcome-image.sh`; Modify: `test/first-run/check-first-run-apps.sh` (it tested the old checklist; reduce it to the link behavior or delete it and say so)

- [ ] **Step 1: Write `check-welcome-image.sh`** (run against a built image, like the other `test/*/check-*.sh`): on sway, cosmic, niri `ldd /usr/bin/isotope-welcome` has no `libX11`; on i3 it has `libX11` and nothing "not found"; on all four `/usr/share/isotope/first-run-apps.list` exists and `isotope welcome --nonsense` exits 2; on cosmic there is no `/etc/xdg/autostart/com.system76.CosmicInitialSetup.desktop` and no `/usr/bin/cosmic-initial-setup`, and `/usr/share/cosmic-layouts/bottom-panel` exists; the session config of each variant starts `isotope-welcome --login` and no longer starts `isotope-first-run-apps --login`; the agent image has no `/usr/bin/isotope-welcome`.
- [ ] **Step 2: Run it against the current image, see it fail** (use the released `ghcr.io/kairos-community/isotope:<variant>-<tag>` images, as PR #119 did).
- [ ] **Step 3: Make the Dockerfile, rootfs and `update/` changes.** `cd update && go test ./...` for the subcommand.
- [ ] **Step 4: Build the `welcome-build` stage** with `docker buildx build --target welcome-build`; then, to run Step 1's checks without a full image build, copy the two binaries and the changed rootfs files onto the released images in throwaway images (as PR #119 did) and run the script. Expected: PASS. Say in the PR which checks ran this way and which only CI's full build covers.
- [ ] **Step 5: Commit** (`feat(welcome): start Isotope Welcome at first login on every desktop`).
- [ ] **Step 6: On hardware.** Install the CI-built COSMIC ISO on the laptop at 192.168.68.126 (or a VM if it cannot be reinstalled; ask before wiping it) and a VM per tiling variant: go through the flow, log out and in, check that nothing changed and that the flow does not return. Record results in the PR.
