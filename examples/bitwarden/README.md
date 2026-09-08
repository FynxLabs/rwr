# Bitwarden-backed GPG key sync

A small tree with two scripts that keep a GPG key in your Bitwarden vault and
put it back on any machine:

- `gpg-backup` exports the key (public, private, revocation certificate) and
  uploads it as attachments of a vault item, replacing the previous copies.
- `gpg-restore` downloads the private key on a new machine, verifies it is the
  key this tree expects and that the vault's passphrase unlocks it, then
  imports it and points git signing at it.

Neither script maintains a local file of exported secrets. The passphrase is
resolved by rwr at run time through the [`bw:` credential source](../../docs/credentials.md)
and handed to the scripts as `RWR_CRED_GPG_PASSPHRASE`; the key material exists
on disk only inside a temp directory that is removed when the script exits.

## Prerequisites

| Tool | Used for | Notes |
|---|---|---|
| `bw` | everything vault-related | A recent CLI; attachment *upload* needs one that has `bw create attachment`, and the scripts download attachments by id with `--output` (streaming to stdout needs `--raw`, and a filename argument is only a search, not a lookup) - both scripts fail loudly, not silently, when something is off |
| `gpg` | export, import, unlock checks | |
| `jq` | reading item metadata in the scripts | |

Attachment uploads require a paid Bitwarden plan (Premium or an organization).
The `bw:` credential source itself does not - item fields are free - so the
passphrase resolution works on any account. If attachments are not an option,
a Secure Note whose notes field holds the armored private key is the manual
fallback: paste it into a file on the new machine and run `gpg --import`
yourself; the rest of this tree's checks still apply.

## One-time setup

1. RWR offers to install the CLI if it is missing, with no package-manager
   prerequisite. Log in with `bw login`, then `bw unlock` and export
   `BW_SESSION` in the shell you run rwr from. If RWR installed it, use the
   executable path it prints for these shell commands; RWR itself finds it
   automatically.
2. Create a vault item named `gpg-signing` **as a Login item** - a Secure Note
   has no password field, so the source below could never read from it - and
   put the key's **passphrase in the item's password field**. Username and
   URI can stay empty; only the password field matters. That is what the
   `bw:gpg-signing/password` credential source reads.
3. If the key does not exist yet, create it: `gpg --full-generate-key`, with a
   passphrase - the passphrase is what makes the vault backup safe to keep.

## Run it

```sh
# Any machine that should hold the key:
GPG_FINGERPRINT=ABCDEF01... rwr all --profile gpg-restore

# After creating the key, changing the passphrase, or adding a subkey:
GPG_FINGERPRINT=ABCDEF01... rwr all --profile gpg-backup
```

Both scripts read `GPG_FINGERPRINT` and `BW_GPG_ITEM` from the environment -
rwr passes the shell's environment through to every script - so override them
per run, or edit the defaults at the top of the script files.

One thing to know about profiles in rwr: naming none of them runs *everything*.
A bare `rwr all` on any machine therefore runs both scripts - and that is safe
here only because the scripts no-op when their subject is absent (backup
without the key, restore with it already restored, both exit 0 with a
message). Profile the work *and* guard the scripts; either alone is not the
whole fix.

## What the scripts guarantee

- **Optional vault setup.** A missing CLI, locked vault, or logged-out session
  skips the GPG script. Other invalid inputs fail before key material moves.
- **No-op when the subject is absent.** Backup on a machine without the key
  prints "nothing to back up" and exits 0; restore on a machine that already
  holds the key exits 0. Neither can be turned into an accident by running the
  wrong profile on the wrong machine.
- **The passphrase is proven, not assumed.** Backup imports the export into a
  throwaway keyring and signs with it before uploading; restore does the same
  before importing for real. A stale passphrase is caught on the day it
  drifted, with a message that says so.
- **The backup is what it claims.** The uploaded private key is downloaded
  back and byte-compared before backup reports success; restore checks the
  imported fingerprint against the configured one before touching the real
  keyring.
- **One current backup.** Each upload replaces the previous attachment of the
  same name, so the vault item holds exactly one copy per file.

## Files

| File | Purpose |
|---|---|
| `init.yaml` | Declares the `bw:` credential and exposes it to scripts |
| `scripts/scripts.yaml` | The two profile-gated entries |
| `scripts/gpg-backup.sh` | Export, verify passphrase, upload, verify round trip |
| `scripts/gpg-restore.sh` | Download, verify fingerprint and passphrase, import, configure git |
