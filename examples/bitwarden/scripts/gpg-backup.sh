#!/usr/bin/env bash
# gpg-backup: copy a GPG key into Bitwarden.
#
# Uploads the public key, the private key, and the revocation certificate
# (when one exists) as attachments of a vault item, replacing the previous
# copies so the item holds exactly one current backup of each. The private-key
# export keeps the key's passphrase protection - it is not readable without
# the passphrase, and the vault adds its own encryption on top.
#
# rwr exposes the key's passphrase as RWR_CRED_GPG_PASSPHRASE (see init.yaml).
# When set, the script proves the exported key actually unlocks with it before
# anything is uploaded, so a drifted passphrase is caught on backup day, not
# restore day.
set -euo pipefail

# Environment over defaults: rwr passes the shell's environment through to
# every script, so `GPG_FINGERPRINT=abc rwr all --profile gpg-backup` works,
# and so does editing the defaults here.
item="${BW_GPG_ITEM:-gpg-signing}"
fingerprint="${GPG_FINGERPRINT:-}"

die() { echo "gpg-backup: $*" >&2; exit 1; }

# Guards, cheapest first. Each one says what is missing and what to do - a
# script that half-runs against a locked vault is worse than one that does
# not start.
command -v gpg >/dev/null 2>&1 || die "gpg is not installed"
command -v bw  >/dev/null 2>&1 || die "the Bitwarden CLI (bw) is not installed - https://bitwarden.com/help/cli/"
command -v jq  >/dev/null 2>&1 || die "jq is not installed (used to read vault item metadata)"

# The no-op guard: nothing in the keyring means nothing to back up. This is
# what keeps the script harmless when it runs on a machine that was never
# meant to hold the key.
fingerprint=$(printf '%s' "$fingerprint" | tr -d ' ' | tr 'a-f' 'A-F')
if [ -z "$fingerprint" ]; then
  die "no fingerprint given - export GPG_FINGERPRINT or edit the default here"
fi
if [ "${#fingerprint}" -ne 40 ]; then
  die "'$fingerprint' is not a full 40-character fingerprint"
fi
# gpg exits 0 for a list that matched nothing, so the guard reads the
# listing, not the exit code.
if ! gpg --list-secret-keys --with-colons "$fingerprint" 2>/dev/null | grep -q '^sec:'; then
  echo "gpg-backup: key $fingerprint is not in the local keyring - nothing to back up"
  exit 0
fi
# A set-but-stale BW_SESSION passes an emptiness check and then dies deep in
# the run with a misleading error, so ask the CLI what its session really is.
# bw status is local and never prompts.
session_status=$(bw status 2>/dev/null | jq -r '.status' 2>/dev/null || echo unusable)
case "$session_status" in
  unlocked) ;;
  locked) die "the vault is locked - run 'bw unlock' and export BW_SESSION in this shell" ;;
  unauthenticated) die "not logged in to bw - run 'bw login' first" ;;
  *) die "could not read bw status (got '${session_status:-nothing}') - is the bw CLI working?" ;;
esac

# Key material lives only in this directory, which is removed on every exit
# path, normal or not.
workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT
chmod 700 "$workdir"

gpg --armor --export "$fingerprint" > "$workdir/public.asc"

# Since gpg 2.1 the secret key lives in the agent, and exporting a
# passphrase-protected key needs that passphrase, even in batch mode. Try the
# exposed credential first; an empty-passphrase export succeeding means the
# key has no protection at all, which is worth saying out loud.
export_secret() { # $1 = passphrase, possibly empty
  if [ -n "$1" ]; then
    gpg --batch --quiet --pinentry-mode loopback --passphrase-fd 3 3<<<"$1" \
      --armor --export-secret-keys "$fingerprint" 2>"$workdir/export.err"
  else
    gpg --batch --quiet --armor --export-secret-keys "$fingerprint" 2>"$workdir/export.err"
  fi
}
key_is_unprotected=0
if ! export_secret "${RWR_CRED_GPG_PASSPHRASE:-}" > "$workdir/private.asc" || [ ! -s "$workdir/private.asc" ]; then
  if ! export_secret "" > "$workdir/private.asc" || [ ! -s "$workdir/private.asc" ]; then
    if [ -z "${RWR_CRED_GPG_PASSPHRASE:-}" ]; then
      die "the key is passphrase-protected but gpg_passphrase is not exposed - expose the credential so the backup can be proven usable"
    fi
    die "the exposed passphrase does not unlock the key: $(tail -n 1 "$workdir/export.err")"
  fi
  key_is_unprotected=1
