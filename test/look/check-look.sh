#!/usr/bin/env bash
# Assert the Isotope look is in a built image.
#
#   test/look/check-look.sh IMAGE DESKTOP
#
# e.g. test/look/check-look.sh sway-desktop:dev sway
#      test/look/check-look.sh cosmic-desktop:dev cosmic
#
# Prints PASS:/FAIL: lines and exits 1 on any FAIL. DESKTOP is sway, i3, niri
# or cosmic; checks for one variant are guarded by it.
set -euo pipefail
IMAGE="${1:-sway-desktop:dev}"
DESKTOP="${2:-sway}"
fail=0
check() {
    local desc="$1"; shift
    if docker run --rm "$IMAGE" sh -c "$*" >/dev/null 2>&1; then
        echo "PASS: $desc"
    else
        echo "FAIL: $desc"; fail=1
    fi
}

# --- Icons: Conflux on every variant ---------------------------------------
C=/usr/share/icons/Conflux
check "Conflux has an index.theme"            "test -f $C/index.theme"
check "Conflux inherits Cosmic then hicolor"  "grep -qx 'Inherits=Cosmic,hicolor' $C/index.theme"
check "every inherited theme is installed" \
    "test -f $C/index.theme && for t in \$(sed -n 's/^Inherits=//p' $C/index.theme | tr ',' ' '); do test -f /usr/share/icons/\$t/index.theme || exit 1; done"
# With -L, -type l matches only links that do not resolve (BusyBox find has no -xtype).
check "every Conflux link resolves"           "test -d $C && test -z \"\$(find -L $C -type l)\""

# Conflux is GPL-3.0: ship its licence and where it came from, and only the
# theme itself, not the upstream repo's build files and previews.
L=/usr/share/licenses/conflux
check "Conflux's GPL-3.0 LICENSE is installed" \
    "grep -q 'GNU GENERAL PUBLIC LICENSE' $L/LICENSE && test -s $L/CREDITS.md"
# The pin comes from the Dockerfile, so the check follows a bump.
pin=$(sed -n 's/^ARG CONFLUX_COMMIT=//p' "$(dirname "$0")/../../Dockerfile")
if [ -z "$pin" ]; then echo "FAIL: no CONFLUX_COMMIT in the Dockerfile"; fail=1; else
    check "Conflux's SOURCE names the pinned commit $pin" "grep -q '$pin' $L/SOURCE"
fi
check "Conflux holds only the theme (no Workspace, images, preview.png, update.sh, .github)" \
    "for n in Workspace images preview.png update.sh .github README.md; do test ! -e $C/\$n || exit 1; done"
check "the Conflux alias links are listed in conflux-links.txt" "test -s /usr/share/isotope/conflux-links.txt"

# An app ID resolves if Conflux has apps/scalable/<id>.svg, or it is listed in
# conflux-missing.txt (an app with no fitting Conflux icon).
# shellcheck disable=SC2016 # expanded later, inside the image
resolve='
id_ok() {
    test -e /usr/share/icons/Conflux/apps/scalable/$1.svg && return 0
    grep -q "^$1 " /usr/share/isotope/conflux-missing.txt
}'
check "first-run apps have a Conflux icon or are listed as missing" \
    "$resolve; test -f /usr/share/isotope/first-run-apps.list && grep -v '^#' /usr/share/isotope/first-run-apps.list | cut -d'|' -f2 | while read -r id; do [ -n \"\$id\" ] || continue; id_ok \$id || { echo \$id; exit 1; }; done"
if [ "$DESKTOP" = cosmic ]; then
    F=/usr/share/cosmic/com.system76.CosmicAppList/v1/favorites
    check "COSMIC favorites have a Conflux icon or are listed as missing" \
        "$resolve; test -f $F && grep -o '\"[^\"]*\"' $F | tr -d '\"' | while read -r id; do id_ok \$id || { echo \$id; exit 1; }; done"
fi

check "gsettings icon-theme is Conflux" \
    "test \"\$(gsettings get org.gnome.desktop.interface icon-theme)\" = \"'Conflux'\""
# GTK's window buttons go on the left on COSMIC only (its own schema override),
# as its patched libcosmic does; the tiling variants keep GTK's default.
if [ "$DESKTOP" != cosmic ]; then
    check "gsettings button-layout is not COSMIC's left layout" \
        "v=\$(gsettings get org.gnome.desktop.wm.preferences button-layout) && test -n \"\$v\" && test \"\$v\" != \"'close,minimize,maximize:'\""
fi
check "fuzzel uses Conflux" "grep -qx 'icon-theme=Conflux' /etc/xdg/fuzzel/fuzzel.ini"
if [ "$DESKTOP" = i3 ]; then
    check "dunst uses Conflux" "grep -qx '[[:space:]]*icon_theme = Conflux' /etc/dunst/dunstrc"
    check "dunst looks icons up recursively" \
        "grep -qx '[[:space:]]*enable_recursive_icon_lookup = true' /etc/dunst/dunstrc"
