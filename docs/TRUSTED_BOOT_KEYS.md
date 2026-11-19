# Trusted Boot release keys

The Trusted Boot edition (`isotope-<flavor>-uki-<tag>-amd64.iso` and the
`<flavor>-uki-<tag>` upgrade images) is signed with keys this project
generates and holds. A machine installed from it boots only images signed with
these keys, and its disk key is sealed to them. Losing them means no more
upgrades for those machines; leaking them means anyone can sign an image those
machines will boot. Treat them that way.

The design is in
[superpowers/specs/2026-11-07-trusted-boot-edition-design.md](superpowers/specs/2026-11-07-trusted-boot-edition-design.md).

## The keys

| File | Where it lives | What it does |
|---|---|---|
| `db.key`, `db.pem` | GitHub secrets `UKI_DB_KEY`, `UKI_DB_PEM`, and offline | signs every UKI (the Secure Boot db key) |
| `tpm2-pcr-private.pem` | secret `UKI_TPM_PCR_KEY`, and offline | signs the PCR policy the disk key is sealed to |
| `PK.auth`, `KEK.auth`, `db.auth` | secrets `UKI_PK_AUTH`, `UKI_KEK_AUTH`, `UKI_DB_AUTH` (base64), and offline | public: what systemd-boot enrolls into the firmware |
| `PK.key`, `KEK.key` | offline only | sign later changes to the enrolled keys (a new db, a revocation) |

`db.auth` and `KEK.auth` include Microsoft's certificates, so graphics cards and
other option ROMs signed by Microsoft keep working after enrollment.

## Creating them, once

On a trusted machine with Docker, `gh` logged in with admin rights on the
repository, and `openssl`:

```sh
scripts/uki-release-keys.sh generate ~/isotope-uki-keys
scripts/uki-release-keys.sh upload ~/isotope-uki-keys
gh variable set TRUSTED_BOOT_RELEASE --body true --repo kairos-community/isotope
```

Then copy `~/isotope-uki-keys` to two offline places (an encrypted USB stick
and a password manager vault, say) and delete it from the machine. The
certificates are valid for ten years (`DAYS` changes that); UEFI firmware does
not check their dates, but the release job warns 60 days before the db
certificate expires.

`DRY_RUN=1 scripts/uki-release-keys.sh upload DIR` shows what would be set
without setting anything.

## What CI does with them

The `build-uki` job in `.github/workflows/release.yml` runs only when
`TRUSTED_BOOT_RELEASE` is `true`. It writes the six secrets to `/dev/shm`
(umask 077, never the workspace), checks that `db.pem` belongs to `db.key`,
and signs the UKI ISO and upgrade image of every full line (sway, i3, cosmic,
niri) and the server. The release job then requires those five ISOs and
images; a missing one stops the release.

## Things this does not cover yet

- **Rotating the db key.** It needs a `db` update signed with `KEK.key`,
  installed on every machine, and a release signed with both keys while
  machines move over. Not automated.
- **A leaked db key.** Revoking it needs a `dbx` update signed with `KEK.key`.
  Until that exists, a leak cannot be undone on installed machines.
- **Your own keys.** A user who wants their own root of trust builds with
  `make uki-keys uki-iso` and must build every upgrade with the same keys:
  the project's `<flavor>-uki-<tag>` images do not boot on their machines.
