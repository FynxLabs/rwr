# Blueprint Validation — Deltas

## ADDED Requirements

### Requirement: Template strictness at validate matches the run

`rwr validate` SHALL resolve template references strictly for the `User`,
`System`, and `Flags` namespaces — a reference that does not exist is a
validation error — and leniently (`missingkey=zero`) only for `UserDefined`,
whose values legitimately vary per machine.

Why: validate resolved every namespace leniently, so a typo like
`{{ .User.hoem }}` validated clean and failed at run time — the exact class of
error validate exists to catch early.

#### Scenario: Misspelled built-in reference

- **WHEN** a blueprint references `{{ .User.hoem }}`
- **THEN** `rwr validate` reports it as an error naming the reference
- **AND** an undefined `{{ .UserDefined.anything }}` still validates

### Requirement: Declaring both name and names is flagged

An entry declaring both `name` and `names` SHALL produce a validation warning
naming the entry, since only the `names` list will be processed.

#### Scenario: Both declared

- **WHEN** a packages entry declares both `name` and `names`
- **THEN** validate warns that `name` is ignored
