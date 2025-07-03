# Alternative Blueprint Layouts

This directory demonstrates simplified approaches to organizing RWR blueprints, proving that the complex nested directory structure is **not required**.

## Directory Overview

### [`minimal_files/`](./minimal_files/)

The absolute simplest approach - just 2 files per format, organized by format:

- `yaml/` - `init.yaml` + `all_in_one.yaml`
- `json/` - `init.json` + `all_in_one.json`
- `toml/` - `init.toml` + `all_in_one.toml`

### [`flattened/`](./flattened/)

Individual blueprint files with clear separation, organized by format:

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

### 3. Flexible File Organization

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

### Run with flattened structure

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

## Benefits of Simplified Structures

- **Faster setup** - fewer files to create
- **Easier maintenance** - less complex file organization
- **Better for small projects** - no over-engineering
- **Clearer dependencies** - everything visible in fewer files
- **Reduced cognitive load** - simpler mental model

## When to Use Each Approach

| Approach             | Best For                                              |
| -------------------- | ----------------------------------------------------- |
| **minimal_files**    | Quick setups, personal configs, learning RWR          |
| **flattened**        | Small-medium projects, clear separation of concerns   |
| **nested structure** | Large projects, multiple environments, shared configs |

The nested structure in the main examples is great for organization and learning, but these simplified approaches prove that RWR is flexible enough to work however you prefer to organize your files.
