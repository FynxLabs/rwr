# Profile System

The Profile System is a powerful feature in RWR that allows you to organize and selectively install packages and configurations based on different use cases, environments, or contexts. This page provides a comprehensive guide to understanding and using profiles effectively.

## Introduction

The RWR Profile System enables you to create flexible, context-aware configurations that can be selectively applied based on your current needs. Whether you're setting up a work environment, gaming setup, or development workstation, profiles let you organize your blueprints for maximum efficiency and reusability.

### Key Benefits

- **Selective Installation**: Install only what you need for specific contexts
- **Configuration Reuse**: Share common base configurations across different setups
- **Environment Management**: Easily switch between work, personal, and gaming configurations
- **Backward Compatibility**: Existing configurations work without modification

## Core Concepts

### The Additive Model

The RWR Profile System uses an **additive model** rather than an exclusive one:

```text
Items with no profiles field    →  Always installed (the "base" concept)
+ Profile Items                 →  Installed when profile selected
+ Additional Profile Items      →  Installed when multiple profiles selected
```

> [!IMPORTANT]
> The filter only runs when you name at least one profile. `rwr all` with no
> `--profile` installs **everything**, profile items included. Naming a profile
> is what narrows the run down to the base items plus that profile.

### Base Items vs Profile Items

#### Base Items (No Profiles Field)

Items without a `profiles` field are considered "base" items and are **always installed**, regardless of which profiles are active.

```yaml
# Base items - always installed
packages:
  - name: vim
    action: install
  - name: git
    action: install
  - name: curl
    action: install
```

#### Profile Items

Items with a `profiles` field are **conditionally installed** when their profiles are active.

```yaml
# Profile items - conditionally installed
packages:
  - name: docker
    profiles: ["work"]
    action: install
  - name: steam
    profiles: ["gaming"]
    action: install
```

### User-Defined Profile Names

**Profile names are completely user-defined** - there are no predefined or required profile names. You have complete freedom to name profiles whatever makes sense for your setup.

#### Popular Profile Naming Strategies

##### Role-Based Profiles

```yaml
profiles: ["work", "personal", "gaming"]
```

##### Environment-Based Profiles

```yaml
profiles: ["desktop", "laptop", "server"]
```

##### Intensity-Based Profiles

```yaml
profiles: ["minimal", "standard", "full"]
```

##### Context-Based Profiles

```yaml
profiles: ["home", "office", "travel"]
```

### Special Keywords

Only **one** keyword is reserved:

- `all` - Installs everything regardless of profiles

**Important**: "base" is NOT a profile name - it's just the conceptual term for items without a `profiles` field.

## CLI Usage

The command that applies blueprints is `rwr all` (everything) or
`rwr run <processor>` (one processor). Bare `rwr` prints the help text and
applies nothing, so `rwr --profile work` does nothing at all.

### Basic Profile Commands

#### Install Everything

```bash
rwr all
```

With no profile named, no filtering happens: every item applies, whether it
carries a `profiles` field or not.

#### Install Base + Specific Profile

```bash
rwr all --profile work
rwr all -p work
```

Installs base items + items with the "work" profile.

#### Install Multiple Profiles

```bash
rwr all --profile work,gaming
rwr all -p work -p gaming
```

Installs base items + work profile items + gaming profile items.

#### Install Everything Explicitly

```bash
rwr all --profile all
```

`all` is the reserved profile name; it matches every item regardless of profiles.

### Profile Discovery

#### List Available Profiles

```bash
rwr profiles
```

Reads your blueprint tree and lists the profile names it declares, with the
number of items carrying each. See [Profile CLI Commands](cli/profiles.md).

### Run Specific Processors with Profiles

```bash
rwr run packages --profile work
rwr run services --profile gaming
rwr run files --profile work,dev
```

## Configuration Examples

The `profiles` field is available on packages, repositories, files, templates,
directories, services, users, groups, git repositories, scripts and SSH keys.

