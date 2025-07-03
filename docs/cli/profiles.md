# Profile CLI Commands

This page provides comprehensive documentation for profile-related CLI commands and flags in RWR. Profiles allow you to selectively install packages and configurations based on different contexts or use cases.

## Profile CLI Overview

RWR provides two main ways to work with profiles:

1. **Profile Selection Flags**: `--profile` / `-p` to specify which profiles to activate
2. **Profile Discovery Command**: `rwr profiles` to explore available profiles in your configuration

## Profile Selection Flags

### `--profile` / `-p` Flag

The profile flag allows you to specify which profiles should be active during RWR execution.

#### Syntax

```bash
rwr [command] --profile PROFILE1,PROFILE2,...
rwr [command] -p PROFILE1 -p PROFILE2
```

#### Single Profile

```bash
# Long form
rwr --profile work

# Short form
rwr -p work

# With specific commands
rwr run packages --profile work
rwr run services -p gaming
```

#### Multiple Profiles

```bash
# Comma-separated (recommended)
rwr --profile work,gaming,dev

# Multiple flags
rwr -p work -p gaming -p dev

# Mixed usage
rwr --profile work,gaming -p dev
```

#### Special "all" Profile

```bash
# Install everything regardless of profiles
rwr --profile all
rwr -p all

# "all" overrides other profiles
rwr --profile work,gaming,all  # Same as --profile all
```

### Profile Behavior with Different Commands

#### Default RWR Command

```bash
# No profiles - base items only
rwr

# With profiles - base + profile items
rwr --profile work
rwr --profile work,gaming
```

#### Run Command with Specific Processors

```bash
# Run specific processor with profiles
rwr run packages --profile work
rwr run services --profile gaming
rwr run files --profile work,dev

# Multiple processors with profiles
rwr run packages,services --profile work
```

#### Other Commands with Profiles

```bash
# Validate with profiles
rwr validate --profile work

# Debug with profiles
rwr --debug --profile work

# Dry run with profiles
rwr --dry-run --profile work,gaming
```

## Profile Discovery Command

### `rwr profiles` Command

The `profiles` command scans your configuration and displays all available profiles with usage statistics and examples.

#### Basic Usage

```bash
rwr profiles
```

#### Example Output

```text
Available user-defined profiles:
  - work (found in 15 items across packages, services, files)
  - gaming (found in 8 items across packages, services)
  - dev (found in 12 items across packages, files, users)
  - personal (found in 5 items across packages, files)

Usage examples:
  rwr                          # Items with no profiles field only
  rwr --profile work           # No-profile items + work profile
  rwr --profile work,gaming    # No-profile items + work + gaming profiles
  rwr --profile all            # Everything (all profiles)

Note: Profile names are completely user-defined. Use whatever makes sense for your setup!
```

#### Profile Statistics

The `profiles` command provides helpful statistics:

* **Profile Names**: All unique profile names found in your configuration
* **Item Count**: How many items use each profile
* **Blueprint Types**: Which blueprint types contain each profile
* **Usage Examples**: Practical command examples for your specific profiles

## Command Integration

### Global Flags

Profile flags work with all RWR commands as global flags:

```bash
# These are equivalent
rwr --profile work run packages
rwr run packages --profile work

# Global flags can appear anywhere
rwr --debug --profile work run services --dry-run
```

### Command-Specific Behavior

#### Standard Commands

```bash
# Run all processors with profiles
rwr --profile work

# Validate configuration with profiles
rwr validate --profile work

# Show help
rwr --help  # Displays profile flag documentation
```

#### Run Command

```bash
# Run specific processors
rwr run packages --profile work
rwr run services,files --profile gaming

# Run all processors (equivalent to base rwr command)
rwr run all --profile work
```

#### Bootstrap Command

```bash
# Bootstrap with profiles (if supported in your bootstrap configuration)
rwr bootstrap --profile minimal
```

## Advanced Usage Patterns

### Profile Validation

RWR automatically validates that specified profiles exist in your configuration:

```bash
# Valid profile
rwr --profile work
# ✓ Proceeds normally

# Invalid profile
rwr --profile worx
# ✗ Error: Profile 'worx' not found. Available profiles: work, gaming, dev
# Suggestion: Did you mean 'work'?
```

### Debugging Profile Behavior

#### Debug Mode

```bash
rwr --debug --profile work
```

Example debug output:

