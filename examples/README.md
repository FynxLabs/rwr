# RWR examples

These directories show working blueprint layouts and processor syntax. Start
with the example closest to your task; there is no need to read them all.

## Choose an example

| Goal | Example |
|---|---|
| See the smallest possible tree | [Minimal and flattened layouts](alternative_layouts/) |
| Compare Linux distributions | [Linux](linux/) |
| Build a macOS setup | [macOS](macos/) |
| Build a Windows setup | [Windows](windows/) |
| Share resources across machines | [Multi-machine tree](multi-machine/) |
| Reuse entries through imports | [Nested imports](imports/) |
| Set up Bitwarden and GPG tasks | [Bitwarden credentials](bitwarden/) |
| Manage Omarchy through the configuration blueprint | [Omarchy](omarchy/) |

The platform trees include YAML, JSON, TOML, and CUE versions so you can compare
the same idea in your preferred format.

## Directory layout is your choice

RWR can identify a blueprint from a recognized processor directory:

```text
blueprints/
├── packages/
│   └── common.yaml
└── files/
    └── shell.yaml
```

It can also identify it from top-level keys, so a flat file works:

```yaml
packages:
  - name: git
    action: install

files:
  - name: example.conf
    action: create
    target: "{{ .User.home }}/.config/example"
    content: "enabled=true"
```

That file is routed to both the `packages` and `files` processors. Choose folders
that make the repository easy for you to maintain.

Examples contain placeholder package names, repository URLs, users, and paths.
Copy the structure you need, replace those values, then validate and preview the
result:

```bash
rwr validate PATH_TO_TREE
rwr all --init-file PATH_TO_INIT --dry-run
```

For the underlying rules, read [How blueprints work](../docs/blueprints-general.md).
