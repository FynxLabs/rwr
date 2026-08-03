# Tasks

Steps 1–5 are prerequisites and carry the risk; 6+ are additive.

- [x] 1. `Plan` struct + stage 1 resolve; wire `rwr validate` to it. Parity test
      against current validate output (fails if outputs diverge).
- [x] 2. `Reporter` interface + `LogReporter` wired into `All()` and
      `commands.go`. Byte-identical-output test vs master (the safety net).
- [x] 3. Store, views, slog handler; test processor attribution of records.
- [x] 4. Stage 2 resolve: provider grouping, lane counts.
- [x] 5. Error accumulation in `All()`; test: non-interactive run with one
      failing processor continues, exits nonzero (fails today — first error
      aborts).
- [x] 6. Static TUI: header, strip, collapsed list, panel, help bar.
- [x] 7. Terminal handoff (`tea.ExecProcess`); test sudo prompt and per-item
      `interactive: true` under `--interactive=false`.
- [x] 8. Filters, search, pinning.
- [x] 9. Summary + dry-run frames.
- [x] 10. Theming + glyph fallback (`NO_COLOR=1`, `TERM=xterm`, conhost ASCII).
- [ ] 11. Mouse + hover (bubblezone). PARTIAL: native click-to-select on the
      strip and wheel/click pinning are in; bubblezone hover is NOT — the
      released bubblezone targets bubbletea v1 (github.com module path) and
      cannot be imported against the charm.land v2 stack. Needs bubblezone v2
      or a fork.
- [x] 12. Animation (harmonica; kill ticks when springs settle).
- [x] 13. Compact mode + live resize.
- [x] 14. OSC 9 notification (unfocused, >30s, recognized terminal only).
- [ ] 15. Full test checklist from design.md §18. REMAINING: per-item
      ResourceDone/LaneUpdate emission from the ten Process* functions
      (mechanical, design §5) so lanes fill live; manual sudo-prompt handoff
      check on a real TTY; NO_COLOR/TERM=xterm/conhost visual passes.
