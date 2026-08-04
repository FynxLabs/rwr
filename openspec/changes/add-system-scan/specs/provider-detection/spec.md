# Provider Detection - Deltas

## ADDED Requirements

### Requirement: Providers may declare an explicit-install list command

The provider schema SHALL accept an optional `list_explicit` command - the
package manager's query for explicitly-installed packages (`pacman -Qe`,
`apt-mark showmanual`, `brew leaves`) - and scan consumers SHALL prefer it
over `list`. Absence is not an error: consumers fall back to
`list` and say so.

#### Scenario: Schema accepts the verb

- **GIVEN** a CUE provider declaring `list_explicit`
- **WHEN** the export pipeline runs
- **THEN** it exports and the loaded provider carries the command

#### Scenario: Absence falls back

- **WHEN** a provider defines only `list`
- **THEN** loading succeeds and scans mark its results unfiltered
