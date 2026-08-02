# Alternative Blueprint Layouts

This directory shows simplified layouts for RWR blueprints. RWR does **not require** the nested directory structure.

## Directory Overview

### [`minimal_files/`](./minimal_files/)

The simplest layout is 2 files per format, organized by format:

- `yaml/` - `init.yaml` + `all_in_one.yaml`
- `json/` - `init.json` + `all_in_one.json`
- `toml/` - `init.toml` + `all_in_one.toml`

### [`flattened/`](./flattened/)

Each blueprint type is in its own file, organized by format:

- `yaml/` - Individual YAML files per blueprint type
- `json/` - Individual JSON files per blueprint type
- `toml/` - Individual TOML files per blueprint type

Each format directory contains: `init`, `packages`, `git`, `files`, `scripts`

## Key Principles Demonstrated

### 1. Content-Based Blueprint Detection

RWR identifies blueprints by their content keys, not filenames:

```yaml
packages:          # ← This key identifies it as a packages blueprint
  - name: git
    action: install

git:              # ← This key identifies it as a git blueprint
  - name: dotfiles
    action: clone
```

### 2. Multiple Blueprint Types Per File

You can combine multiple blueprint types in a single file:

```yaml
# Single file with multiple blueprint types
packages:
  - name: git
    action: install

files:
  - name: .bashrc
    action: create
    content: "alias ll='ls -la'"

git:
  - name: dotfiles
    action: clone
    url: "https://github.com/user/dotfiles.git"
```

### 3. Flexible File Layout

These are all equivalent and valid:

- `blueprints/packages/common.yaml`
- `blueprints/packages.yaml`
- `blueprints/my-setup.yaml` (containing `packages:` key)
- `blueprints/everything.yaml` (containing all blueprint types)

## Usage Examples

### Run with minimal_files

```bash
# Choose your preferred format
cd minimal_files/yaml
rwr run

# OR
cd minimal_files/json
rwr run

# OR
cd minimal_files/toml
rwr run
```

### Run with the flattened layout

```bash
# Choose your preferred format
cd flattened/yaml && rwr run
cd flattened/json && rwr run
cd flattened/toml && rwr run
```

### Run with different formats

```bash
# Each format in its own directory
cd minimal_files/yaml && rwr run   # Will process all_in_one.yaml
cd minimal_files/json && rwr run   # Will process all_in_one.json
cd minimal_files/toml && rwr run   # Will process all_in_one.toml
```

## Benefits of Simplified Layouts

- **Faster setup** - you create fewer files
- **Easier maintenance** - the file layout is less complex
- **Better for small projects** - a small project stays simple
- **Clearer dependencies** - everything is visible in fewer files
- **Reduced cognitive load** - the mental model is simpler

## When to Use Each Layout

| Layout               | Best For                                              |
| -------------------- | ----------------------------------------------------- |
| **minimal_files**    | Quick setups, personal configs, learning RWR          |
| **flattened**        | Small-medium projects, clear separation of concerns   |
| **nested layout**    | Large projects, multiple environments, shared configs |

The nested layout in the main examples helps with organization and learning. These simplified layouts show that RWR accepts any file layout.
