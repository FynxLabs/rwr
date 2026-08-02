# Tasks

Steps 1–5 are prerequisites and carry the risk; 6+ are additive.

- [x] 1. `Plan` struct + stage 1 resolve; wire `rwr validate` to it. Parity test
      against current validate output (fails if outputs diverge).
- [x] 2. `Reporter` interface + `LogReporter` wired into `All()` and
      `commands.go`. Byte-identical-output test vs master (the safety net).
- [ ] 3. Store, views, slog handler; test processor attribution of records.
- [ ] 4. Stage 2 resolve: provider grouping, lane counts.
- [ ] 5. Error accumulation in `All()`; test: non-interactive run with one
      failing processor continues, exits nonzero (fails today — first error
      aborts).
- [ ] 6. Static TUI: header, strip, collapsed list, panel, help bar.
- [ ] 7. Terminal handoff (`tea.ExecProcess`); test sudo prompt and per-item
      `interactive: true` under `--interactive=false`.
- [ ] 8. Filters, search, pinning.
- [ ] 9. Summary + dry-run frames.
- [ ] 10. Theming + glyph fallback (`NO_COLOR=1`, `TERM=xterm`, conhost ASCII).
- [ ] 11. Mouse + hover (bubblezone).
- [ ] 12. Animation (harmonica; kill ticks when springs settle).
- [ ] 13. Compact mode + live resize.
- [ ] 14. OSC 9 notification (unfocused, >30s, recognized terminal only).
- [ ] 15. Full test checklist from design.md §18.
