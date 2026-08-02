# Minimal Files Layout

This directory shows the simplest way to organize RWR blueprints: 2 files per format.

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

- **Content-based detection**: RWR identifies blueprint types by their keys (for example `packages:` and `git:`), not filenames
- **Format flexibility**: The same content works in YAML, JSON, or TOML
- **Minimal complexity**: Good for personal setups, quick prototypes, and learning RWR
- **Single-file layout**: Everything you need is in one blueprint file

## When to Use This Layout

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

This layout shows that RWR does not require a complex directory structure. A simple, flat layout works.