fi
if [ "$DESKTOP" = sway ]; then
    # mako globs <icon-path>/*/<number>/<name>.*, so test that glob for an app
    # icon (a link) and a status icon, with the icon-path from its own config.
    check "mako finds an app icon and a status icon in its icon-path" \
        "p=\$(sed -n 's/^icon-path=//p' /etc/mako/config | tr ':' ' '); test -n \"\$p\"; for n in org.videolan.VLC protonvpn-tray; do f=; for d in \$p; do for x in \$d/*/[0-9]*/\$n.*; do test -e \"\$x\" && f=\$x; done; done; test -n \"\$f\" || exit 1; done"
fi

# --- COSMIC theme: the Isotope theme as COSMIC's system defaults ------------
if [ "$DESKTOP" = cosmic ]; then
    T=/usr/share/cosmic/com.system76.CosmicTheme
    check "COSMIC theme dirs exist (dark, light, builders)" \
        "for d in Dark Light Dark.Builder Light.Builder; do test -s $T.\$d/v2/accent || exit 1; done"
    check "COSMIC defaults to dark" "test \"\$(cat $T.Mode/v1/is_dark)\" = true"
    # The night accent of theme/palette.toml, as the hex COSMIC's config writes
    # (RON: Some("#E9E7C1FF")). Compared case-insensitively.
    accent_of() { (cd "$(dirname "$0")/../.." && awk -v sec="[$1]" '$0==sec{n=1;next} /^\[/{n=0} n && $1=="accent"{gsub(/"/,"",$3); print toupper(substr($3,2,6)); exit}' theme/palette.toml); }
    for m in night:Dark day:Light; do
        want=$(accent_of "${m%%:*}"); M=${m##*:}
        if [ -z "$want" ]; then echo "FAIL: no ${m%%:*} accent in theme/palette.toml"; fail=1; continue; fi
        check "COSMIC $M builder accent is the Isotope accent #$want" \
            "grep -qi '#${want}ff' $T.$M.Builder/v2/accent"
        check "COSMIC $M theme accent is the Isotope accent #$want" \
            "grep -qi 'base: \"#${want}ff\"' $T.$M/v2/accent"
    done
fi

# --- COSMIC toolkit, compositor, panels and the Isotope layout -------------
if [ "$DESKTOP" = cosmic ]; then
    K=/usr/share/cosmic/com.system76.CosmicTk/v1
    check "CosmicTk icon_theme is Conflux"     "grep -qx '\"Conflux\"' $K/icon_theme"
    check "CosmicTk interface_font is Inter Variable" "grep -q 'family: \"Inter Variable\"' $K/interface_font"
    check "CosmicTk monospace_font is JetBrains Mono" "grep -q 'family: \"JetBrains Mono\"' $K/monospace_font"
    check "CosmicTk header_size and interface_density are Standard" \
        "grep -qx Standard $K/header_size && grep -qx Standard $K/interface_density"
    # Window buttons on the left: every COSMIC binary that reads CosmicTk must
    # be built with the patched libcosmic (patches/cosmic/libcosmic), whose
    # CosmicTk has the window_controls_left key; at least one must read it.
    check "every ELF that reads CosmicTk knows window_controls_left (patched libcosmic)" \
        "n=0; for f in \$(find /usr/bin /usr/libexec -type f 2>/dev/null); do head -c4 \$f | grep -q ELF || continue; grep -qa com.system76.CosmicTk \$f || continue; n=\$((n+1)); grep -qa window_controls_left \$f || { echo \$f; exit 1; }; done; [ \$n -gt 0 ]"
    check "CosmicTk window_controls_left is true" "grep -qx true $K/window_controls_left"
    check "gsettings button-layout puts the window buttons on the left" \
        "test \"\$(gsettings get org.gnome.desktop.wm.preferences button-layout)\" = \"'close,minimize,maximize:'\""
    P=/usr/share/cosmic/com.system76.CosmicComp/v1
    check "CosmicComp active_hint is false"    "grep -qx false $P/active_hint"
    check "CosmicComp shadows tiled windows"   "grep -q 'shadow_tiled_windows: true' $P/appearance_settings"
    L=/usr/share/cosmic-layouts/isotope
    check "the Isotope layout has an icon and a name" "test -s $L/icon.png && grep -q '^name en=\"Isotope\"' $L/layout.kdl"
    check "the system panel defaults are the Isotope layout's (no drift)" \
        "set -- $L/com.system76.CosmicPanel*; test -d \$1 && for d in \"\$@\"; do diff -r \$d /usr/share/cosmic/\$(basename \$d) || exit 1; done"
    check "the Isotope layout is listed first (lowest id)" \
        "own=\$(sed -n 's/^id //p' $L/layout.kdl); test -n \"\$own\"; for f in /usr/share/cosmic-layouts/*/layout.kdl; do [ \$f = $L/layout.kdl ] && continue; o=\$(sed -n 's/^id //p' \$f); [ \"\$own\" -lt \"\$o\" ] || exit 1; done; test \$(ls -d /usr/share/cosmic-layouts/*/ | wc -l) -gt 1"
fi

exit $fail
