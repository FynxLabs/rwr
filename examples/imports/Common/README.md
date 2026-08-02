# Common — Shared Definitions

Shared blueprint definitions that other trees pull in with the `import` directive. This directory is not run directly — it holds the reusable pieces.

## Contents

- `packages/base-aur-arch.yaml` — a base set of AUR packages (`yay`, `paru`, `visual-studio-code-bin`, `google-chrome`, `slack-desktop`, `discord`) installed via `paru`

## How it is used

[`../Arch/packages/packages.yaml`](../Arch/packages/packages.yaml) imports this file:

```yaml
packages:
  - import: ../../Common/packages/base-aur-arch.yaml
```

RWR resolves the path relative to the importing blueprint and merges the imported entries with the local ones. See [`../README.md`](../README.md) for the full import feature description.
