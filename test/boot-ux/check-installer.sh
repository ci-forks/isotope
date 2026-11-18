#!/usr/bin/env bash
# Assert the installer TUI is present and wired up in a built image.
#
#   test/boot-ux/check-installer.sh [IMAGE]
set -euo pipefail
IMAGE="${1:-sway-desktop:dev}"
fail=0
check() {
    local desc="$1"; shift
    if docker run --rm "$IMAGE" sh -c "$*" >/dev/null 2>&1; then
        echo "PASS: $desc"
    else
        echo "FAIL: $desc"; fail=1
    fi
}

check "isotope-install-ui present in /usr/bin" 'test -x /usr/bin/isotope-install-ui'
# Both of the next two are guarded on the binary EXISTING. Written the obvious
# way ("! ldd ... | grep -q 'not found'", "! bin --nonsense") they pass against
# an image with no binary at all -- a missing file also produces no "not found"
# line and also exits non-zero. Statically: no PT_INTERP, so the ELF carries no
# dynamic-loader path; a dynamically linked build embeds "/lib64/ld-linux-*".
check "isotope-install-ui is static (no interpreter needed)" \
    'test -x /usr/bin/isotope-install-ui && ! grep -qa "ld-linux" /usr/bin/isotope-install-ui'
check "rejects unknown arguments with exit 2" \
    '/usr/bin/isotope-install-ui --nonsense >/dev/null 2>&1; [ $? -eq 2 ]'
check "openssl available for password hashing" 'command -v openssl'
check "old shell installer removed" '! test -e /usr/local/bin/isotope-install'
check "installer drop-in points at isotope-install-ui" \
    'grep -q "/usr/bin/isotope-install-ui" /system/oem/90_desktop_installer.yaml'
# immucore decides the install intent and writes it as these files; Kairos'
# 52_installer.yaml reads only them. The cmdline is no guard: an installed
# Trusted Boot system keeps install-mode on it.
check "drop-in is gated on the install sentinels" \
    'grep -q "/run/cos/install_mode_interactive" /system/oem/90_desktop_installer.yaml && grep -q "\[ -f /run/cos/install_mode \]" /system/oem/90_desktop_installer.yaml'
check "hooks the interactive installer service" \
    'grep -q "kairos-interactive.service.d/10-desktop.conf" /system/oem/90_desktop_installer.yaml'
check "no stage reads install-mode from the cmdline" \
    '! grep -rlq "grep -q \"install-mode\" /proc/cmdline" /system/oem/'
check "the Kairos stage this hooks reads the same sentinels" \
    'grep -q "/run/cos/install_mode_interactive" /system/oem/52_installer.yaml'

# nmtui is the way onto a network from a text console, for when the graphical
# session does not start. Every edition ships it, and this script runs against
# the desktop and the server images. Each check needs the binary to exist, for
# the reason given above.
check "nmtui present in /usr/bin" 'test -x /usr/bin/nmtui'
check "nmtui finds all its libraries in the image (newt, slang)" \
    'test -x /usr/bin/nmtui && ! ldd /usr/bin/nmtui 2>&1 | grep -q "not found"'
check "nmtui runs (--help)" 'test -x /usr/bin/nmtui && /usr/bin/nmtui --help'

# The graphical installer, on the desktop variants. Each of these stopped it
# on the live ISO before the first frame (cage, then the GUI exited and the
# text installer took over).
if docker run --rm "$IMAGE" test -x /usr/bin/isotope-install-gui >/dev/null 2>&1; then
    check "cage present next to the GUI" 'test -x /usr/bin/cage'
    # libxkbcommon reads the locale's Compose table; Gio stops without one.
    check "Compose tables for the GUI (libX11 locale data)" \
        'test -s /usr/share/X11/locale/compose.dir && test -s /usr/share/X11/locale/en_US.UTF-8/Compose'
    # Gio asks for left_ptr and the other X11 names; every link must land on
    # an Adwaita cursor in this image.
    check "installer cursor theme resolves (left_ptr and the rest)" \
        'd=/usr/share/icons/isotope-installer/cursors; test -e $d/left_ptr && [ -z "$(find -L $d -type l)" ]'
fi

exit "$fail"
