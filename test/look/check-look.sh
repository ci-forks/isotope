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

exit $fail