> [!IMPORTANT]
> **Fonts and configuration blueprints do not support `profiles`.** Blueprints
> decode strictly, so a `profiles` key in a `fonts/` or `configuration/`
> blueprint is an error:
>
> ```text
> field profiles not found in type types.Font
> ```

Here are examples for the types that do support it:

### Packages Blueprint

```yaml
packages:
  # Base packages - always installed
  - names:
      # System utilities and base development tools
      - base-devel
      - git
      - tree
      - unzip
      - zip
      - rsync
      - cmake
      - neovim
      - jq
    action: install
    package_manager: pacman

  # Work profile packages
  - names:
      - docker
      - docker-compose
      - kubectl
      - terraform
      - helm
    profiles: ["work"]
    action: install
    package_manager: pacman

  # Gaming profile packages
  - names:
      - steam
      - discord
      - obs-studio
    profiles: ["gaming"]
    action: install
    package_manager: pacman

  # Development profile packages
  - names:
      - code
      - nodejs
      - npm
      - python3
      - go
    profiles: ["dev"]
    action: install
    package_manager: pacman
```

### Services Blueprint

```yaml
services:
  # Base service - always enabled
  - name: sshd
    action: enable

  # Work profile services
  - name: docker
    profiles: ["work"]
    action: enable

  # Development profile services
  - name: postgresql
    profiles: ["dev"]
    action: enable

  # Multi-profile service
  - name: nginx
    profiles: ["work", "dev"]
    action: enable
```

### Files Blueprint

```yaml
files:
  # Base configuration files
  - name: bashrc
    source: templates/bashrc.j2
    target: ~/.bashrc

  # Work-specific configurations
  - name: work-ssh-config
    profiles: ["work"]
    source: configs/ssh/work_config
    target: ~/.ssh/config

  # Development configurations
  - name: dev-gitconfig
    profiles: ["dev"]
    source: configs/git/dev_gitconfig
    target: ~/.gitconfig
```

### Users & Groups Blueprint

```yaml
users:
  # Development user - only created with dev profile
  - name: developer
    profiles: ["dev"]
    action: create
    groups: ["docker", "sudo"]
    shell: /bin/zsh

groups:
  # Docker group for containerization profiles
  - name: docker
    profiles: ["work", "dev"]
    action: create
```

### Scripts Blueprint

```yaml
scripts:
  # Base system setup
  - name: system-update
    content: |
      #!/bin/bash
      sudo apt update && sudo apt upgrade -y
    action: run

  # Work environment setup
  - name: work-setup
    profiles: ["work"]
    source: scripts/work-environment.sh
    action: run
```

### SSH Keys Blueprint

```yaml
ssh_keys:
  # Personal SSH key
  - name: personal-key
    profiles: ["personal"]
    source: ~/.ssh/personal_id_rsa.pub
    target: ~/.ssh/authorized_keys

  # Work SSH key
  - name: work-key
    profiles: ["work"]
    source: ~/.ssh/work_id_rsa.pub
    target: ~/.ssh/authorized_keys
```

### Git Repositories Blueprint

```yaml
git:
  # Base development tools
  - name: dotfiles
    url: https://github.com/user/dotfiles.git
    target: ~/.dotfiles

  # Work repositories
  - name: work-configs
    profiles: ["work"]
    url: https://github.com/company/work-configs.git
    target: ~/work/configs

  # Gaming configurations
  - name: gaming-configs
    profiles: ["gaming"]
    url: https://github.com/user/gaming-configs.git
    target: ~/.config/gaming
```

## Multi-Format Examples

RWR supports YAML, JSON, TOML, and CUE formats for all configurations. Here are the same profile examples in different formats:

### YAML Format

```yaml
packages:
  - name: vim
    action: install
  - name: docker
    profiles: ["work"]
    action: install
  - name: steam
    profiles: ["gaming"]
    action: install
```

### JSON Format

```json
{
  "packages": [
    {
      "name": "vim",
      "action": "install"
    },
    {
      "name": "docker",
      "profiles": ["work"],
      "action": "install"
    },
    {
      "name": "steam",
      "profiles": ["gaming"],
      "action": "install"
    }
  ]
}
```

