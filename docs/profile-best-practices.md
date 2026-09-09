# Profile Best Practices

## Overview

This guide provides practical tips and examples for using profiles effectively in RWR configurations. These are suggestions based on common use cases, not rules you must follow.

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

Group tools by the technologies you're working with.

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

Items can belong to multiple profiles, which is useful for shared tools.

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

Use the profiles command to understand what's available in your configuration.

```bash
# See all available profiles
rwr profiles

# See what would be installed for a profile
rwr profiles --show development

# Get statistics about profile usage
rwr profiles --stats
```

## Testing Configurations

Test your profile configurations before running them.

```bash
# Dry run to see what would happen
rwr all --profile development --dry-run

# Check specific combinations
rwr all --profile frontend --profile development --dry-run
```

## Organization Tips

### Start Simple

Begin with basic profiles and add complexity as needed.

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

Remember that base items (no profile) are always installed, and profiles add to that.

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

### Guard Scripts for Machines That Lack the Subject

Naming no profile at all runs *everything*, profile items included - the
filter only narrows a run when at least one `--profile` is passed. A script
gated behind a profile therefore needs to survive running on a machine that
was never meant for it, because a forgotten `--profile` is all it takes to
put it there.

The pattern is: check for the subject, and leave quietly when it is not
there. Fail loudly only for a prerequisite the operator can fix (a missing
tool, an unset session variable), never for the machine's shape:

```bash
#!/usr/bin/env bash
set -euo pipefail

# Prerequisites the operator can fix fail loudly...
command -v gpg >/dev/null 2>&1 || { echo "gpg is not installed" >&2; exit 1; }

# ...the machine's shape does not: no key, no work, exit 0.
# Read the listing itself - gpg exits 0 even when nothing matches.
if ! gpg --list-secret-keys --with-colons "$FPR" 2>/dev/null | grep -q '^sec:'; then
  echo "key $FPR not in the keyring - nothing to back up"
  exit 0
fi
```

A restore script inverts it: leave quietly when the subject is *already*
there, so re-runs and wrong-machine runs are no-ops. Both guards together are
what make a tree safe to point at any machine with any profile combination.
For a complete example of the pattern, see
[examples/bitwarden](../examples/bitwarden/README.md), whose backup and
restore scripts are built from it.

### Profile Inheritance

You can simulate inheritance by using multiple profiles.

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

If a profile doesn't seem to be working:

1. Check the profile name matches exactly (case-sensitive)
2. Verify the profile exists in your configuration
3. Use `rwr profiles --show <profile-name>` to see what should be included

### Unexpected Installations

If you're getting unexpected packages:

1. Remember base items (no profile) are always installed
2. Check if items belong to multiple profiles
3. Use `--dry-run` to preview what will be installed

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

Be mindful of profile combinations that might install conflicting software.

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

This guide provides practical examples without telling you how you must organize your profiles. Use what works for your specific needs and context.
