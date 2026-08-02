# Flattened Structure Approach

This directory demonstrates organizing blueprint files individually with clear separation of concerns, organized by format.

## Structure

```bash
flattened/
├── yaml/
│   ├── init.yaml
│   ├── packages.yaml
│   ├── git.yaml
│   ├── files.yaml
│   └── scripts.yaml
├── json/
│   ├── init.json
│   ├── packages.json
│   ├── git.json
│   ├── files.json
│   └── scripts.json
└── toml/
    ├── init.toml
    ├── packages.toml
    ├── git.toml
    ├── files.toml
    └── scripts.toml
```

## Key Concept

Each file contains **one blueprint type**, identified by its content key:

```yaml
# packages.yaml
packages:
  - name: git
    action: install

# git.yaml
git:
  - name: dotfiles
    action: clone

# files.yaml
files:
  - name: .bashrc
    action: create
```

## Usage

Choose your preferred format:

```bash
# YAML format
cd flattened/yaml
rwr run

# JSON format
cd flattened/json
rwr run

# TOML format
cd flattened/toml
rwr run
```

RWR will automatically discover and process all files with the specified format extension in the directory, regardless of their names.

## What This Demonstrates

- **Blueprint discovery**: RWR finds blueprints by scanning for files with the correct format extension
- **Content-based typing**: File names don't matter - `packages.yaml`, `my-packages.yaml`, or `stuff.yaml` all work if they contain `packages:`
- **Flat organization**: No subdirectories needed - everything at root level
- **Clear separation**: Each blueprint type in its own file for easy maintenance

## Benefits

✅ **Advantages:**

- **Simple structure** - no nested directories
- **Easy to navigate** - everything at one level
- **Clear separation** - each blueprint type has its own file
- **Maintainable** - easy to find and edit specific blueprint types
- **Version control friendly** - clear file-level changes

✅ **Good for:**

- Small to medium projects
- Clear separation of concerns
- Teams that prefer organized but simple structures
- When you want dedicated files per blueprint type

## Comparison with Other Approaches

| Aspect                   | Flattened | Minimal Files | Nested Structure |
| ------------------------ | --------- | ------------- | ---------------- |
| Files per blueprint type | 1         | All in 1      | 1+ (in subdirs)  |
| Directory depth          | 1 level   | 1 level       | 3+ levels        |
| Separation of concerns   | High      | Low           | High             |
| Setup complexity         | Low       | Lowest        | High             |
| Scalability              | Medium    | Low           | High             |

## File Naming Flexibility

These are all equivalent ways to organize the same functionality:

```bash
# Current approach
packages.yaml    # Contains packages:
git.yaml         # Contains git:
files.yaml       # Contains files:

# Alternative names (also valid)
my-packages.yaml # Contains packages:
repositories.yaml # Contains git:
dotfiles.yaml    # Contains files:

# Even this works
setup.yaml       # Contains packages:
repos.yaml       # Contains git:
configs.yaml     # Contains files:
```

**The key is the content, not the filename!**

This approach strikes a balance between simplicity and organization, making it perfect for projects that need clear structure without excessive complexity.