### TOML Format

```toml
[[packages]]
name = "vim"
action = "install"

[[packages]]
name = "docker"
profiles = ["work"]
action = "install"

[[packages]]
name = "steam"
profiles = ["gaming"]
action = "install"
```

## Real-World Scenarios

### Scenario 1: Developer Workstation Setup

```yaml
# Base tools everyone needs
packages:
  - name: git
    action: install
  - name: curl
    action: install
  - name: vim
    action: install

# Frontend development
packages:
  - name: nodejs
    profiles: ["frontend"]
    action: install
  - name: npm
    profiles: ["frontend"]
    action: install

# Backend development
packages:
  - name: docker
    profiles: ["backend"]
    action: install
  - name: postgresql
    profiles: ["backend"]
    action: install

# Mobile development
packages:
  - name: android-studio
    profiles: ["mobile"]
    action: install
```

Usage examples:

```bash
# Frontend developer setup
rwr all --profile frontend

# Full-stack developer setup
rwr all --profile frontend,backend

# Mobile developer setup
rwr all --profile mobile

# Complete development environment
rwr all --profile all
```

### Scenario 2: Multi-Environment Management

```yaml
# Common tools for all environments
packages:
  - name: htop
    action: install
  - name: neofetch
    action: install

# Desktop-specific packages
packages:
  - name: firefox
    profiles: ["desktop"]
    action: install
  - name: gimp
    profiles: ["desktop"]
    action: install

# Laptop-specific packages
packages:
  - name: powertop
    profiles: ["laptop"]
    action: install
  - name: tlp
    profiles: ["laptop"]
    action: install

# Server-specific packages
packages:
  - name: nginx
    profiles: ["server"]
    action: install
  - name: fail2ban
    profiles: ["server"]
    action: install
```

### Scenario 3: Gaming & Productivity Balance

```yaml
# Essential productivity tools
packages:
  - name: firefox
    action: install
  - name: libreoffice
    action: install

# Work productivity
packages:
  - name: slack
    profiles: ["work"]
    action: install
  - name: zoom
    profiles: ["work"]
    action: install

# Gaming setup
packages:
  - name: steam
    profiles: ["gaming"]
    action: install
  - name: discord
    profiles: ["gaming"]
    action: install
  - name: obs-studio
    profiles: ["gaming", "streaming"]
    action: install

# Content creation (overlaps with gaming)
packages:
  - name: kdenlive
    profiles: ["streaming", "content"]
    action: install
```

## Advanced Patterns

### Multi-Profile Items

Items can belong to multiple profiles, making them flexible for overlapping use cases:

```yaml
packages:
  # Installed with either work OR dev profile
  - name: tmux
    profiles: ["work", "dev"]
    action: install

  # Installed with gaming OR streaming profile
  - name: obs-studio
    profiles: ["gaming", "streaming"]
    action: install

  # Installed with any of multiple profiles
  - name: python3
    profiles: ["dev", "work", "data-science", "automation"]
    action: install
```

### Profile Hierarchies (Conceptual)

While RWR doesn't have built-in profile inheritance, you can achieve similar results with thoughtful profile design:

```yaml
# Base development tools
packages:
  - name: git
    profiles: ["dev-base"]
    action: install
  - name: vim
    profiles: ["dev-base"]
    action: install

# Frontend includes dev-base
packages:
  - name: nodejs
    profiles: ["frontend", "dev-base"]
    action: install

# Backend includes dev-base
packages:
  - name: docker
    profiles: ["backend", "dev-base"]
    action: install
```

Usage:

```bash
# Frontend development (includes base tools)
rwr all --profile frontend,dev-base

# Backend development (includes base tools)
rwr all --profile backend,dev-base
```

## Best Practices

### Profile Naming

1. **Use Descriptive Names**: Choose names that clearly indicate purpose
   - Good: `work`, `gaming`, `development`
   - Avoid: `p1`, `setup`, `misc`

2. **Be Consistent**: Use consistent naming patterns across your configuration
   - Role-based: `work`, `personal`, `gaming`
   - Environment-based: `desktop`, `laptop`, `server`

