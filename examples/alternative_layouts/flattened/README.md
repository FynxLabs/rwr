# Flattened Layout

This directory shows one file per blueprint type, organized by format.

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

RWR finds and processes all files in the directory that have the specified format extension. The file names have no effect.

## What This Demonstrates

- **Blueprint discovery**: RWR scans for files with the correct format extension to find blueprints
- **Content-based detection**: File names have no effect. `packages.yaml`, `my-packages.yaml`, and `stuff.yaml` all work if they contain `packages:`
- **Flat layout**: No subdirectories are needed. Everything is at the root level
- **Clear separation**: Each blueprint type is in its own file for easy maintenance

## Benefits

✅ **Advantages:**

- **Simple layout** - there are no nested directories
- **Easy to navigate** - everything is at one level
- **Clear separation** - each blueprint type has its own file
- **Maintainable** - you can find and edit each blueprint type easily
- **Version control friendly** - changes are visible at the file level

✅ **Good for:**

- Small to medium projects
- Clear separation of concerns
- Teams that prefer organized but simple layouts
- When you want dedicated files per blueprint type

## Comparison with Other Layouts

| Aspect                   | Flattened | Minimal Files | Nested Layout    |
| ------------------------ | --------- | ------------- | ---------------- |
| Files per blueprint type | 1         | All in 1      | 1+ (in subdirs)  |
| Directory depth          | 1 level   | 1 level       | 3+ levels        |
| Separation of concerns   | High      | Low           | High             |
| Setup complexity         | Low       | Lowest        | High             |
| Scalability              | Medium    | Low           | High             |

## File Naming Flexibility

These file names are all equivalent:

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

This layout balances simplicity and organization. It is good for projects that need clear structure without complexity.
