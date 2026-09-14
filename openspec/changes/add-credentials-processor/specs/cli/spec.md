## ADDED Requirements

### Requirement: Processor exclusions
RWR SHALL accept repeatable, comma-separated `--except` processor names and `blueprints.except` defaults. Names SHALL be normalized and validated before execution. Named invocations SHALL override init exclusions; CLI exclusions SHALL win. An empty effective selection SHALL remain empty.

#### Scenario: Explicit conflict
- **WHEN** `rwr run credentials --except credentials` is invoked
- **THEN** RWR reports a selection error before provider access

#### Scenario: Excluded credential setup
- **WHEN** `rwr run all --except credentials` is invoked
- **THEN** independent provisioning continues without managed vault access

#### Scenario: Absent processor
- **WHEN** a known processor with no files is excluded
- **THEN** exclusion is accepted without an unknown-name error
