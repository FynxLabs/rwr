# Quick start

This walkthrough creates one text file in your home directory. It is small on
purpose: the goal is to see the complete RWR workflow before adding packages,
services, or privileged changes.

## 1. Install RWR

Use the command for your platform in the [installation guide](install.md), then
check the binary:

```bash
rwr version
```

## 2. Create a blueprint tree

Make a directory with this layout:

```text
my-blueprints/
├── init.yaml
└── files/
    └── welcome.yaml
```

Put this in `init.yaml`:

```yaml
blueprints:
  format: yaml
  location: .
```

The init file is the entry point. Here it says that YAML blueprints live in the
same directory and its subdirectories.

Put this in `files/welcome.yaml`:

```yaml
files:
  - name: rwr-was-here.txt
    action: create
    target: "{{ .User.home }}"
    content: |
      This file was created by RWR.
```

`{{ .User.home }}` is filled from the current user at run time. RWR joins the
target directory with `name`, producing `rwr-was-here.txt` in your home
directory.

## 3. Validate and preview

From the directory containing `my-blueprints`, run:

```bash
rwr validate ./my-blueprints
rwr all --init-file ./my-blueprints/init.yaml --dry-run
```

Validation checks the tree's formats, fields, templates, and supported schema.
Dry-run shows the selected operations without changing the machine.

## 4. Apply it

```bash
rwr all --init-file ./my-blueprints/init.yaml
```

Check `rwr-was-here.txt` in your home directory. Running the same command again
should converge on the same result rather than creating another copy.

You can also inspect the run record:

```bash
rwr status --init-file ./my-blueprints/init.yaml
```

## 5. Save the location

You can keep passing `--init-file`, which is useful while experimenting. To save
the location for normal use, run:

```bash
rwr config create
```

Choose your init file when prompted. Future commands can then use `rwr all`
without the path.

## Add real resources

Add another blueprint file under the same tree. Each top-level key selects a
processor:

```yaml
packages:
  - names: [git, curl]
    action: install
```

Package names differ between operating systems and package managers. Use the
[platform examples](../examples/README.md) as a starting point, then validate and
dry-run again before applying.

To run only one part of the tree:

```bash
rwr run packages
rwr run files
```

## Add profiles when you need them

A profile keeps optional items in the same tree:

```yaml
packages:
  - name: git
    action: install
  - name: podman
    action: install
    profiles: [development]
```

```bash
rwr all --profile development
```

Unprofiled items always apply. Profiled items are filtered only when at least
one `--profile` is given; plain `rwr all` applies them all. Read
[Profiles](profiles.md) before designing a larger profile layout.

## Where to go next

- [How blueprints work](blueprints-general.md)
- [Blueprint type reference](blueprints/README.md)
- [Init file](init-file.md)
- [Variables and templates](variables.md)
- [Credentials and Bitwarden](credentials.md)
- [Omarchy desktop setup](blueprints/omarchy.md)

If you are starting from a machine you already configured by hand, read
[`rwr capture`](cli/capture.md).
