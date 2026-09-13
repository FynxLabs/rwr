## MODIFIED Requirements

### Requirement: Trees declare the credentials they need
RWR SHALL accept named init credentials with ordered sources and preserve legacy env, keyring, prompt and Bitwarden syntax. It SHALL resolve only selected resource dependencies, after profile filtering. Ordinary lookup MUST NOT install, log in, unlock, prompt, or persist secrets. Missing credentials SHALL skip dependent resources by default; explicit strict policy SHALL record failures while independent work continues. Unknown dependency names SHALL fail validation.

Declared credentials SHALL remain withheld from templates and child environments unless exposed, and SHALL be redacted in logs. Required values MUST NOT be replaced with empty strings. Provider exclusions SHALL prohibit all managed provider calls, including probes, while permitting already supplied environment values and approved noninteractive keyring values.

#### Scenario: Declared credential resolves from the environment
- **WHEN** a selected consumer requires a declaration with an available environment source
- **THEN** resolution succeeds without prompting and exposure remains gated

#### Scenario: Filtered consumer
- **WHEN** a credential-dependent script's profile is not selected
- **THEN** its credential is never resolved

#### Scenario: Unresolvable credential fails up front
- **WHEN** a selected consumer cannot obtain a credential under strict failure policy
- **THEN** that consumer fails before its operation executes, without running with an empty value
- **AND** initialization remains configuration-only and independent resources continue

#### Scenario: Unavailable dependency under skip policy
- **WHEN** a selected consumer cannot obtain a credential under skip policy
- **THEN** that resource is skipped with a reason and independent resources continue successfully

#### Scenario: Executed operation fails
- **WHEN** a credential task executes and fails
- **THEN** the run reports failure rather than reclassifying it as an availability skip
