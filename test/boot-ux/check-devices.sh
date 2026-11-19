#!/usr/bin/env bash
# Assert the serial getty and the TPM device owners are right in a built image.
#
#   test/boot-ux/check-devices.sh [IMAGE]
#
# Two defects found on a laptop running an installed image:
#   1. console=ttyS0 is on every boot entry, so serial-getty@ttyS0 starts on a
#      computer whose ttyS0 is an empty 8250 slot. agetty failed with an I/O
#      error every 10 seconds for the whole uptime. The drop-in skips the getty
#      on a port of type 0 and keeps it on a port with a UART (QEMU, servers).
#   2. The TPM udev rules name the tss user and group, which did not exist, so
#      /dev/tpm0 and /dev/tpmrm0 stayed root-only.
# Run for every variant (run.sh) and for the server (the Makefile).
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

D=/usr/lib/systemd/system/serial-getty@.service.d/10-isotope-port-present.conf
P=/usr/libexec/isotope-serial-present
check "serial-getty drop-in runs the port check as its ExecCondition" \
    "grep -qx 'ExecCondition=$P %I' $D && test -x $P"
# A fake /sys/class/tty: ttyS0 has a 16550A (type 4), ttyS1 is an empty slot
# (type 0), hvc0 is not a serial_core port and has no type file.
check "port check: a UART passes, an empty 8250 slot fails, no type file passes" \
    "d=\$(mktemp -d) && mkdir \$d/ttyS0 \$d/ttyS1 \$d/hvc0 && echo 4 > \$d/ttyS0/type && echo 0 > \$d/ttyS1/type && $P ttyS0 \$d && ! $P ttyS1 \$d && $P hvc0 \$d && $P ttyUSB9 \$d"
check "tss sysusers file is installed" "test -f /usr/lib/sysusers.d/tpm2-tss.conf"
check "tss user and group exist in the image" \
    "grep -q '^tss:' /etc/passwd && grep -q '^tss:' /etc/group"
# Every user and group a udev rule names must exist, or udev ignores the
# assignment and leaves the device root-only.
check "every OWNER= and GROUP= in the udev rules resolves" \
    "cat /usr/lib/udev/rules.d/*.rules /etc/udev/rules.d/*.rules 2>/dev/null | grep -v '^[[:space:]]*#' | grep -oE '(OWNER|GROUP):?=\"[^\"]+\"' | tr -d ':\"' | sort -u | while IFS== read -r k n; do if [ \$k = OWNER ]; then f=/etc/passwd; else f=/etc/group; fi; grep -q \"^\$n:\" \$f || { echo \"missing \$k \$n\" >&2; exit 1; }; done"

exit "$fail"
