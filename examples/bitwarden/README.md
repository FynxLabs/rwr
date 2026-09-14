# Native Bitwarden credential setup

Install GPG as part of ordinary machine setup. Set the task `fingerprint` fields in credentials/credentials.yaml to your full signing-key fingerprint and adjust the trusted provider/item declarations.

```sh
rwr run all --except credentials --init-file examples/bitwarden/init.yaml
rwr run credentials --profile bitwarden --init-file examples/bitwarden/init.yaml
rwr run credentials --profile gpg-backup --init-file examples/bitwarden/init.yaml
```

Restore is separate from backup. The backup write profile must be explicitly selected. RWR installs Bitwarden if missing, handles login/MFA and unlock during explicit setup, and owns the native GPG flow. No Bash or jq workflow is used. A session is process-local; imported keys and Git signing configuration persist. Repeated restoration avoids the vault when the key is already present. This example leaves ownertrust unchanged.

Dry-run and validation do not contact a vault. See [credentials](../../docs/credentials.md) for task fields and security boundaries.
