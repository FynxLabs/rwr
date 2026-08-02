# Credentials in blueprints

RWR holds two credentials: your GitHub API token and your SSH private key. By
default, blueprints cannot read them.

## Why RWR holds back the credentials

Blueprints usually come from a git repository. Everything that a blueprint can
read, the author of that blueprint can read:

- A template reads from a blueprint file. RWR writes the result to a path that
  the same blueprint selects. A token in template scope can go to any path.
- A script gets the environment of RWR. Every script in the blueprint can read
  each variable in that environment.

This is safe when the blueprint is yours. This is not safe for a blueprint from a
different author. RWR cannot tell the difference between the two.

## How to permit a credential

Some blueprints need a credential. Examples are a blueprint that writes a
`.netrc` file, a blueprint that configures `gh`, and a script that calls the
GitHub API.

Give the name of each credential that the blueprints can read:

```yaml
blueprints:
  format: yaml
  location: "."

exposeCredentials:
  - gh_api_token
```

RWR shares only the credentials that you give. The name `gh_api_token` does not
give access to the SSH key.

These names are correct:

| Name | RWR also accepts |
|---|---|
| `gh_api_token` | `repository.gh_api_token` |
| `ssh_private_key` | `repository.ssh_private_key` |

When a credential is available, RWR warns at startup, so the exposure is
always visible.

## What a permitted credential gives you

In a template:

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

In a script, through the environment:

```bash
#!/usr/bin/env bash
gh auth login --with-token <<< "$RWR_VAR_REPOSITORY_GH_API_TOKEN"
```

## How to keep the risk small

- Give only the credential that the blueprint needs.
- Set `exposeCredentials` in the init file of the tree that needs it. Do not set
  it in an init file that other trees use.
- Let RWR do the work when it can. An `ssh_keys` blueprint sends the key to
  GitHub with the token. The blueprint does not read the token.

## Logs

RWR removes credential values from the logs. This applies to permitted
credentials also.

If you must see a value in the logs, use the `--show-secrets` flag. RWR gives a
warning while this flag is active.
