# Graphical installer for the desktop editions

## Intent

The desktop ISOs (sway, i3, cosmic, niri) start a graphical installer: a
window over the Isotope wallpaper, a step list on the left and one step on the
right, in the spirit of the macOS installer. It asks the same questions as the
text installer, in the same order, and installs the same system. The server
ISO keeps the text installer.

The look is set by the mocks:

- the canvas: https://claude.ai/artifact/RXKAfbtVkXf9ckpHZXqjNC (private until
  shared from its Share menu);
- the same nine screens as plain HTML pages, in
  [assets/graphical-installer/](assets/graphical-installer/index.html): open
  `index.html` in a browser (the fonts load from Google Fonts).

## What does not change

- **The steps, their rules and their texts.** Keyboard, time zone, computer
  name, user name, password, GitHub SSH keys, disk, encryption, passphrase,
  recovery key, review. The validation is the text installer's
  (`install-ui/wizard.go`, `encrypt.go`, `locale.go`, `recoverykey.go`), run
  from the same Go code.
- **What an install writes.** `RenderCloudConfig` writes the cloud-config, and
  `kairos-agent install` or `manual-install` runs it. The passphrase reaches
  kcrypt only through a 0600 file in `/run`. The recovery key is shown once,
  saved nowhere, and confirmed by typing one group back.
- **How the installer is started.** `90_desktop_installer.yaml` still points
  `kairos-interactive.service` (and `kairos-installer.service`) at
  `/usr/bin/isotope-install-ui`. That program decides between text and
  graphics; see "Starting".
- **Unattended installs.** A seeded config in `/oem`,
  `ISOTOPE_INSTALL_NONINTERACTIVE=1` and `--progress-only` (the agent appliance)
  skip the wizard as they do today. In phase 1 they keep the text progress
  screen.
- **Trusted Boot media.** `/run/cos/uki_install_mode` changes the encryption
  step the same way it changes the text one: no passphrase, and "Create a
  recovery key?" with the TPM text.

## The screens

The mocks have nine screens. The text installer's fourteen steps fold into
seven pages, because a window has room for several fields:

