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
    check "mako resolves icons from Conflux" "grep -q '^icon-path=.*Conflux' /etc/mako/config"
fi

exit $fail