3. **Avoid Conflicts**: Don't use reserved words or system terms
   - Avoid: `all`, `base`, `default`

### Profile Organization

1. **Start with Base Items**: Identify truly universal packages/configs
2. **Group Related Items**: Keep related packages in the same profile
3. **Use Multi-Profile Items**: For packages that serve multiple purposes
4. **Test Profile Combinations**: Verify that profile combinations work correctly

## Troubleshooting

### Common Issues

#### A Misspelled Profile Is Silent

RWR does not check the names you pass against the ones your blueprints declare.
`rwr all --profile worx` produces no error and no warning: nothing matches
`worx`, so only the base items are applied.

**Solution**: Compare against `rwr profiles`. Names are case-sensitive — `Work`
and `work` are different profiles.

#### Fewer Items Than Expected

1. Verify profile names match exactly (case-sensitive)
2. Check that the items carry the `profiles` field you think they do
3. Confirm the blueprint type supports `profiles` at all — fonts and
   configuration blueprints do not

#### More Items Than Expected

1. If you passed no `--profile`, that is the cause: no filtering happens
2. Check for items without a `profiles` field — they are base items and always
   apply
3. Verify multi-profile items aren't matching a profile you did not intend

### Profile Debugging Tips

#### Get Available Profiles

```bash
rwr profiles
```

#### Dry Run with Profiles

```bash
rwr all --dry-run --profile work
```

#### Debug Mode

```bash
rwr all --debug --profile work
```

Reports, for each blueprint file, how many entries survived the filter:

```text
DEBU Filtering packages: 3 total, 3 matching active profiles [work]
DEBU Filtering services: 1 total, 1 matching active profiles [work]
```

There is no per-item line explaining an individual decision.

## Migration Guide

### From Non-Profile Configurations

Existing configurations work without modification. To add profiles:

1. **Identify Base Items**: Items everyone needs regardless of context
2. **Group Similar Items**: Identify packages that belong together
3. **Add Profile Fields**: Add `profiles: ["profile-name"]` to appropriate items
4. **Test Combinations**: Verify different profile combinations work correctly

### Example Migration

**Before** (no profiles):

```yaml
packages:
  - name: vim
    action: install
  - name: docker
    action: install
  - name: steam
    action: install
```

**After** (with profiles):

```yaml
packages:
  # Base item - always installed
  - name: vim
    action: install

  # Work profile item
  - name: docker
    profiles: ["work"]
    action: install

  # Gaming profile item
  - name: steam
    profiles: ["gaming"]
    action: install
```

## FAQ

### Can I use any profile names?

Yes! Profile names are completely user-defined. Use whatever makes sense for your setup.

### What happens if I don't specify any profiles?

Everything is installed. With no active profiles RWR skips the filter entirely,
so profile items are applied along with the base items. To install the base items
and one profile, name that profile: `rwr all --profile work`.

### Can an item belong to multiple profiles?

Yes! Use `profiles: ["profile1", "profile2"]` to include an item in multiple profiles.

### Are profile names case-sensitive?

Yes, profile names are case-sensitive. `Work` and `work` are different profiles.

### How do I see what profiles are available?

Run `rwr profiles` to see all available profiles in your configuration.

### Can I use profiles with any blueprint type?

Almost. Packages, repositories, files, templates, directories, services, users,
groups, git repositories, scripts and SSH keys all support `profiles`. Fonts and
configuration blueprints do not, and because blueprints decode strictly, adding
the key there is an error rather than a no-op.

### Does RWR warn me about a profile name that does not exist?

No. An unknown name simply matches nothing. Use `rwr profiles` to check.

## Related Documentation

- [Profile CLI Commands](cli/profiles.md) - Detailed CLI reference
- [Profile Best Practices](profile-best-practices.md) - Organizational guidelines
- [General Best Practices](best-practices.md) - Overall RWR best practices
- [Template Variables](variables.md) - Using variables with profiles

For specific blueprint types, see the respective blueprint documentation pages which include profile examples.
