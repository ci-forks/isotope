#!/usr/bin/env bash
# The checks are strings that the inner sh expands, on purpose.
# shellcheck disable=SC2016
# Check that a desktop image can run desktop OpenGL through EGL, the way
# Xwayland's glamor and GTK do it with libepoxy.
#
#   test/graphics/check-desktop-gl.sh IMAGE
#
# libepoxy takes the GL entry points of a desktop GL context made through EGL
# from libOpenGL.so.0 (glvnd) or libGL.so.1 (Mesa's GLX library), and aborts
# the process when neither is there. On the Wayland variants that killed
# Xwayland on every real GPU, so the COSMIC greeter showed only a cursor. A VM
# without 3D never shows it: glamor does not start on llvmpipe there.
#
# The Wayland variants (sway, cosmic, niri) get glvnd: libOpenGL.so.0 and
# libEGL.so.1 from glvnd, Mesa as the EGL vendor. i3 and the agent keep
# Mesa's own libGL.so.1 (GLX for XLibre). On every image, gl-probe.c makes a
# desktop GL context on llvmpipe (EGL surfaceless platform, no GPU and no
# display needed) and calls glGetString through libepoxy, and does the same
# with an OpenGL ES 2 context, which the compositors render with. The probes
# run as root and as an unprivileged user.
#
# The probe is compiled with the Hadron toolchain the image is built with
# (Dockerfile.probe); it needs only libc.
set -euo pipefail
IMAGE="${1:?usage: $0 IMAGE}"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(dirname "$(dirname "$HERE")")"
fail=0
check() {
    local desc="$1"; shift
    if docker run --rm --entrypoint sh "$IMAGE" -c "$*" >/dev/null 2>&1; then
        echo "PASS: $desc"
    else
        echo "FAIL: $desc"; fail=1
    fi
}
libs() { # DESC PATH: PATH exists and every library it needs resolves
    check "$1" "test -e $2 && ldd $2 >/tmp/ldd 2>&1 && ! grep -q 'not found' /tmp/ldd"
}

flavor="$(docker run --rm --entrypoint sh "$IMAGE" -c \
    'sed -n "s/^FLAVOR=//p" /etc/isotope/release' 2>/dev/null || true)"
echo "==> desktop GL: $IMAGE (flavor ${flavor:-unknown})"

libs "libepoxy finds its libraries" /usr/lib/libepoxy.so.0
libs "libEGL finds its libraries" /usr/lib/libEGL.so.1
case "$flavor" in
    i3|agent)
        libs "Mesa's libGL.so.1 (GLX) finds its libraries" /usr/lib/libGL.so.1
        ;;
    *)
        libs "glvnd's libOpenGL.so.0 finds its libraries" /usr/lib/libOpenGL.so.0
        libs "glvnd's libGLESv2.so.2 finds its libraries" /usr/lib/libGLESv2.so.2
        libs "Mesa's EGL vendor library finds its libraries" /usr/lib/libEGL_mesa.so.0
        check "Mesa is a glvnd EGL vendor" \
            'grep -q "\"library_path\" *: *\"libEGL_mesa.so.0\"" /usr/share/glvnd/egl_vendor.d/50_mesa.json'
        check "libEGL.so.1 is glvnd's, not a second Mesa libEGL" \
            'ldd /usr/lib/libEGL.so.1 | grep -q libGLdispatch.so.0'
        # glvnd reads the vendor files as the session user (cosmic-greeter,
        # the desktop user): with a directory that only root can enter it
        # finds no vendor, and the compositor gets no EGL platform.
        check "the glvnd vendor directories and file are world-readable" \
            'for d in /usr/share/glvnd /usr/share/glvnd/egl_vendor.d; do
                 case "$(stat -c %A "$d")" in d??????r?x) ;; *) exit 1 ;; esac
             done
             case "$(stat -c %A /usr/share/glvnd/egl_vendor.d/50_mesa.json)" in
                 -??????r??) ;; *) exit 1 ;;
             esac'
        ;;
esac

tmp="$(mktemp -d)"
chmod 755 "$tmp" # the unprivileged probe run must reach it
trap 'rm -rf "$tmp"' EXIT
hadron="$(awk -F= '/^ARG HADRON_VERSION=/{print $2; exit}' "$REPO/Dockerfile")"
if ! docker build -q --build-arg HADRON_VERSION="$hadron" -f "$HERE/Dockerfile.probe" \
        --output "type=local,dest=$tmp" "$HERE" >/dev/null; then
    echo "FAIL: could not build gl-probe"; exit 1
fi
probe() { # DESC USER [gles]: run gl-probe on llvmpipe in the image as USER
    local desc="$1" user="$2" out rc
    shift 2
    set +e
    out="$(docker run --rm --user "$user" --entrypoint /probe/gl-probe -v "$tmp:/probe:ro" \
        -e EGL_PLATFORM=surfaceless -e LIBGL_ALWAYS_SOFTWARE=1 \
        -e MESA_SHADER_CACHE_DISABLE=true "$IMAGE" "$@" 2>&1)"
    rc=$?
    set -e
    printf '%s\n' "$out" | while IFS= read -r l; do echo "    $l"; done
    if [ "$rc" -eq 0 ]; then
        echo "PASS: $desc"
    else
        echo "FAIL: $desc (exit $rc)"; fail=1
    fi
}
# As root, and as an unprivileged user (65534, nobody), like the greeter and
# the desktop session: a GL file that only root can read fails only there.
for user in 0:0 65534:65534; do
    probe "an OpenGL ES 2 context through EGL and libepoxy answers glGetString (uid ${user%%:*})" "$user" gles
    probe "a desktop GL context through EGL and libepoxy answers glGetString (uid ${user%%:*})" "$user"
done
exit $fail
