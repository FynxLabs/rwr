#!/usr/bin/env bash
# gpg-restore: put a GPG key back onto this machine from Bitwarden.
#
# Downloads the private-key attachment, proves it is the right key and that it
# unlocks with the vault's passphrase, then imports it into the local keyring
# and points git signing at it. Safe to re-run: a machine that already holds
# the key is a no-op.
set -euo pipefail

# Environment over defaults: rwr passes the shell's environment through to
# every script, so `GPG_FINGERPRINT=abc rwr all --profile gpg-restore` works,
# and so does editing the defaults here.
item="${BW_GPG_ITEM:-gpg-signing}"
fingerprint="${GPG_FINGERPRINT:-}"

die() { echo "gpg-restore: $*" >&2; exit 1; }

command -v gpg >/dev/null 2>&1 || die "gpg is not installed"
command -v bw  >/dev/null 2>&1 || die "the Bitwarden CLI (bw) is not installed - https://bitwarden.com/help/cli/"
command -v jq  >/dev/null 2>&1 || die "jq is not installed (used to read vault item metadata)"

fingerprint=$(printf '%s' "$fingerprint" | tr -d ' ' | tr 'a-f' 'A-F')
if [ -z "$fingerprint" ]; then
  die "no fingerprint given - export GPG_FINGERPRINT or edit the default here"
fi
if [ "${#fingerprint}" -ne 40 ]; then
  die "'$fingerprint' is not a full 40-character fingerprint"
fi

# The no-op guard, and the reason this profile is safe to leave in a shared
# tree: a machine that already holds the key has nothing left to do.
# gpg exits 0 for a list that matched nothing, so the guard reads the
# listing, not the exit code.
if gpg --list-secret-keys --with-colons "$fingerprint" 2>/dev/null | grep -q '^sec:'; then
  echo "gpg-restore: key $fingerprint is already in the keyring - nothing to do"
  exit 0
fi
if [ -z "${BW_SESSION:-}" ]; then
  die "BW_SESSION is not set - run 'bw unlock' and export BW_SESSION first"
fi

itemid=$(bw get item "$item" | jq -r '.id')
if [ -z "$itemid" ] || [ "$itemid" = "null" ]; then
  die "no vault item named '$item' - the backup lives there; run gpg-backup on the source machine first"
fi

workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT
chmod 700 "$workdir"

bw get attachment private.asc --itemid "$itemid" > "$workdir/private.asc" \
  || die "could not download private.asc from '$item' - is the backup there?"

# Import into a throwaway keyring first. The attachment is expected to be the
# right key, but importing key material is not something to do on trust: what
# comes back must be exactly the fingerprint this tree was configured for.
# The sandbox is passed to every gpg call as a per-command environment, never
# as an exported variable - an exported one would leak into every later gpg
# call, and a merely assigned one would silently NOT reach the children at
# all when the operator had no GNUPGHOME set. The real import below targets
# $real_gnupghome the same way, so the operator's keyring choice is honored.
real_gnupghome="${GNUPGHOME:-}"
sandbox="$workdir/gnupg"
mkdir -m 700 -p "$sandbox"
GNUPGHOME="$sandbox" gpg --batch --quiet --import "$workdir/private.asc"
# Every key the attachment holds lands in the sandbox, so compare the whole
# set, not just the first fingerprint: a tampered or wrong attachment must be
# refused, not partially imported into the real keyring afterwards.
imported=$(GNUPGHOME="$sandbox" gpg --batch --list-secret-keys --with-colons | awk -F: '/^fpr:/ {print $10}' | sort -u)
if [ "$imported" != "$fingerprint" ]; then
  die "the vault backup does not hold exactly $fingerprint (found: $(printf '%s' "$imported" | tr '\n' ' ')) - not importing"
fi

# Same proof as on backup day: the passphrase in the vault must unlock the
# key, here and now, before it becomes this machine's signing key.
if [ -n "${RWR_CRED_GPG_PASSPHRASE:-}" ]; then
  check_sign() {
    printf 'rwr unlock check\n' | GNUPGHOME="$sandbox" gpg --batch --quiet --pinentry-mode loopback \
      --local-user "$fingerprint" --clearsign --output /dev/null \
      --passphrase-fd 3 3<<<"$1" 2>/dev/null
  }
  if check_sign "$RWR_CRED_GPG_PASSPHRASE"; then
    echo "gpg-restore: passphrase verified against the key"
  elif check_sign ""; then
    echo "gpg-restore: warning: the key has no passphrase - the vault is its only protection" >&2
  else
    die "the vault passphrase does not unlock this key - it may be stale; fix it in the vault first"
  fi
else
  echo "gpg-restore: note: gpg_passphrase not exposed; skipping the unlock check" >&2
fi

GNUPGHOME="$real_gnupghome" gpg --batch --quiet --import "$workdir/private.asc"
printf '%s:6:\n' "$fingerprint" | GNUPGHOME="$real_gnupghome" gpg --batch --import-ownertrust

if command -v git >/dev/null 2>&1; then
  git config --global user.signingkey "$fingerprint"
  git config --global commit.gpgsign true
  echo "gpg-restore: git signing configured for $fingerprint"
fi

echo "gpg-restore: restored $fingerprint from '$item'"
