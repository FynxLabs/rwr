# Arch - Machine Tree

The per-machine side of the import example: the tree a specific machine would actually run. It imports shared definitions from [`../Common/`](../Common/) and adds machine-specific entries on top.

## Contents

- `packages/packages.yaml` - imports the shared AUR base set from `Common/packages/base-aur-arch.yaml`, then adds Arch-specific packages (`nvm`, `gosec`, `protontricks`, and others) via `paru`

## How the import works

```yaml
packages:
  # Shared base packages
  - import: ../../Common/packages/base-aur-arch.yaml
  # Machine-specific additions
  - names:
      - nvm
      - gosec
    action: install
    package_manager: paru
```

RWR merges the imported entries with the local ones at processing time. See [`../README.md`](../README.md) for details on path resolution, circular-import detection, and supported blueprint types.
