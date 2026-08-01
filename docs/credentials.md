# Credentials in blueprints

RWR holds two credentials: your GitHub API token and your SSH private key. By
default, blueprints cannot read either.

## Why they are withheld by default

Blueprints are usually cloned from a git repository, and everything a blueprint
can reach, whoever wrote that blueprint can reach:

- a **template** renders from a blueprint file and is written to a path the same
  blueprint chooses, so a token in template scope can be copied anywhere
- a **script** inherits RWR's environment, so anything exported is readable by
  every script the blueprint runs

That is fine when the blueprint is yours. It is not fine for a blueprint you
cloned from someone else, and there is no way for RWR to tell the difference.

## Opting in

Some blueprints legitimately need a credential — writing a `.netrc`, configuring
`gh`, calling the GitHub API from a script. Name the ones you want available:

```yaml
blueprints:
  format: yaml
  location: "."

exposeCredentials:
  - gh_api_token
```

Only what you name is shared. Opting into `gh_api_token` does not expose the SSH
key.

Accepted names:

| Name | Also accepted as |
|---|---|
| `gh_api_token` | `repository.gh_api_token` |
| `ssh_private_key` | `repository.ssh_private_key` |

RWR warns at startup when anything is exposed, so it is never silent.

## What opting in gives you

**In templates**, as before:

```yaml
templates:
  - name: netrc
    action: copy
    source: ./src/netrc.tmpl
    target: "{{ .User.home }}/.netrc"
```

```
machine github.com login {{ .User.username }} password {{ .Flags.ghAPIToken }}
```

**In scripts**, through the environment:

```bash
#!/usr/bin/env bash
gh auth login --with-token <<< "$RWR_VAR_REPOSITORY_GH_API_TOKEN"
```

## Keeping the blast radius small

- Name only the credential a blueprint actually needs.
- Set it in the init file of the tree that needs it, not in a shared one.
- Prefer having RWR do the work itself where it can — `ssh_keys` blueprints
  already upload to GitHub using the token without exposing it to anything.

## Logs

Credential values are redacted in logs regardless of this setting. Use
`--show-secrets` when you need to confirm RWR is reading the value you expect;
it warns while enabled.
