#!/usr/bin/env bash
# Generate the Trusted Boot release keys, and store the ones CI needs as
# GitHub secrets. Run it on a trusted machine, not in CI. See
# docs/TRUSTED_BOOT_KEYS.md.
#
#   scripts/uki-release-keys.sh generate <dir>   new keys in <dir> (must not exist)
#   scripts/uki-release-keys.sh upload <dir>     the CI keys from <dir> as secrets
#
# Env:
#   REPO          repository for the secrets (default kairos-community/isotope)
#   DAYS          certificate lifetime for generate (default 3650)
#   AURORA_IMAGE  AuroraBoot image (default: the Makefile pin)
#   DRY_RUN=1     upload: print what would be set, set nothing
#
# upload sets UKI_DB_KEY, UKI_DB_PEM, UKI_TPM_PCR_KEY (PEM) and UKI_PK_AUTH,
# UKI_KEK_AUTH, UKI_DB_AUTH (base64 of the .auth files), the names
# release.yml's build-uki job reads. It never uploads PK.key or KEK.key:
# those only sign later changes to the keys enrolled in firmware, and belong
# offline.
set -euo pipefail
cd "$(dirname "$0")/.."

REPO="${REPO:-kairos-community/isotope}"
DAYS="${DAYS:-3650}"
AURORA_IMAGE="${AURORA_IMAGE:-$(make -s print-aurora-image)}"

die() { echo "uki-release-keys: $*" >&2; exit 1; }

ci_files=(db.key db.pem tpm2-pcr-private.pem PK.auth KEK.auth db.auth)

check_keys() { # dir
  local d="$1" f
  for f in "${ci_files[@]}"; do
    [ -s "$d/$f" ] || die "$d/$f is missing"
  done
  # The certificate must belong to the key, or every UKI is signed with a
  # key no enrolled machine trusts.
  [ "$(openssl x509 -in "$d/db.pem" -noout -pubkey | openssl sha256)" = \
    "$(openssl pkey -in "$d/db.key" -pubout | openssl sha256)" ] \
    || die "$d/db.pem is not the certificate of $d/db.key"
  openssl x509 -in "$d/db.pem" -noout -checkend 0 >/dev/null \
    || die "$d/db.pem has expired"
}

cmd="${1:-}"
dir="${2:-}"
[ -n "$cmd" ] && [ -n "$dir" ] || die "usage: $0 generate|upload <dir>"

case "$cmd" in
generate)
  [ ! -e "$dir" ] || die "$dir exists; machines enrolled with keys in it need them, so this will not overwrite it"
  mkdir -p "$dir"
  chmod 700 "$dir"
  # Microsoft's certificates stay in KEK and db (genkey's default): firmware
  # option ROMs (graphics cards, some NICs) are signed by Microsoft, and a
  # machine that cannot load them may not show a screen.
  docker run --rm -u "$(id -u):$(id -g)" -v "$(cd "$dir" && pwd)":/keys "$AURORA_IMAGE" \
    genkey -e "$DAYS" --output /keys "Isotope"
  check_keys "$dir"
  echo
  echo "Keys are in $dir. Next:"
  echo "  1. $0 upload $dir"
  echo "  2. Copy $dir to offline storage (two copies). PK.key and KEK.key never go"
  echo "     anywhere else: they are needed only to change the enrolled keys."
  echo "  3. Delete $dir from this machine."
  ;;
upload)
  [ -d "$dir" ] || die "no directory $dir"
  check_keys "$dir"
  set_secret() { # name, value-producing command...
    local name="$1"; shift
    if [ "${DRY_RUN:-}" = 1 ]; then
      echo "would set $name on $REPO ($("$@" | wc -c) bytes)"
    else
      "$@" | gh secret set "$name" --repo "$REPO"
      echo "set $name on $REPO"
    fi
  }
  set_secret UKI_DB_KEY      cat "$dir/db.key"
  set_secret UKI_DB_PEM      cat "$dir/db.pem"
  set_secret UKI_TPM_PCR_KEY cat "$dir/tpm2-pcr-private.pem"
  set_secret UKI_PK_AUTH     base64 -w0 "$dir/PK.auth"
  set_secret UKI_KEK_AUTH    base64 -w0 "$dir/KEK.auth"
  set_secret UKI_DB_AUTH     base64 -w0 "$dir/db.auth"
  echo
  echo "Turn the Trusted Boot line on for the next release with:"
  echo "  gh variable set TRUSTED_BOOT_RELEASE --body true --repo $REPO"
  ;;
*)
  die "unknown command $cmd (want generate or upload)"
  ;;
esac
