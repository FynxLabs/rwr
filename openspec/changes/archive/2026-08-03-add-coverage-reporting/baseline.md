# Coverage baseline (2026-08-02, task 1)

Measured with `go test -coverprofile` across `./...`:

| Package | Coverage |
|---|---|
| cmd | 42.7% |
| internal/helpers | 45.7% |
| internal/processors | 55.3% |
| internal/prompts | 8.1% |
| internal/system | 57.6% |
| internal/types | 64.2% |
| internal/validate | 46.0% |
| **total** | **52.8%** |

(`main` and `internal/exectest` carry no tests: the former is a thin
entrypoint, the latter is itself test tooling.)

Gate threshold: **50%** - at-or-below the measured total so CI does not go
red on day one; ratchet deliberately as coverage grows (prompts at 8.1% is
the obvious first target).
