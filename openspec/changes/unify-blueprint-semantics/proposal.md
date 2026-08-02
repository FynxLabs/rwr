# Change: Unify blueprint semantics and close the download-integrity schema gaps

Depends on: `add-format-registry` (per-file formats; content detection decodes
through the registry). Carries the schema half of the 2026-08 security review's
HIGH finding and every cross-processor inconsistency it catalogued.

## Why

Six semantics questions have two answers each in today's code — import base
directories, name/names precedence, template strictness at validate, dead
init-file sections, path-only type dispatch — and the blueprint schema cannot
express a content digest for the two most security-sensitive downloads it
causes: repository signing keys and `files:` sources. Each inconsistency is a
blueprint that behaves differently depending on which processor reads it.

## What Changes

**Download-integrity schema (completes the review's HIGH):**

- `types.Repository` gains `key_sha256`: the digest of the signing key at
  `key_url`. Wired into the apt/dnf key-download steps as the step `sha256`.
  Policy: a repository with `key_url` but no `key_sha256` logs a prominent
  warning ("unpinned signing key") now; a later major refuses.
- `types.File` gains `sha256`: URL sources are verified before install
  (`files.go` currently hard-codes an empty digest). Same warn-now policy for
  URL sources without one.

**Semantics unification (each is a current two-answer inconsistency):**

- Import resolution is file-relative everywhere. Six processors (packages,
  git, ssh_keys, services, users, repositories) resolve top-level imports
  against the tree root while files/scripts/fonts/configuration are
  file-relative — the blueprint-processing spec already mandates
  file-relative, so this also closes the open code-violates-spec item.
- `names` wins over `name` when both are set, everywhere. files/fonts already
  do this; packages had it backwards. Declaring both is a validate warning.
- Init-file inline resource sections (`repositories`, `packages`, `services`,
  `files`, `templates`, `directories`, `configuration`) are REMOVED from the
  schema. They were decoded, validated, profile-counted, and never applied at
  runtime — a declaration that silently does nothing. (Decision: drop rather
  than apply; blueprints stay the single declaration path. A `rwr convert`
  /migration tool is a planned follow-up change for moving old trees forward.)
- Content-based blueprint-type detection: a file whose path names no processor
  directory is decoded shallowly and typed by its top-level keys
  (`packages:` → packages, …). This makes the shipped
  `examples/alternative_layouts/` (flattened, minimal_files) actually execute —
  today they exit 0 having run nothing — and honors the README's promise.
  Ambiguous or unrecognized content keeps the loud unrouted-file warning.
- Validate template strictness matches the spec: `missingkey=zero` only for
  the `UserDefined` namespace; `User`/`System`/`Flags` references that do not
  exist are validate errors.

**Riders:** tautological tests deleted as encountered (roadmap item);
`alternative_layouts` READMEs corrected (`rwr run` is not a command).

## Breakage

- Init-file inline resource sections now fail strict decode (unknown keys).
  They never did anything; a tree using them was already not getting what it
  wrote. Migration: move the entries into blueprint files (future `rwr
  convert` automates this).
- A packages entry declaring BOTH `name` and `names` now installs the `names`
  list (was: only `name`). Declaring both also warns at validate.
- Validate becomes stricter about unknown `.User/.System/.Flags` template
  references — previously silently zero-valued. Runs were already strict;
  this only moves the failure earlier.
- Everything else is additive (`key_sha256`, `files[].sha256`, content
  detection for previously dead layouts).

## Impact

- Affected specs: `blueprint-processing`, `blueprint-validation`,
  `initialization`.
- Affected code: `internal/types` (Repository, File, initialize),
  `internal/processors` (six import sites, packages precedence, blueprints
  dispatch, repository step data), `internal/validate`, provider TOMLs
  (apt/dnf key steps), `examples/alternative_layouts/`, docs.
