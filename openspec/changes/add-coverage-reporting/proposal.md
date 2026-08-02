# Change: Code coverage reporting and thresholds

(Was issue #102.)

## Why

Tests exist (70+ test files) but coverage is not measured or enforced; a
regression in test coverage is invisible in review.

## What Changes

- `mise` tasks: `test:coverage` (exists), add `test:coverage:html` and
  `test:coverage:summary`.
- A coverage gate script with a global threshold (starting point 70%,
  calibrated to actual current coverage before enforcement so CI doesn't go
  red on day one).
- CI runs the gate on the ubuntu test job.
- README coverage badge.

## Breakage

Nothing breaks. CI gains a gate; the threshold is set at-or-below current
measured coverage initially and ratcheted deliberately.

## Impact

- Affected specs: none (tooling only).
