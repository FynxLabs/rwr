# Blueprint Import Examples

This directory demonstrates the **import** feature for RWR blueprints. With the import directive, a blueprint includes definitions from another file. You can share common definitions across systems and projects.

## How Imports Work

Add an `import` field to any blueprint entry to include definitions from another file:

```yaml
packages:
  - import: ../../Common/packages/base-aur-arch.yaml
  - names:
      - custom-package
    action: install
```

### Key Features

- **Relative Paths**: RWR resolves import paths relative to the blueprint directory
- **Circular Detection**: RWR detects circular imports and skips them
- **Merge Behavior**: RWR merges imported items with local definitions
- **All Blueprint Types**: The import directive works with packages, files, services, git repos, scripts, SSH keys, users, and more

## Example Structure

```text
examples/imports/
├── Common/
│   └── packages/
│       └── arch/
│           └── base-aur.yaml          # Shared AUR packages
└── Arch/
    └── packages/
        └── packages.yaml               # Imports common + adds specific packages
```

## Usage Example

### Shared Configuration (Common/packages/base-aur-arch.yaml)

```yaml
packages:
  - names:
      - yay
      - paru
      - visual-studio-code-bin
      - google-chrome
    action: install
    package_manager: paru
```

### System-Specific Configuration (Arch/packages/packages.yaml)

```yaml
packages:
  # Import shared packages
  - import: ../../Common/packages/base-aur-arch.yaml

  # Add system-specific packages
  - names:
      - nvm
      - gosec
      - protontricks
    action: install
    package_manager: paru
```

## Supported Blueprint Types

The import directive works with all blueprint types:

- **Packages**: You can share package definitions across systems
- **Files**: You can reuse file definitions and templates
- **Services**: You can import common service definitions
- **Git**: You can share repository definitions
- **Scripts**: You can reuse script definitions
- **Repositories**: You can import package repository definitions
- **SSH Keys**: You can share SSH key definitions
- **Users/Groups**: You can import user and group definitions

## Best Practices

1. **Organize by Purpose**: Group shared definitions in a Common directory
2. **Use Clear Paths**: Make import paths descriptive and easy to understand
3. **Document Imports**: Add comments that explain what each import provides
4. **Avoid Deep Nesting**: Keep import chains shallow for maintainability
5. **Profile Filtering**: Imports respect profile filtering, the same as regular entries

## Error Handling

- **Missing Files**: If the referenced file does not exist, the import fails
- **Circular Imports**: RWR detects them, skips them, and logs a warning
- **Invalid Format**: If RWR cannot parse the file, the import fails
