#!/bin/sh
# Checks the patch support of tools/cosmic-build.sh with stubbed curl, cargo and
# just, no network and no docker:
#
# - patches/<repo>/*.patch is applied to the unpacked source;
# - libcosmic/*.patch is applied once to the libcosmic checkout the Cargo.lock
#   names, and a second build does not apply it again;
# - a patch that does not apply fails the build;
# - CARGO_TARGET_DIR, as cargo sees it, ends in 12 hex when there are patches
#   and is untouched when there are none;
# - a directory argument is copied instead of downloaded.
#
#   test/cosmic-build/test-cosmic-build-patches.sh
set -eu

here=$(cd "$(dirname "$0")" && pwd)
script=$(cd "$here/../.." && pwd)/tools/cosmic-build.sh
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
fail=0
ok() { echo "ok: $*"; }
bad() { echo "FAIL: $*" >&2; fail=1; }

mkdir -p "$work/bin" "$work/build" "$work/out" "$work/cargo/git/checkouts/libcosmic-x/abc1234" "$work/fixture/fake-repo"
cd "$work/fixture/fake-repo"
printf 'hello\n' > hello.txt
cat > Cargo.lock <<'LOCK'
[[package]]
name = "libcosmic"
source = "git+https://github.com/pop-os/libcosmic?rev=abc1234#abc1234def5678abc1234def5678abc1234def56"
[[package]]
name = "cosmic-config"
source = "git+https://github.com/pop-os/libcosmic?rev=abc1234#abc1234def5678abc1234def5678abc1234def56"
LOCK
(cd "$work/fixture" && tar -czf "$work/fake-repo.tar.gz" fake-repo)
printf 'lib\n' > "$work/cargo/git/checkouts/libcosmic-x/abc1234/lib.txt"

cat > "$work/bin/curl" <<STUB
#!/bin/sh
cat "$work/fake-repo.tar.gz"
STUB
cat > "$work/bin/cargo" <<STUB
#!/bin/sh
echo "\$1 \$CARGO_TARGET_DIR" >> "$work/cargo.log"
STUB
printf '#!/bin/sh\nexit 0\n' > "$work/bin/just"
chmod +x "$work/bin/"*

mkdir -p "$work/patches/fake-repo" "$work/patches/libcosmic"
cat > "$work/patches/fake-repo/0001-hello.patch" <<'P'
--- a/hello.txt
+++ b/hello.txt
@@ -1 +1 @@
-hello
+hello isotope
P
cat > "$work/patches/libcosmic/0001-lib.patch" <<'P'
--- a/lib.txt
+++ b/lib.txt
@@ -1 +1 @@
-lib
+lib isotope
P

run() {
    PATH="$work/bin:$PATH" COSMIC_VERSION=1 COSMIC_OUT="$work/out" COSMIC_BUILD_DIR="$work/build" \
        CARGO_HOME="$work/cargo" COSMIC_PATCHES="${PATCHES:-$work/patches}" CARGO_TARGET_DIR="$work/target" \
        sh "$script" "$@"
}
# The build removes the unpacked source, so keep a copy of what the repo
# patch produced through a just stub that records it.
cat > "$work/bin/just" <<STUB
#!/bin/sh
cp hello.txt "$work/hello.seen"
STUB

run fake-repo >"$work/log1" 2>&1 || { cat "$work/log1"; bad "build with patches failed"; }
[ "$(cat "$work/hello.seen" 2>/dev/null)" = "hello isotope" ] && ok "repo patch applied" || bad "repo patch not applied"
[ "$(cat "$work/cargo/git/checkouts/libcosmic-x/abc1234/lib.txt")" = "lib isotope" ] && ok "libcosmic patch applied" || bad "libcosmic patch not applied"
ls "$work/cargo/git/checkouts/libcosmic-x/abc1234"/.isotope-patched-* >/dev/null 2>&1 && ok "stamp written" || bad "no stamp"

run fake-repo >"$work/log2" 2>&1 || { cat "$work/log2"; bad "second build failed"; }
[ "$(grep -c 'patching libcosmic' "$work/log1" "$work/log2" | tr '\n' ' ')" = "$work/log1:1 $work/log2:0 " ] \
    && ok "libcosmic patch applied once across two builds" || bad "libcosmic patch applied again or never"

grep -Eq "^build $work/target/[0-9a-f]{12}\$" "$work/cargo.log" && ok "CARGO_TARGET_DIR ends in 12 hex" || bad "no hashed CARGO_TARGET_DIR: $(cat "$work/cargo.log")"

# A broken patch fails the build.
printf 'lib\n' > "$work/cargo/git/checkouts/libcosmic-x/abc1234/lib.txt"
rm -f "$work/cargo/git/checkouts/libcosmic-x/abc1234"/.isotope-patched-*
cat > "$work/patches/libcosmic/0002-broken.patch" <<'P'
--- a/nothere.txt
+++ b/nothere.txt
@@ -1 +1 @@
-x
+y
P
if run fake-repo >"$work/log3" 2>&1; then bad "broken patch did not fail the build"; else ok "broken patch fails the build"; fi
rm "$work/patches/libcosmic/0002-broken.patch"

# No patches: a missing dir and an empty dir leave CARGO_TARGET_DIR alone.
: > "$work/cargo.log"
mkdir "$work/empty"
for d in "$work/missing" "$work/empty"; do
    PATCHES=$d run fake-repo >"$work/log4" 2>&1 || { cat "$work/log4"; bad "build with patches dir $d failed"; }
done
[ "$(sort -u "$work/cargo.log")" = "build $work/target" ] && ok "no patches: CARGO_TARGET_DIR untouched" || bad "target dir changed without patches: $(cat "$work/cargo.log")"

# Directory mode (the checkout the failed build half-patched is reset first).
printf 'lib\n' > "$work/cargo/git/checkouts/libcosmic-x/abc1234/lib.txt"
rm -f "$work/cargo/git/checkouts/libcosmic-x/abc1234"/.isotope-patched-*
mkdir -p "$work/applets" "$work/patches/applets"
printf 'hello\n' > "$work/applets/hello.txt"
cp "$work/fixture/fake-repo/Cargo.lock" "$work/applets/"
cp "$work/patches/fake-repo/0001-hello.patch" "$work/patches/applets/"
run "$work/applets" >"$work/log5" 2>&1 || { cat "$work/log5"; bad "directory mode failed"; }
[ "$(cat "$work/hello.seen")" = "hello isotope" ] && ok "directory copied and patched" || bad "directory mode did not patch"

exit $fail