| Page | Text installer steps | Notes |
|---|---|---|
| 1. Welcome | (none) | The mark, one paragraph, "Shut down" and "Continue". |
| 2. Keyboard and time zone | keyboard, timezone | A search box and a list for the layout, a field to try it, a map and a field for the zone. |
| 3. Computer and account | hostname, username, password, ssh keys | Password typed twice. "3 keys found" checks the GitHub name before Continue. |
| 4. Disk | target disk | One card per disk: model, size, bus, device. A bar of what is on the chosen disk now, and a red line naming what will be erased. |
| 5. Encryption | encryption, passphrase, confirm, recovery key choice | Two cards (don't encrypt, encrypt), the passphrase twice with the strength meter, a "recovery key" checkbox. |
| 6. Recovery key | show the key, confirm the key | Only when a key was asked for. 8 groups in a 4x2 grid; Continue stays disabled until the asked group is typed. |
| 7. Summary | review | Each choice with a "Change" link to its page; the disk in red; the button is "Erase and install", in the error colour. |
| 8. Installing | progress screen | The mark, a progress bar, the stages as a checklist, "Show details" for the log. |
| 9. Done | finish | "Restart". On failure: see "Errors". |

The sidebar lists the seven pages: done ones with a check, the current one
highlighted, later ones dimmed. Pages 1 to 7 can go back; nothing can go back
from page 8.

### What is new compared with the text installer

These are in the mocks and need code the text installer does not have:

- **Password confirmation.** A second field; Continue is disabled until both
  match. The text installer has no confirm for the account password.
- **What is on a disk.** `lsblk -J -b -o NAME,SIZE,FSTYPE,LABEL,PARTLABEL` per
  disk, drawn as a bar with one segment per partition. The warning names what
  it recognises (an NTFS partition labelled for Windows, an ext4 or btrfs
  root, a Kairos `COS_*` layout) and falls back to "Everything on this disk".
  The disk's model and bus come from sysfs (`device/model`, and where the
  disk's sysfs entry points: nvme, ata, usb, virtio, mmc). The parsing and
  the wording live in `install-ui/core/diskinfo.go`.
- **Time zone map.** A world map (Natural Earth 1:110m, public domain) drawn
  as vector shapes, with the zone's band highlighted and a dot on its main
  city. Clicking picks the nearest zone city. The field under it accepts
  typing, with the same list and filter as today: while it holds a search,
  the list of matches takes the map's place. The outlines are simplified
  from the 1:110m land polygons by `install-gui/worldmap_gen.go` (about
  20 KB, embedded); the cities come from tzdata's `zone.tab` and
  `zone1970.tab` (`core.ZoneCoords`, `core.NearestZone`).
- **"Try it" field.** It types with the chosen layout (see "Keyboard").
- **GitHub key lookup.** `https://github.com/NAME.keys`, once the name has
  stayed the same for 0.7 s, and again on Continue; the result never blocks
  the install (`core.CountGithubKeys`).
- **QR code for the recovery key.** A button that shows the key as a QR code,
  for a phone photo. It is drawn in the window only, like the key.
- **Time estimate.** "About N minutes left" from the stage weights in
  `install-ui/core/estimate.go`, next to the steps of `core/steps.go`, and
  the time spent so far. Shown only after the first
  30 seconds of "Installing the active system"; until then, nothing.

## Visual system

From the mocks. Colours are `theme/palette.toml` (night), the same as the
splash and GRUB:

| Role | Value |
|---|---|
| Desktop background | radial glow `#2c3a1c` at the top over `#15141c` to `#0e0d13` |
| Window | `#1f1e28`, 16 px corners, a 1 px `#f2f1f5` at 9% border, a large soft shadow |
| Sidebar | `#191820` |
| Field | `#2a2935`; focus border `#8fd12e` with a 3 px 18% ring |
| Accent (buttons, current step, selection) | `#8fd12e`; text on it `#15141c` |
| Glow (key text, done checks) | `#c6f36b`; checks fill `#4e9a06` |
| Text, secondary, dim | `#f2f1f5`, `#cbc8d6`, `#8f8ba3` |
| Error (erase warnings, Erase and install) | `#f7768e` |

- **Type.** Space Grotesk for page titles (28 px, 600), Inter for the rest (14 px
  body, 13 px labels), JetBrains Mono for device paths, layout codes and the
  recovery key. Inter and JetBrains Mono are already in the image
  (`isotope-fonts` stage); Space Grotesk (SIL OFL) is added there.
- **Mark.** The six-triangle Isotope mark, geometry from
  `tools/gen-wallpaper.py`, in the sidebar (30 px), on the welcome and done
  pages (about 140 px), and on the installing page with the pushed-out
  triangle lit, as the splash does at the flash.
- **Layout.** The window is 920x620 logical pixels, centred. On an output
  smaller than 1000x700 logical pixels it fills the output with no corners or
  shadow. The output scale comes from the compositor, so HiDPI panels get
  crisp text.
- **Motion.** Pages slide 24 px and fade over 180 ms; the progress bar eases.
  Nothing moves on its own except the installing page's spinner and the
  mark's slow breathing, as on the splash.

## Architecture

### One core, two front ends

`install-ui` today is one `package main`. It splits into:

- `install-ui/core`: the wizard as a state machine with no drawing:
  the steps, their validation, `Choice`, `RenderCloudConfig`, the disk and
  locale lists, the recovery key, and the install run (the child process, the
  log stream, `AdvanceStep`, the encryption pre, cleanup and post steps). It
  sends events (stage changed, log line, finished, failed). Every rule lives
  here, so the two installers cannot drift.
- `install-ui` (the text installer): Bubble Tea over `core`. Static, no cgo,
  on every image.
- `install-gui` (new): the graphical front end over `core`, built only for the
  desktop images.

The existing tests move with the code they test. `core` gets the tests that
are about rules today (`wizard_test.go` drives the Bubble Tea model; the rules
it checks become `core` tests, and a thinner test stays for each front end).

### Toolkit: Gio

The GUI is written with [Gio](https://gioui.org) (MIT/Unlicense):

- it is Go, so `install-gui` imports `core` directly, with no IPC and no
  second language;
- it draws everything itself, so the mocks can be matched exactly
  (custom cards, the disk bar, the map, the step list) without fighting a
  widget theme;
- it speaks Wayland and renders through EGL/OpenGL ES, which Mesa provides on
  both GPU lines, llvmpipe included, so it runs in a VM with no 3D and with
  `nomodeset` (simpledrm);
- it can render a frame headless (`gioui.org/app/headless`), which gives
  screenshot tests in `go test`.

It needs cgo and links libwayland-client, libEGL and libxkbcommon, which the
Dockerfile already builds. `install-gui` is therefore built in the Hadron
toolchain stage, not in the `golang:alpine` stage that builds the static text
installer.

Considered and not chosen:

- **GTK 3** (built already for waybar): its theme engine would need a full
  custom CSS theme to look like the mocks, and the code would be C or Go
  bindings over a second widget model.
- **A web page in a kiosk browser**: the closest to the mocks, but it ships a
  browser engine in every desktop image for one screen, and the browser would
  need a local server with the right to wipe a disk.
- **Qt/QML**: not built for Hadron today; the largest new dependency.

### The compositor: cage

The installer does not start the variant's desktop. It runs in
[cage](https://github.com/cage-kiosk/cage), a kiosk Wayland compositor that
shows one application full screen. cage is built on wlroots, which the
Dockerfile already builds for sway, and it is small (one C binary). It goes
in every desktop image, cosmic and niri included, because the ISO is the
image.

- cage runs on tty1 with a seat from libseat's builtin backend (the service
  runs as root), so no seatd or logind session is needed.
- `WLR_RENDERER` is left unset: wlroots picks GLES2 and falls back to its
  pixman renderer when there is no GPU, the `nomodeset` case.
- The cursor theme and the wallpaper colour come from the image. The mouse and
  touchpad work through libinput, as in the desktop.

X is not needed. On i3 the installed desktop runs XLibre, but the installer
does not depend on the variant's display server. It does need Wayland there:
the i3 image gets the Wayland client and server libraries and
libdisplay-info, and its Mesa (`mesa-x11`) is built with the Wayland platform
next to X11, so cage and the GUI run on i3 as on the other three.

cage is built from its release for the wlroots series the image has (0.2.0
for wlroots 0.18), with wlroots linked in as a meson subproject, so cosmic,
niri and i3, which do not use wlroots, need no libwlroots.

The agent image removes cage and the GUI: it installs unattended with
`--progress-only`, which never starts them.

## Starting

`isotope-install-ui` keeps its role as the program the Kairos services run.
Before it draws anything it decides:

1. Unattended (seeded config, `ISOTOPE_INSTALL_NONINTERACTIVE`,
   `--progress-only`): the text progress screen, as today.
2. `/usr/bin/cage` and `/usr/bin/isotope-install-gui` both present, and
   `isotope.installer=text` not on the kernel command line: exec
   `cage -s -- /usr/bin/isotope-install-gui`.
3. Otherwise: the text installer. This is the server, whose image has neither
   binary.

If cage exits before the GUI has written its "started" file to `/run` (no DRM
device, no seat, a driver hang), `isotope-install-ui` starts the text
installer on tty1 instead. The fallback happens before any question is
answered, so nothing is lost. A crash after that point shows the text
installer from the start with a line saying the graphical installer stopped;
an install already running is never started twice (the GUI writes
`/run/isotope-install/install-running` while its install runs, and
`isotope-install-ui` then stops with an error instead of starting the text
installer).

The GUI talks to `isotope-install-ui` with its exit status: 75 asks for a new
cage with another keyboard layout (see "Keyboard"), 76 for the text
installer's rescue shell on tty1 (see "Errors").

The GRUB menu of the desktop ISOs gets one more entry, "isotope (text
installer)", which adds `isotope.installer=text`. Safe graphics keeps the
graphical installer, on simpledrm.

## Keyboard

Choosing a layout must change what the keys type at once, in the "Try it"
field and in every field after it. cage takes its keymap from
`XKB_DEFAULT_LAYOUT` and `XKB_DEFAULT_VARIANT` when it starts and has no way
to change it later.

The first plan was for `install-gui` to keep its own `xkb_state`, built from
the chosen layout, and translate cage's raw keycodes itself. Gio does not
allow that: its Wayland backend keeps the xkb state in an internal package
(`gioui.org/app/internal/xkb`), loads only the keymap the compositor sends,
and gives the program translated key events and text, never keycodes. Doing
it anyway would mean a fork of Gio.

So the GUI restarts cage with the new layout. Picking a layout other than the
one cage was started with (a click on it, Enter in the search field, or
Continue) saves the answers to `/run/isotope-install/state.json` (0600; never
a password, a passphrase or a recovery key) and exits with status 75.
`isotope-install-ui` checks the layout with `core.ValidateKeyboard` and starts
cage again with `XKB_DEFAULT_LAYOUT` and `XKB_DEFAULT_VARIANT`. The GUI opens
where it was: on the keyboard page with the focus in "Try it", or on the
account page with the password fields empty. That costs a short flash.

The console keymap (`loadkeys`) and the cloud-config's keymap stage are set
as today, so the boot passphrase prompt gets the same layout.

## Errors

- **Field errors** sit under the field, in the error colour, with the text
  installer's wording. Continue stays enabled; pressing it focuses the first
  wrong field.
- **No disk.** Page 4 says no disk was found, with a "Rescan" button.
- **GitHub lookup.** "3 keys found", "no keys for that name" (a warning, not
  an error: the install can go on without keys), or "could not reach GitHub"
  (the same).
- **Install failure.** Page 8 turns into "Isotope could not be installed": the
  stage it stopped at, the last 200 lines of the log (selectable), and three
  buttons: "Save the log" (to a USB stick if one is mounted, else
  `/run/isotope-install/install.log`, where the GUI writes the whole log as
  it runs), "Open a terminal", and "Restart". "Open a terminal" starts foot
  as a second window in cage where the image has it (sway and niri); on i3
  and cosmic, whose terminals do not run in cage, the GUI exits with status
  76 and `isotope-install-ui` gives tty1 to `/bin/sh`, the text installer's
  rescue shell.

## Accessibility

- Every control is reachable with Tab and works with Enter and Space; Esc is
  "Go back"; the focus ring is the accent ring from the mocks.
- Text meets 4.5:1 on its background (the dim `#8f8ba3` on `#1f1e28` is
  5.1:1).
- The red warnings carry an icon and words, not colour alone.
- Ctrl+Plus and Ctrl+Minus scale the whole window by 25% steps, up to 200%.
- A screen reader is out of scope for phase 1. Gio has no AT-SPI bridge
  today; that is a known gap, and `isotope.installer=text` plus the console
  remains the accessible path.

## Testing

- **core:** the existing rule tests, moved; nothing about drawing.
- **Screenshots:** `go test` renders each page headless at 1440x900 and
  1024x768 into PNGs and compares them with golden files in
  `install-gui/testdata/`, within a small tolerance. `go test -update`
  rewrites them; a PR that changes a page shows the new PNGs.
- **Boot test:** a QEMU run of a desktop ISO (`test/boot-ux/`), driven with
  QMP: type through every page, install to a blank disk with
  `ISOTOPE_INSTALL_CMD` faked, and assert from screendumps that each page
  appeared and that the cloud-config written matches the text installer's
  for the same answers. One run with `nomodeset`.
- **Fallback test:** the same ISO with cage removed from `/usr/bin` at boot
  (a yip stage in the test medium) must show the text installer.

## Delivery

1. Split `install-ui` into `core` and the text front end. No behaviour
   change; the text installer's tests pass unchanged. One PR.
2. `install-gui` with the nine pages, the cage stage in the Dockerfile, the
   start-up decision, and the screenshot tests. The desktop ISOs switch to it;
   the server is untouched. One PR, with screenshots of every page.
3. The new parts: disk contents, the map, QR, estimate, the GitHub key
   lookup, Save the log and Open a terminal, the breathing mark.
4. Docs: README's install section and screenshots, `docs/DEVELOPMENT.md`,
   the feature table in `AGENTS.md` (the installer row: graphical on the four
   variants, text on the server and the agent).

Steps 2, 3 and 4 landed together, in one PR, with this note.

## Open questions

- **Gio and per-application keymaps.** Answered: Gio gives no raw keycodes,
  so the GUI restarts cage (see "Keyboard").
- **Image size.** cage, Gio's binary and Space Grotesk add a few MB to every
  desktop image. Measure in step 2.
- **The agent appliance.** It installs unattended behind `--progress-only`.
  A graphical progress page would suit it, but it is out of scope here.
- **Language.** The installer is English only, like the text one. A language
  page would come first in the list and needs the texts in a catalogue; not
  in this note.
