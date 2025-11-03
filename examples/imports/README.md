# Blueprint Import Examples

This directory demonstrates the new **import** feature for RWR blueprints. The import directive allows you to reference and include blueprint definitions from other files, making it easy to share common configurations across multiple systems or projects.

## How Imports Work

Add an `import` field to any blueprint entry to include definitions from another file:

```yaml
packages:
  - import: ../../Common/packages/arch/base-aur.yaml
  - names:
      - custom-package
    action: install
```

### Key Features

- **Relative Paths**: Import paths are resolved relative to the blueprint directory
- **Circular Detection**: Automatically detects and skips circular imports
- **Merge Behavior**: Imported items are merged with local definitions
- **All Blueprint Types**: Works with packages, files, services, git repos, scripts, SSH keys, users, and more

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

### Shared Configuration (Common/packages/arch/base-aur.yaml)

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
  - import: ../../Common/packages/arch/base-aur.yaml

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

- **Packages**: Share package lists across configurations
- **Files**: Reuse file definitions and templates
- **Services**: Import common service configurations
- **Git**: Share repository lists
- **Scripts**: Reuse script definitions
- **Repositories**: Import package repository configurations
- **SSH Keys**: Share SSH key configurations
- **Users/Groups**: Import user and group definitions

## Best Practices

1. **Organize by Purpose**: Group shared configurations in a Common directory
2. **Use Clear Paths**: Make import paths descriptive and easy to understand
3. **Document Imports**: Add comments explaining what each import provides
4. **Avoid Deep Nesting**: Keep import chains shallow for maintainability
5. **Profile Filtering**: Imports respect profile filtering just like regular entries

## Error Handling

- **Missing Files**: Import fails if the referenced file doesn't exist
- **Circular Imports**: Automatically detected and skipped with a warning
- **Invalid Format**: Import fails if the file cannot be parsed
