# Profile Best Practices

## Overview

This guide gives practical tips and examples for profiles in RWR configurations. These are suggestions from common use cases. They are not rules that you must obey.

## Common Use Cases

### Environment Separation

Use profiles to separate development, staging, and production configurations.

```yaml
packages:
  # Always installed (using names for multiple packages)
  - names:
      - git
      - curl
      - htop
    action: install

  # Development only (package list)
  - names:
      - docker
      - nodejs
      - npm
      - python3
    action: install
    profiles:
      - dev

  # Production monitoring
  - names:
      - datadog-agent
      - prometheus-node-exporter
    action: install
    profiles:
      - prod
```

### Role-Based Configuration

Different tools for different team members.

```yaml
packages:
  # Everyone gets these
  - names:
      - slack
      - git
    action: install

  # Developers
  - names:
      - vscode
      - docker
      - nodejs
    action: install
    profiles:
      - developer

  # Designers
  - names:
      - figma
      - sketch
      - adobe-creative-suite
    action: install
    profiles:
      - designer
```

### Personal vs Work

Separate personal and work-related installations.

```yaml
packages:
  # Work tools
  - names:
      - slack
      - zoom
      - vpn-client
      - teams
    action: install
    profiles:
      - work

  # Personal tools
  - names:
      - steam
      - vlc
      - spotify
    action: install
    profiles:
      - personal
```

### Technology Stacks

Group the tools by technology.

```yaml
packages:
  # Frontend development
  - names:
      - nodejs
      - yarn
      - npm
      - webpack
    action: install
    profiles:
      - frontend
      - fullstack

  # Backend development
  - names:
      - postgresql
      - redis
      - docker
      - nginx
    action: install
    profiles:
      - backend
      - fullstack

  # Data science
  - names:
      - python3
      - jupyter
      - pandas
      - numpy
    action: install
    profiles:
      - datascience
```

## Multiple Profiles

An item can belong to more than one profile. This is useful for shared tools.

```yaml
packages:
  # Shared across multiple contexts
  - names:
      - docker
      - git
      - tmux
    action: install
    profiles:
      - backend
      - frontend
      - devops

  # Specific to one context
  - names:
      - react-native-cli
      - android-studio
    action: install
    profiles:
      - mobile
```

## Profile Discovery

Use the `rwr profiles` command to see the profiles in your configuration.

```bash
# See all available profiles
rwr profiles

# See what would be installed for a profile
rwr profiles --show development

# Get statistics about profile usage
rwr profiles --stats
```

## Testing Configurations

Test your profile configurations before you run them.

```bash
# Dry run to see what would happen
rwr all --profile development --dry-run

# Check specific combinations
rwr all --profile frontend --profile development --dry-run
```

## Organization Tips

### Start Simple

Begin with basic profiles. Add more profiles when you need them.

```yaml
# Start with this
packages:
  - names:
      - git
    action: install
  - names:
      - docker
    action: install
    profiles:
      - dev

# Add more later as you understand your needs
```

### Use Meaningful Names

Choose profile names that make sense to you and your team.

```yaml
# Clear and meaningful
profiles: [work, personal, gaming, development]

# You decide what works for your context
profiles: [laptop, desktop, server, minimal]
```

### Document Your Profiles

Add comments to explain what each profile is for.

```yaml
packages:
  # Development environment setup
  - name: docker
    action: install
    profiles: [dev]  # Local development only

  # Design tools for creative work
  - name: figma
    action: install
    profiles: [design]  # UI/UX designers
```

## Common Patterns

### Additive Approach

RWR always installs the base items (items with no profile). Profiles add items to this base.

```yaml
packages:
  # Base system - always installed
  - name: git
    action: install
  - name: curl
    action: install

  # Additional tools per profile
  - name: docker
    action: install
    profiles: [development]
  - name: slack
    action: install
    profiles: [work]
```

### Profile Inheritance

You can simulate inheritance with multiple profiles.

```yaml
packages:
  # Basic development tools
  - name: git
    action: install
    profiles: [dev-basic]

  # Advanced development tools
  - name: docker
    action: install
    profiles: [dev-advanced]

  # Use both: --profile dev-basic --profile dev-advanced
```

### Environment-Specific Configs

Use profiles for different deployment environments.

```yaml
files:
  # Development config
  - path: /etc/app/config.yml
    source: configs/dev-config.yml
    profiles: [development]

  # Production config
  - path: /etc/app/config.yml
    source: configs/prod-config.yml
    profiles: [production]
```

## Troubleshooting

### Profile Not Working

If a profile has no effect:

1. Make sure that the profile name is an exact match (names are case-sensitive)
2. Make sure that the profile exists in your configuration
3. Use `rwr profiles --show <profile-name>` to see the items that the profile includes

### Unexpected Installations

If RWR installs packages that you did not expect:

1. RWR always installs the base items (items with no profile)
2. Find the items that belong to more than one profile
3. Use `--dry-run` to see what RWR installs

## Performance Considerations

### Large Configurations

For large configurations with many profiles:

```bash
# Only install what you need
rwr all --profile specific-profile

# Rather than installing everything
rwr all --profile profile1 --profile profile2 --profile profile3
```

### Profile Combinations

Some profile combinations can install software that conflicts. Think about this before you combine profiles.

```yaml
packages:
  # These might conflict
  - name: python2
    action: install
    profiles: [legacy]
  - name: python3
    action: install
    profiles: [modern]

  # Document the conflict in comments
  # Note: Don't use 'legacy' and 'modern' profiles together
```

This guide gives practical examples. It does not tell you how you must organize your profiles. Use what works for your needs and context.
