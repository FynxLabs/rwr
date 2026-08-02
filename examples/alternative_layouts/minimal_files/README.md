# Minimal Files Approach

This directory demonstrates the **absolute simplest** way to organize RWR blueprints - just 2 files per format.

## Structure

```bash
minimal_files/
├── yaml/
│   ├── init.yaml
│   └── all_in_one.yaml
├── json/
│   ├── init.json
│   └── all_in_one.json
└── toml/
    ├── init.toml
    └── all_in_one.toml
```

## Key Concept

Each `all_in_one.*` file contains **multiple blueprint types** in a single file:

```yaml
packages:          # Package management
  - name: git
    action: install

git:              # Git repositories
  - name: dotfiles
    action: clone

files:            # File operations
  - name: .bashrc
    action: create

scripts:          # Script execution
  - name: setup-dev-env
    action: inline
```

## Usage Examples

Choose your preferred format and run:

```bash
# YAML format
cd yaml
rwr run

# JSON format
cd json
rwr run

# TOML format
cd toml
rwr run
```

## What This Demonstrates

- **Content-based identification**: RWR identifies blueprint types by their keys (`packages:`, `git:`, etc.), not filenames
- **Format flexibility**: Same functionality in YAML, JSON, or TOML
- **Minimal complexity**: Perfect for personal setups, quick prototypes, or learning RWR
- **Single-file organization**: Everything you need in just one blueprint file

## When to Use This Approach

✅ **Good for:**

- Personal configurations
- Quick setups and prototypes
- Learning RWR basics
- Simple projects with few blueprints
- When you want everything visible at once

❌ **Consider alternatives for:**

- Large, complex configurations
- Team projects with multiple contributors
- When you need clear separation of concerns
- Projects with many blueprint files

This approach proves that RWR doesn't require complex directory structures - it's perfectly happy with simple, flat organization.