fi

gnupg_home="${GNUPGHOME:-$HOME/.gnupg}"
revocation="$gnupg_home/openpgp-revocs.d/${fingerprint}.rev"
if [ -f "$revocation" ]; then
  cp "$revocation" "$workdir/revocation.rev"
fi

# Prove the artifact, not just the keyring: import the exported file into a
# throwaway keyring and sign with it. This is the exact operation a restore
# will perform, run before anything is uploaded. The sandbox is passed as a
# per-command environment, never an exported variable - an exported one would
# leak into every later gpg call, and a merely assigned one would silently
# NOT reach the children at all when the operator had no GNUPGHOME set.
sandbox="$workdir/gnupg"
mkdir -m 700 -p "$sandbox"
GNUPGHOME="$sandbox" gpg --batch --quiet --import "$workdir/private.asc"
# The payload rides stdin; the passphrase candidate rides fd 3, so one helper
# serves both the vault passphrase and the empty-passphrase probe.
check_sign() {
  printf 'rwr unlock check\n' | GNUPGHOME="$sandbox" gpg --batch --quiet --pinentry-mode loopback \
    --local-user "$fingerprint" --clearsign --output /dev/null \
    --passphrase-fd 3 3<<<"$1" 2>/dev/null
}
if [ "$key_is_unprotected" -eq 1 ]; then
  echo "gpg-backup: warning: the key has no passphrase - the vault is its only protection" >&2
elif check_sign "${RWR_CRED_GPG_PASSPHRASE:-}"; then
  echo "gpg-backup: passphrase verified against the exported key"
else
  die "the exported key does not unlock with the exposed passphrase - fix it in the vault before backing up"
fi

itemid=$(bw get item "$item" | jq -r '.id')
if [ -z "$itemid" ] || [ "$itemid" = "null" ]; then
  die "no vault item named '$item' - create one (a Secure Note is fine)"
fi

# Each attachment name uploads once, so this snapshot of the item cannot go
# stale between the old-id lookups and the deletes below.
itemjson=$(bw get item "$item")

upload() {
  name=$1
  path=$2
  old=$(jq -r --arg n "$name" '.attachments[]? | select(.fileName == $n) | .id' <<<"$itemjson")
  bw create attachment --file "$path" --itemid "$itemid" >/dev/null
  # A previous run that died between create and delete leaves duplicates; a
  # multi-line $old used to fail the delete and grow the pile forever. Remove
  # every prior copy - best effort, since the new upload is already in place.
  while IFS= read -r id; do
    [ -n "$id" ] && [ "$id" != "null" ] || continue
    bw delete attachment "$id" --itemid "$itemid" >/dev/null \
      || echo "gpg-backup: warning: could not delete previous attachment $id of '$name'" >&2
  done <<<"$old"
}

upload public.asc "$workdir/public.asc"
upload private.asc "$workdir/private.asc"
if [ -f "$workdir/revocation.rev" ]; then
  upload revocation.rev "$workdir/revocation.rev"
fi

# The round trip is the proof: whatever the vault now holds must byte-match
# the export it came from. Two traps: the attachment's id is not knowable
# before the upload (the pre-upload snapshot cannot name it), and streaming
# content to stdout needs --raw on current CLIs - downloading by id with
# --output avoids both.
attid=$(bw get item "$item" | jq -r --arg n "private.asc" '.attachments[]? | select(.fileName == $n) | .id')
if [ -z "$attid" ] || [ "$attid" = "null" ]; then
  die "private.asc did not land on '$item' - the upload step lied"
fi
bw get attachment "$attid" --itemid "$itemid" --output "$workdir/verify.asc"
if ! cmp -s "$workdir/private.asc" "$workdir/verify.asc"; then
  die "the copy downloaded back from the vault differs from the upload - investigate before relying on this backup"
fi

echo "gpg-backup: backed up $fingerprint to '$item' (public, private$([ -f "$workdir/revocation.rev" ] && echo ', revocation'), round trip verified)"