```text
DEBUG: Active profiles: [work]
DEBUG: Including package 'vim' (no profiles - always installed)
DEBUG: Including package 'docker' (profiles: [work])
DEBUG: Skipping package 'steam' (profiles: [gaming], selected: [work])
DEBUG: Including service 'nginx' (profiles: [work, dev])
```

#### Dry Run Mode

```bash
rwr --dry-run --profile work,gaming
```

Shows what would be installed without actually performing actions.

### Profile Discovery Workflow

#### Exploring New Configurations

```bash
# 1. First, see what profiles are available
rwr profiles

# 2. Try a specific profile with dry run
rwr --dry-run --profile work

# 3. Run with debug to understand behavior
rwr --debug --profile work

# 4. Execute when satisfied
rwr --profile work
```

## Error Handling

### Common Profile Errors

#### Profile Not Found

```text
Error: Profile 'xyz' not found. Available profiles: work, gaming, dev
```

**Resolution**: Check spelling or run `rwr profiles` to see available options.

#### No Profiles Available

```text
Warning: No profiles found in configuration. All items will be processed.
```

**Resolution**: This is normal for configurations without profile fields.

#### Multiple Profile Issues

```text
Error: Invalid profile combination. Profile 'conflicting' conflicts with 'work'.
```

**Resolution**: This would only occur with custom validation logic (not built into core RWR).

### Troubleshooting Commands

#### List Available Profiles

```bash
rwr profiles
```

#### Validate Configuration

```bash
rwr validate --profile work
```

#### Debug Profile Selection

```bash
rwr --debug --dry-run --profile work
```

## Command Reference

### CLI Profile Selection Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--profile PROFILES` | Comma-separated list of profiles to activate | `--profile work,gaming` |
| `-p PROFILE` | Single profile to activate (can be repeated) | `-p work -p gaming` |

### Profile Discovery Commands

| Command | Description | Example |
|---------|-------------|---------|
| `rwr profiles` | List all available profiles with statistics | `rwr profiles` |

### Global Flag Combinations

| Combination | Description |
|-------------|-------------|
| `--profile work --debug` | Run work profile with debug output |
| `--profile work --dry-run` | Show what would be installed for work profile |
| `--profile all --validate` | Validate all items regardless of profiles |
| `-p work -p gaming --debug` | Multiple profiles with debug output |

## Examples by Use Case

### Developer Workflow

```bash
# Check available development profiles
rwr profiles

# Set up frontend development environment
rwr --profile frontend

# Set up full development stack
rwr --profile frontend,backend,database

# Quick development setup with debugging
rwr --debug --profile dev
```

### System Administration

```bash
# Server setup
rwr --profile server

# Desktop workstation setup
rwr --profile desktop,productivity

# Minimal system setup
rwr --profile minimal

# Complete system setup
rwr --profile all
```

### Multi-Environment Management

```bash
# Home setup
rwr --profile home,personal

# Office setup
rwr --profile office,work

# Travel setup
rwr --profile travel,minimal

# Context switching
rwr profiles  # See what's available
rwr --profile travel
```

### Testing and Validation

```bash
# Test specific profile combinations
rwr --dry-run --profile work,gaming

# Validate configuration for specific profiles
rwr validate --profile work

# Debug profile behavior
rwr --debug --profile work 2>&1 | grep -i profile
```

## Best Practices

### Command Line Usage

1. **Use Short Flags for Interactive Use**: `-p` is faster to type than `--profile`
2. **Use Long Flags for Scripts**: `--profile` is more readable in automation
3. **Group Related Profiles**: `--profile work,dev` is better than separate commands
4. **Test Before Running**: Use `--dry-run` to verify profile behavior

### Profile Discovery

1. **Explore Before Using**: Always run `rwr profiles` in new configurations
2. **Understand Profile Coverage**: Check which blueprint types use each profile
3. **Validate Profile Names**: Use tab completion or check available profiles

### Debugging

1. **Use Debug Mode**: `--debug` helps understand profile filtering decisions
2. **Combine Flags**: `--debug --dry-run` is safe for exploration
3. **Check Statistics**: `rwr profiles` shows how many items use each profile

## Related Documentation

* [Profile System Overview](../profiles.md) - Complete profile system documentation
* [CLI Commands & Flags](command-and-flags.md) - General CLI documentation
* [Profile Best Practices](../profile-best-practices.md) - Organizational guidelines
* [General Best Practices](../best-practices.md) - Overall RWR best practices

For profile configuration examples, see the individual blueprint type documentation pages.
