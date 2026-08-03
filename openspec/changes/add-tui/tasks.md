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
- [x] 11. Mouse + hover (bubblezone). bubblezone v2
      (github.com/lrstanley/bubblezone/v2, charm.land v2 deps) shipped;
      strip cells and list rows are zones, hover reverses the cell and names
      the processor, click selection goes through the zones instead of
      hardcoded coordinates, all-motion mouse mode set on the view.
- [x] 12. Animation (harmonica; kill ticks when springs settle).
- [x] 13. Compact mode + live resize.
- [x] 14. OSC 9 notification (unfocused, >30s, recognized terminal only).
- [ ] 15. Full test checklist from design.md §18. Per-item
      ResourceDone/LaneUpdate emission from all ten Process* functions is DONE.
      REMAINING: manual terminal checks only — sudo-prompt handoff on a real
      TTY; NO_COLOR/TERM=xterm/conhost visual passes.
