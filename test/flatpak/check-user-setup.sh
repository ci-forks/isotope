#!/usr/bin/env bash
set -euo pipefail
IMAGE="${IMAGE:?set IMAGE to the Isotope image under test}"

test "$(docker run --rm "$IMAGE" stat -c %a /tmp)" = 1777

docker run --rm -i "$IMAGE" sh -eu -s <<'CLEAN'
useradd -m -u 1000 -s /bin/sh fixture
/usr/bin/isotope-user-setup
/usr/bin/isotope-user-setup
details=$(su - fixture -c \
  "flatpak --user remotes --show-disabled --columns=name,url,options")
printf "%s\n" "$details" | awk -F "\t" '
  $1 == "flathub" && $2 == "https://dl.flathub.org/repo/" &&
  $3 !~ /no-gpg-verify/ && $3 !~ /disabled/ { found=1 }
  END { exit !found }
'
su - fixture -c \
  "flatpak --user remote-info --show-commit flathub org.chromium.Chromium" \
  >/dev/null
# COSMIC lists the icon folders once per session, so they must exist at boot.
owner="$(id -u fixture):$(id -g fixture)"
test "$(stat -c '%u:%g %a' /home/fixture/.local/share/flatpak)" = "$owner 700"
for d in .local .local/share \
         .local/share/flatpak/exports .local/share/flatpak/exports/share \
         .local/share/flatpak/exports/share/icons \
         .local/share/flatpak/exports/share/icons/hicolor \
         .local/share/flatpak/exports/share/applications; do
    got=$(stat -c '%u:%g %a' "/home/fixture/$d")
    if [ "$got" != "$owner 755" ]; then
        echo "/home/fixture/$d is $got, want $owner 755" >&2
        exit 1
    fi
done
for d in /var/lib/flatpak/exports/share/icons/hicolor \
         /var/lib/flatpak/exports/share/applications; do
    test "$(stat -c '%u:%g %a' "$d")" = "0:0 755"
done
CLEAN

docker run --rm -i "$IMAGE" sh -eu -s <<'EXPORTS_IN_HOME'
useradd -m -u 1000 -s /bin/sh fixture
# A folder that exists keeps its mode.
su - fixture -c "mkdir -p -m 0700 ~/.local/share"
/usr/bin/isotope-user-setup
test "$(stat -c %a /home/fixture/.local/share)" = 700
test -d /home/fixture/.local/share/flatpak/exports/share/icons/hicolor
# A symlink planted in a home cannot make root create folders elsewhere.
useradd -m -u 1001 -s /bin/sh planted
mkdir -m 0755 /var/lib/victim
ln -s /var/lib/victim /home/planted/.local
/usr/bin/isotope-user-setup || true
if [ -e /var/lib/victim/share ]; then
    echo "followed a symlink in a home as root" >&2
    exit 1
fi
EXPORTS_IN_HOME

docker run --rm -i "$IMAGE" sh -eu -s <<'REPAIR'
useradd -m -u 1000 -s /bin/sh fixture
su - fixture -c \
  "flatpak --user remote-add --no-gpg-verify flathub https://dl.flathub.org/repo/"
su - fixture -c \
  "flatpak --user remote-modify --url=https://example.invalid/repo/ flathub"
/usr/bin/isotope-user-setup
details=$(su - fixture -c \
  "flatpak --user remotes --show-disabled --columns=name,url,options")
printf "%s\n" "$details" | awk -F "\t" '
  $1 == "flathub" && $2 == "https://dl.flathub.org/repo/" &&
  $3 !~ /no-gpg-verify/ && $3 !~ /disabled/ { found=1 }
  END { exit !found }
'
su - fixture -c \
  "flatpak --user remote-info --show-commit flathub org.chromium.Chromium" \
  >/dev/null
REPAIR

docker run --rm -i "$IMAGE" sh -eu -s <<'FAIL_CLOSED'
useradd -m -u 1000 -s /bin/sh fixture
su - fixture -c \
  "flatpak --user remote-add --no-gpg-verify flathub https://dl.flathub.org/repo/"
printf "%s\n" not-a-public-key > /usr/share/isotope/flathub.gpg
if /usr/bin/isotope-user-setup; then
    echo "malformed key unexpectedly succeeded" >&2
    exit 1
fi
details=$(su - fixture -c \
  "flatpak --user remotes --show-disabled --columns=name,url,options")
printf "%s\n" "$details" | awk -F "\t" '
  $1 == "flathub" && $3 ~ /disabled/ { found=1 }
  END { exit !found }
'
FAIL_CLOSED

docker run --rm -i "$IMAGE" sh -eu -s <<'BAD_DESCRIPTOR'
useradd -m -u 1000 -s /bin/sh fixture
printf '%s\n' '[not-a-flatpak-repository' > \
  /usr/share/isotope/flathub.flatpakrepo
if /usr/bin/isotope-user-setup; then
    echo "malformed descriptor unexpectedly succeeded" >&2
    exit 1
fi
details=$(su - fixture -c \
  "flatpak --user remotes --show-disabled --columns=name,url,options")
printf "%s\n" "$details" | awk -F "\t" '
  $1 == "flathub" && $3 !~ /disabled/ { bad=1 }
  END { exit bad }
'
BAD_DESCRIPTOR
