#!/usr/bin/env bash
# Boot ordering of a built image: what decides how soon the login screen
# appears, and that tty1 always has a way to log in.
#
#   test/boot-ux/check-boot-units.sh [IMAGE]
#
# - No boot stage waits for the network. The Kairos boot stage runs inside
#   cos-setup-boot.service, and ly and cosmic-greeter start after it, so a
#   `systemctl start` there of a unit ordered after network-online.target
#   held the login screen until the network came up or wait-online gave up
#   (120 s with no cable, or on Wi-Fi).
# - NetworkManager is the only network manager: systemd-networkd and its
#   wait-online (which cannot see Wi-Fi) are not enabled.
# - getty@tty1 is enabled, not masked: Kairos' own boot stage enables it, and
#   it is the login when no login screen runs. A login screen on tty1 takes
#   the VT with Conflicts=getty@tty1 and gives it back with OnFailure=.
#
# Works on every desktop image, the agent and the server (which has no login
# screen, so only the first three groups apply).
set -euo pipefail
IMAGE="${1:-sway-desktop:dev}"
fail=0
run() { docker run --rm --entrypoint sh "$IMAGE" -c "$1"; }
check() {
    local desc="$1"; shift
    if run "$*" >/dev/null 2>&1; then
        echo "PASS: $desc"
    else
        echo "FAIL: $desc"; fail=1
    fi
}

# Every unit a boot stage starts and waits for, that is ordered after
# network-online.target. Prints "FILE: UNIT"; empty is a pass.
# shellcheck disable=SC2016
waits='
for f in /system/oem/*.yaml; do
    grep -E "^[[:space:]]*-?[[:space:]]*systemctl start " "$f" | grep -v -- "--no-block" |
        sed -E "s/.*systemctl start //; s/(\|\||;|&&).*//" | tr " " "\n" | grep -v "^-" | grep . |
        while read -r u; do
            case "$u" in *.*) ;; *) u="$u.service" ;; esac
            t="${u%%@*}"; [ "$t" = "$u" ] || t="$t@.${u##*.}"
            for d in /usr/lib/systemd/system /etc/systemd/system; do
                cat "$d/$u" "$d/$t" "$d/$u.d/"*.conf "$d/$t.d/"*.conf 2>/dev/null
            done | grep -qE "^(After|Wants|Requires)=.*network-online\.target" && echo "$f: $u"
        done
done
true'
blocking="$(run "$waits" 2>/dev/null)" || { echo "FAIL: could not read the boot stages"; fail=1; }
if [ -z "$blocking" ]; then
    echo "PASS: no boot stage waits for a unit ordered after network-online.target"
else
    echo "FAIL: a boot stage waits for a unit ordered after network-online.target (use --no-block):"
    printf '  %s\n' "$blocking"
    fail=1
fi
# Same probe, on a stage that does wait, so a probe that finds nothing at all
# cannot pass the check above.
check "the probe finds a blocking start" \
    "mkdir -p /system/oem && printf 'stages:\n  boot:\n    - commands:\n        - systemctl start docker.service || true\n' > /system/oem/99_probe.yaml && test -f /usr/lib/systemd/system/docker.service && ($waits) | grep -q 99_probe.yaml"

ena() { echo "[ \"\$(systemctl --root=/ is-enabled $1 2>/dev/null)\" = enabled ]"; }
for u in systemd-networkd.service systemd-networkd.socket systemd-networkd-wait-online.service; do
    check "$u is not enabled (NetworkManager manages the network)" "! $(ena "$u") && ! ls /etc/systemd/system/*.wants/$u >/dev/null 2>&1"
done
check "NetworkManager and its wait-online are enabled" \
    "$(ena NetworkManager.service) && $(ena NetworkManager-wait-online.service)"

check "getty@tty1 is enabled, not masked" "$(ena getty@tty1.service)"
# Kairos' 09_systemd_services.yaml runs this at every boot; on a masked unit it
# fails and the boot logs "failed to run systemctl enable getty@tty1".
check "Kairos' 'systemctl enable getty@tty1' succeeds" "systemctl enable getty@tty1"

# The login screens on tty1: the template ly@ (sway, i3, niri, agent) and
# cosmic-greeter. UNIT FILE DROPIN GETTY
dm() {
    local unit="$1" file="$2" dropin="$3" getty="$4"
    if ! run "test -f $file" >/dev/null 2>&1; then return; fi
    check "$unit gives tty1 to its login screen (Conflicts=$getty)" "grep -qx 'Conflicts=$getty' $file"
    check "$unit gives tty1 back when it fails (OnFailure=$getty)" "grep -qx 'OnFailure=$getty' $dropin"
    check "getty@tty1 can start again after the login screen gave up" \
        "grep -qx 'StartLimitIntervalSec=0' /usr/lib/systemd/system/getty@tty1.service.d/20-isotope.conf"
}
dm ly@tty1 /usr/lib/systemd/system/ly@.service \
    /usr/lib/systemd/system/ly@.service.d/20-getty-fallback.conf 'getty@%i.service'
dm cosmic-greeter /usr/lib/systemd/system/cosmic-greeter.service \
    /usr/lib/systemd/system/cosmic-greeter.service.d/20-getty-fallback.conf getty@tty1.service

exit "$fail"
