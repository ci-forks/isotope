#!/bin/sh
# isotope-first-run-apps is the old name of the first-run app picker, kept so
# habits, menus and old configs keep working: with or without --login it opens
# Isotope Welcome's Apps page (`isotope welcome apps`). The picker itself, its
# login gate and its stamps are Isotope Welcome's (welcome/, its Go tests).
# Runs in an Alpine container, with a fake isotope that logs its arguments.
#
#   test/first-run/check-first-run-apps.sh            run in alpine (docker)
#   test/first-run/check-first-run-apps.sh --inner    run here
set -eu

if [ "${1:-}" != --inner ]; then
    repo=$(cd "$(dirname "$0")/../.." && pwd)
    exec docker run --rm -v "$repo:/src:ro" "${ALPINE_IMAGE:-alpine:3.22}" \
        sh /src/test/first-run/check-first-run-apps.sh --inner
fi

src=${SRC:-/src}
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
mkdir -p "$work/bin"
printf '#!/bin/sh\necho "isotope $*" > "%s/log"\n' "$work" > "$work/bin/isotope"
chmod +x "$work/bin/isotope"
PATH="$work/bin:$PATH"

fail=0
for args in "" "--login"; do
    : > "$work/log"
    # shellcheck disable=SC2086
    sh "$src/rootfs/usr/bin/isotope-first-run-apps" $args
    got=$(cat "$work/log")
    if [ "$got" = "isotope welcome apps" ]; then
        echo "PASS: isotope-first-run-apps $args runs 'isotope welcome apps'"
    else
        echo "FAIL: isotope-first-run-apps $args ran [$got]" >&2; fail=1
    fi
done
exit "$fail"
