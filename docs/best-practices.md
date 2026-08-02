# Best Practices

This page gives the best practices for Rinse, Wash, Repeat (RWR) on personal machines. It covers blueprint organization, configuration management, and system maintenance.

## Organizing Blueprints

When you organize your blueprints, use these practices:

1. Use a consistent naming convention for your blueprint files, such as `<blueprint-type>.yaml` (for example `packages.yaml` or `repositories.yaml`).

2. Group related blueprints into subdirectories for better organization. For example:

    ```text
    blueprints/
    ├── packages/
    │   ├── core.yaml
    │   └── development.yaml
    ├── repositories/
    │   ├── common.yaml
    │   └── personal.yaml
    └── services/
        └── web.yaml
    ```

3. Split large blueprints into smaller, focused files for better readability and maintainability.

4. Use meaningful names for your blueprints, variables, and templates to make your configuration self-explanatory.

## Managing Configurations

When you manage configurations for your personal machine, use these practices:

1. Use variables to parameterize values that can change, such as file paths, URLs, or personal preferences.

2. Create separate blueprint files for different use cases or configurations (for example `core.yaml` for essential packages, and `development.yaml` for development tools and libraries).

3. Use a version control system (for example Git) to track changes to your blueprints and keep a history of your configurations.

4. You can use branches or tags to represent different configurations or snapshots of your system.

## Keeping Your System Maintainable

To keep your system maintainable with RWR, use these practices:

1. Regularly update your blueprints to reflect changes in your system or application requirements.

2. Use comments in your blueprints to document complex configurations or provide explanations for specific settings.

3. Do not duplicate configuration values across blueprints. Instead, use variables or templates to centralize common configuration.

4. Test your blueprints in a virtual machine before you apply them to your local system.

5. Keep a backup of your system configuration, or create snapshots, before you apply significant changes to your local system.

## Using Profiles Effectively

With profiles, you can install different configurations for different contexts. Use these approaches:

1. **Environment-based profiles**: Separate development, testing, and production configurations.

    ```yaml
    packages:
      # Always installed
      - name: git
        action: install
      # Development only
      - name: docker
        action: install
        profiles: [development]
    ```

2. **Role-based profiles**: Organize tools by user roles or responsibilities.

    ```yaml
    packages:
      # Developer tools
      - name: vscode
        action: install
        profiles: [developer]
      # Designer tools
      - name: figma
        action: install
        profiles: [designer]
    ```

3. **Use meaningful profile names**: Choose names that make sense to you and your team. There are no naming restrictions - use whatever works for your context.

4. **Document your profiles**: Add comments that explain the purpose of each profile.

5. **Test profile combinations**: Use `--dry-run` to see what RWR installs before you run the command.

## Security Considerations

When you work with RWR, use these security practices:

1. RWR withholds your GitHub token and SSH private key from blueprints. Give a
   credential to a blueprint only when the blueprint needs it, with the
   `exposeCredentials` key. Read [Credentials](credentials.md) before you use
   that key.

2. RWR removes credential values from the logs. Use the `--show-secrets` flag
   only when you must see a value, and do not keep that output.

3. Update your system packages and dependencies regularly to remove known vulnerabilities.

4. Limit access to your RWR configuration files and repositories to authorized users only.

5. Use secure communication channels (for example HTTPS or SSH) when you access remote repositories or servers.

These practices give you organized, maintainable, and secure configurations.
