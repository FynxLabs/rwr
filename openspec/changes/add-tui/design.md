# rwr TUI Implementation Plan

Target repo: `github.com/fynxlabs/rwr` (master)

Replaces the current streaming-log output with a Bubble Tea dashboard. Non-TTY behavior is unchanged.

---

## 1. Terminology

| Term | Meaning |
|---|---|
| Processor | The 10 blueprint types plus pre-steps (blueprints, init, bootstrap). One strip block each. |
| Provider | Package manager from `internal/system/definitions/providers/*.toml`. Renders as a lane inside a processor panel. |
| Resource | A single item: one package, one file, one service, one font. Recorded always, rendered only in the summary. |

Do not use "step" or "engine" in user-facing strings. Help bar uses `j/k move`, not `j/k processor`.

---

## 2. Dependencies

Already in `go.sum` via `charm.land/huh/v2`. Promote to direct in `go.mod`:

- `charm.land/bubbletea/v2` v2.0.2
- `charm.land/bubbles/v2` v2.0.0
- `charm.land/lipgloss/v2` v2.0.1
- `github.com/catppuccin/go` v0.3.0
- `github.com/charmbracelet/colorprofile` v0.4.3

New:

- `github.com/charmbracelet/harmonica`
- `github.com/lrstanley/bubblezone`

Bubble Tea v2 changed message types. Key events are `KeyPressMsg`/`KeyReleaseMsg`; mouse events are split into click, motion, wheel, and release messages rather than a single `MouseMsg`. Verify against v2.0.2 source. Most examples online are v1 and will not compile.

Components to use rather than hand-roll: `viewport`, `progress`, `spinner`, `stopwatch`, `help`, `key`, `textinput`. Only the strip and the panel height animation are custom.

---

## 3. Prerequisites

Three changes outside the TUI. All are on the critical path.

### 3.1 Resolve phase and `Plan`

Hoist file reading, template resolution, and unmarshalling out of the execution loop in `internal/processors/all.go` into a resolve phase producing one struct.

```go
type Plan struct {
    Init      *types.InitConfig
    Order     []string                  // from GetBlueprintRunOrder
    Files     map[string][]ResolvedFile // from GetBlueprintFileOrder, plus resolved content
    Providers []ProviderState
    Resources []Resource
    Diags     []Diagnostic
}
```

Two stages, because provider detection has a dependency the rest does not:

| Stage | Runs | Produces |
|---|---|---|
| 1 | before init, needs only `SetPaths` + `DetectOS` | parse, imports, schema, template resolution, diagnostics |
| 2 | after init and bootstrap complete | provider states, resource enumeration, lane counts |

Bootstrap can install the package manager later blueprints depend on. Detecting providers before bootstrap produces wrong lanes.

Consumers of `Plan`: `rwr validate` (stage 1 only), the TUI, the executor. Do not add an output-format flag to validate and call it from the run path. That resolves twice.

### 3.2 Error accumulation in `All()`

`internal/processors/all.go` currently returns on the first processor error. Push-through mode does not exist yet.

- Fatal in both modes, abort immediately: blueprint fetch, blueprint location missing, package manager install, run order unresolvable. Everything before the processor loop.
- Recoverable, collected: anything returned by the ten `Process*` calls.

```go
if err != nil {
    if interactive {
        return fmt.Errorf("error processing %s: %w", processor, err)
    }
    stepErrs = append(stepErrs, StepError{Processor: processor, Err: err})
}
```

Exit nonzero if `stepErrs` is non-empty.

### 3.3 Output routing in `internal/system/commands.go`

Two branches, handled differently.

Interactive branch currently sets `command.Stdin/Stdout/Stderr = os.Std*`. This is deliberate. Capturing stderr swallows sudo's password prompt and hangs the run. Do not pipe it.

```go
if cmd.Interactive {
    done := make(chan error, 1)
    reporter.Emit(TerminalReq{Processor: currentProcessor, Cmd: command, Done: done})
    return <-done
}
```

TUI handles `TerminalReq` with `tea.ExecProcess`, which restores cooked mode, hands over the real tty, and repaints on exit. `LogReporter` implements it as the current direct wiring, so headless output is byte-identical to today.

This path must work in non-interactive runs too. `helpers.ResolveInteractive` lets a single blueprint item set `interactive: true` inside an otherwise non-interactive run. Do not gate handoff on the global flag.

Non-interactive branch, in `setOutputStreams`, replacing `cmd.Stdout = os.Stdout` under debug:

```go
cmd.Stdout = &lineWriter{proc: currentProcessor, src: SrcStdout}
cmd.Stderr = io.MultiWriter(&stderr, &lineWriter{proc: currentProcessor, src: SrcStderr})
```

`lineWriter` buffers to newline and emits one record per line. Keep the existing `stderr bytes.Buffer` so the error path still has full text. Per-blueprint `logName` files keep working via `io.MultiWriter`.

---

## 4. Event bus

Goal: `All()` and the ten processors do not know a TUI exists.

```go
type Reporter interface{ Emit(Event) }

type ProcStarted   struct{ Processor string; Files int; Providers []string }
type ProcFinished  struct{ Processor string; Err error; Dur time.Duration }
type ProcSkipped   struct{ Processor, Reason string }
type LaneUpdate    struct{ Processor, Provider string; Done, Total int; Status Status }
type ResourceDone  struct{ Resource Resource }
type TerminalReq   struct{ Processor string; Cmd *exec.Cmd; Done chan error }
type RunFinished   struct{ Errs []StepError }
```

Two implementations, chosen in `cmd/`:

- `TUIReporter` wraps `p.Send`
- `LogReporter` reproduces current streaming output exactly

### Log capture without touching the processors

`charmbracelet/log` implements `slog.Handler`. Install a custom handler at startup that stamps each record with a package-level atomic `currentProcessor` and appends to the store. Every existing `log.Infof("Processing packages")` across all ten processor files flows into the buffer correctly attributed, with zero edits to those files. Same pattern as the existing `dryRunMode` and `current` executor package state.

Explicit events are then needed in only two places: the `All()` loop, and `runCommand`.

---

## 5. Data model

```go
type LogRecord struct {
    Seq       uint64
    Time      time.Time
    Level     log.Level
    Processor string
    Provider  string
    Msg       string
    Attrs     []slog.Attr
    Src       Source // SrcLog | SrcStdout | SrcStderr
}

type Resource struct {
    Processor string
    Provider  string        // empty for files, services, git, scripts
    Name      string        // "neovim", "~/.config/nvim/"
    Action    string        // install, copy, enable, clone
    Status    Status        // ok, failed, skipped, unknown, planned, present
    Detail    string
    Dur       time.Duration
}

type Diagnostic struct {
    Severity  Severity // Error | Warning
    Processor string
    File      string
    Line      int
    Msg       string
}
```

`unknown` is required. rwr shells out and several providers do not report per-package results reliably. Do not force those into ok or failed.

Each `Process*` function emits one `ResourceDone` per item. Ten files touched, mechanical.

### Buffer

- One append-only store with monotonic `Seq`, capped ring
- `--tui-buffer` sizes it, default 50k lines
- Every record also writes to a run log file unconditionally at debug level, regardless of display level
- Run log: `os.CreateTemp("", "rwr-<timestamp>-*.log")`, mode 0600. `--log-file` overrides. Path printed on exit
- When scrolling past the top of the ring, show a boundary row pointing at the file. Do not silently truncate

Eviction gotcha: views hold `Seq` values, not array indices. Track `oldest` and drop stale head entries lazily on read. Storing positions will panic or scramble output once the ring wraps.

### Views

```go
type View struct {
    Processor string   // "" = all
    idx       []uint64 // Seqs matching, appended as records arrive
    offset    int      // scroll position, remembered
    follow    bool
}
```

Records stored once. Filtering is O(1) at render, not a scan. Each view keeps its own scroll position and follow state.

Level, stdout, and search filters are global, applied on top of whichever view is active.

---

## 6. Model and states

```go
type State int
const (
    Resolving State = iota
    Running
    Suspended  // terminal handed to child
    Prompting  // interactive error halt
    Summary
)

type Model struct {
    plan   *Plan
    procs  []Proc
    cursor int
    live   int
    pinned bool
    store  *Store
    views  map[string]*View
    scope  string
    level  log.Level
    stdout bool
    state  State
    errs   []StepError
}
```

Processor states: `pending`, `running`, `done`, `degraded`, `failed`, `skipped`.

`degraded` covers a processor where some providers succeeded and others failed. Maps to the `warning` token, no theme changes needed.

Processor state derives from the worst lane state. Do not set it directly.

Pinning: selection follows execution until the user presses `j`/`k` or scrolls. First manual input pins. Once pinned, execution continues and the checklist updates but the viewport does not move. Status line shows where live is. `g` unpins and snaps back.

---

## 7. Layout

Five regions, top to bottom:

1. Header: `rwr`, blueprint source, host, failure counters, elapsed clock
2. Strip: one block per processor, colored by state, one line total
3. Collapsed completed processors
4. Active panel with provider lanes and log viewport
5. Pending chips, then help bar

### Collapse rules

- Successful processors compress onto a single shared line: `✓ blueprints · init · repositories · files · services · git  41.2s`
- Failed and degraded processors always keep a full row, with the reason in place of a duration
- Selecting or clicking the compressed line expands it into individual rows
- Pending compresses to one chip row. Expanding grows inline and shrinks the panel. Never an overlay, it would cover live output
- Expansion is a manual toggle, does not auto-collapse when the next processor starts

### Vertical space priority

When height is constrained, sacrifice in this order, last first:

1. Header and strip. Fixed, never sacrificed
2. Failed and degraded rows. Never compressed, never scrolled out
3. Active panel, minimum three log lines
4. Help bar, collapses to `? help`
5. Compressed success line
6. Pending chips

If failures alone exceed available height, that list becomes the scrollable region and the panel shrinks to minimum. Failures win over live output.

### Provider lanes

Inside the active panel, one row per provider with its own glyph, counts, progress bar, and duration. Panel border takes the worst lane state.

Denominators are declared resources, not operations. Glob copies, URL sources, and dependency resolution all expand at execution time. Installing 112 declared packages may touch 400. Show `37/112` and let the log carry the rest. Do not imply the count is total work.

### Compact mode

Triggers below ~60 columns or ~14 rows. Either threshold trips it.

Stays inside the TUI so filters and search survive. A raw stream would lose errors-only, which is what matters most on a small pane.

Renders: one pinned status line (running processor, progress, failure counts), the log stream, one help line. No strip, no lanes, no panel borders, no collapsed list.

Switch live on `WindowSizeMsg`. Both layouts render from the same model, so resizing a tmux pane mid-run promotes or demotes with no state lost.

---

## 8. Frames

### Resolving

Progress list: init file, fetch, blueprint files, templates, providers, resources. First four are stage 1, last two are stage 2.

Hold the frame behind a ~150ms delay. If resolve finishes first, go straight to the main view. A loading screen that appears for three frames is worse than none.

Footer: `nothing has been modified yet`, `q` cancels.

### Running

The five regions above.

### Summary

Replaces the active panel when the run ends.

- Tabs per processor, plus a `failures` tab first and selected by default
- Rows are `Resource` records: glyph, provider, name, detail, duration
- Footer totals: applied, skipped, failed, unknown
- Run log path with `Y` to copy
- `enter` on a row jumps to that resource's lines in the run log. The summary is an index into the log, not a dead end
- `tab` cycles tabs, `/` searches within the current tab

### Dry run

Same frame as Summary. Dry run completes in under a second, so there is no live phase to design.

Differences only:

- `DRY RUN` badge in the header
- Strip blocks render hollow rather than filled
- Status vocabulary: `+` would apply, `=` already present, `?` cannot resolve
- Footer counts: would apply, already present, unresolved
- `c` filters to changes only, hiding already-present rows. Most runs on an existing machine are mostly no-ops

Dry run is not necessarily clean. Unreachable file sources, dead font URLs, missing provider binaries, and unresolvable template variables are all detectable without touching the system. Render as amber `?`, not red. It is a plan problem, not a failure.

### Validate

Summary frame with a diagnostics tab instead of failures, and `Diagnostic` rows instead of `Resource` rows. Same tabs, same search, `enter` opens the offending file and line.

---

## 9. Error handling

| | Interactive (default) | `--interactive=false` |
|---|---|---|
| On processor error | halt, jump viewport to it, errors-only auto-applied | mark row, record, continue |
| Prompt | `r` retry lane, `R` retry processor, `s` skip lane, `q` abort | none |
| Viewport | jumps and unpins | does not move |
| End | resolved or aborted | Summary frame, nonzero exit |

Retry defaults to the failed provider lane. Retrying the whole processor means reinstalling 112 packages to recover from one cargo failure. `R` exists for a deliberate clean redo.

`r`, `R`, and `s` only render in `Prompting`. They cost nothing in the normal help bar.

### Failure persistence

Four surfaces, none of which scroll:

1. Strip block turns red and stays red for the rest of the run
2. Collapsed row keeps its glyph and swaps duration for the error summary
3. Header counter `✗ 2 · ⚠ 1`, always present
4. Panel border takes the worst lane state

---

## 10. Theming

Eleven roles. A theme file defines colors and glyphs together.

| Role | Used for |
|---|---|
| `bg` `text` `subtext` | base surfaces and copy |
| `muted` `dim` | pending, skipped, rules, timestamps |
| `accent` | running, panel border, spinner, progress fill |
| `success` | completed |
| `danger` | failed, error lines |
| `warning` | degraded, warn lines, unknown |
| `info` | info-level markers |
| `stdout` `stderr` | raw command output, distinct from rwr's own logs |

Color by state, not by processor. Coloring 13 processors individually requires more distinguishable accents than most themes expose.

Default theme `rwr`, built from Go brand colors: `#00ADD8` accent, `#00A29C` success, `#CE3262` danger, `#FDDD00` warning, `#5DC9E2` info.

Ship embedded: `rwr`, catppuccin's four flavors (free from `catppuccin/go`), nord, gruvbox, dracula, tokyonight, rose-pine, solarized. User themes as TOML in the config dir, same schema, so a user can copy a built-in and edit it. Matches the existing providers pattern.

Resolution order: `--theme`, `config.yaml`, `$RWR_THEME`, default. `NO_COLOR` overrides all of it.

**Do not paint a background.** Inherit the terminal's and use theme colors for foreground and borders only. Light-theme and transparent-terminal users otherwise get a broken-looking frame.

Degradation via `colorprofile`: truecolor gets hex, 256 quantizes, 16 falls back to ANSI names, no-color drops to glyphs and dim/bold. Test with `NO_COLOR=1` and `TERM=xterm`.

### Glyphs

Part of the theme, not a separate mechanism. ASCII set is a base theme others inherit.

| Role | Unicode | ASCII |
|---|---|---|
| done | `✓` | `+` |
| failed | `✗` | `x` |
| degraded | `⚠` | `!` |
| unknown | `?` | `?` |
| pending | `○` | `.` |
| skipped | `–` | `-` |
| strip filled / empty | `▰` `▱` | `#` `.` |
| progress full / empty | `█` `█` | `#` `-` |
| spinner | braille | `\|` `/` `-` `\` |

`lipgloss.ASCIIBorder()` handles panel frames, `spinner.Line` is the ASCII spinner, progress takes custom full and empty runes.

Detection:

- `LANG` or `LC_ALL` containing UTF-8 implies unicode is safe
- On Windows, `WT_SESSION` present means Windows Terminal, unicode is fine. Absent means conhost, use ASCII
- `--ascii` and `--unicode` force either way

---

## 11. Keybinds

| Key | Action |
|---|---|
| `j` `k` | move selection, sets pinned |
| `g` | unpin, snap to live |
| `a` | toggle scope between selected processor and all |
| `d` | cycle level: info, debug, warn |
| `e` | errors only |
| `o` | toggle stdout/stderr lines |
| `E` | jump to first failure, apply errors-only |
| `/` | search |
| `n` `N` | cycle matches |
| `f` | toggle follow |
| `m` | toggle mouse reporting |
| `y` | yank visible viewport |
| `Y` | copy run log path |
| `tab` | cycle summary tabs |
| `c` | changes only, dry run |
| `?` | full help |
| `q` | quit, or abort at a halt |
| `r` `R` `s` | retry lane, retry processor, skip. Prompting only |

Search scopes to the current view: selected processor plus active level and stdout filters. Matches highlight in place, `esc` clears. Widening scope is what `a` already does.

---

## 12. Mouse

BubbleZone marks regions, `zone.Get(id).InBounds(msg)` in `Update`. Call `zone.Scan` on the final rendered output in top-level `View()`.

| Zone | Click |
|---|---|
| Strip block | select that processor |
| Collapsed done row | select and expand |
| Pending chip row | expand pending list |
| `info` / `live` / `stdout` in panel footer | cycle level, toggle follow, toggle raw output |
| Help bar pills | fire that action |
| Viewport | wheel scrolls, sets pinned |

Strip blocks are 1 cell. Render each as 2 cells plus separator for a 3-cell target.

Hover matters more than click. The strip is 13 unlabeled blocks. Brighten the hovered block and print its name to the right. Without it the strip is unusable by mouse.

Guard re-render on `hoveredZone != lastHovered`. Motion events fire on every cell crossed.

### Text selection

Mouse tracking takes over drag-select. On a log-heavy TUI this is the first thing a user reaches for.

- Most terminals bypass tracking on shift-drag. Document in `?` help
- `m` toggles mouse reporting off entirely
- `y` and `Y` provide a real copy path. Bubble Tea v2 pushes to system clipboard over OSC52, and `atotto/clipboard` is already in the dep tree
- Mouse reporting must come down and go back up around terminal handoff. `tea.ExecProcess` restores terminal state, but verify the mouse specifically

---

## 13. Animation

| What | Driver | Stops |
|---|---|---|
| Progress fill | `progress.SetPercent` spring | settled |
| Spinner | `spinner.Tick` | processor finishes |
| Elapsed | `stopwatch` | run ends |
| Panel height on processor change | harmonica spring + `tea.Tick` | spring settles, then cancel the tick |
| Strip block state flip | 3-frame brightness pulse | immediately |

Kill the panel tick the moment the spring is within a cell of target and snap. Ticking at 60fps indefinitely costs real CPU on a laptop mid-install.

---

## 14. Activation

Primary check: `term.IsTerminal(os.Stdout.Fd())`. `golang.org/x/term` is already a direct dep.

Fall back to `LogReporter` on any of:

- `--no-tui`
- `CI` set. Some runners allocate a pty and would otherwise get escape sequences in a build log
- `TERM=dumb`
- not a TTY

`rwr all > install.log` already fails the TTY check and behaves as today.

### Single-processor runs

`rwr run packages` uses the same TUI with one strip block. Collapsed success list is always empty, pending chips never render, and the panel takes the freed vertical space. Result is a full-height lane view with a large viewport. Summary still works with fewer tabs.

---

## 15. Notifications

One notification, at run completion, in both interactive and non-interactive modes. Nothing else.

- Use OSC 9. Escape sequences travel over SSH, so provisioning a remote box notifies the machine you are sitting at. `notify-send` and `osascript` notify the remote box, which nobody sees. Also dependency-free, where libnotify may be missing on a system rwr is mid-build
- Only fire if the run exceeded ~30s
- Only fire when the terminal is unfocused. Bubble Tea reports focus and blur
- Gate the sequence on a recognized terminal. Unknown OSC sequences are usually swallowed, but some older terminals print the payload as literal text into the log view
- Never when not a TTY
- `--no-notify` disables. No bell by default

Payload carries counts: `rwr finished · 2 failed`.

---

## 16. Validation gating

Every run does stage 1 resolve as a precaution. Gating needs severity or the first person with a slightly odd blueprint cannot run at all.

- Errors gate by default
- Warnings surface as `⚠ n` in the header and do not stop anything
- `--strict` promotes warnings to gating
- `--skip-validate` escape hatch

---

## 17. Suggested sequencing

1. `Plan` struct and stage 1 resolve. Wire `rwr validate` to it. No TUI yet, verify parity with current validate output
2. Reporter interface and `LogReporter`. Wire into `All()` and `commands.go`. Output must be byte-identical to today. This is the safety net
3. Store, views, slog handler. Verify records land with correct processor attribution
4. Stage 2 resolve, provider grouping, lane counts
5. Error accumulation in `All()`
6. Static TUI: header, strip, collapsed list, panel, help bar. No animation, no mouse
7. Terminal handoff via `tea.ExecProcess`. Test sudo prompts hard, this is the common path since interactive defaults true
8. Filters, search, pinning
9. Summary and dry-run frames
10. Theming and glyph fallback
11. Mouse and hover
12. Animation
13. Compact mode and resize
14. Notifications

Steps 1 through 5 are prerequisites and carry most of the risk. Steps 6 onward are additive.

---

## 18. Test checklist

- `rwr all > file.log`, output matches current master byte for byte
- `CI=true rwr all`, no escape sequences
- `NO_COLOR=1`, readable, meaning carried by glyphs
- `TERM=xterm`, 16-color degradation
- `TERM=dumb`, falls back to LogReporter
- Windows conhost, ASCII glyphs
- Windows Terminal, unicode
- sudo prompt in a `files` blueprint, handoff and repaint
- `interactive: true` on one package inside `--interactive=false`, handoff still works
- Ring wrap during a long debug run, no panic, boundary row renders
- Resize below and above both thresholds mid-run
- Processor failure at position 1 and at position 13, both stay visible at end
- Provider partial failure, processor renders degraded not failed
- Dry run against a system where everything is already installed
- Light-background terminal, no painted background
- Terminal with transparency, frame does not fill

---

## 19. Out of scope

- Resource-level granularity in the live view. Live stays at processor and provider. Resources are recorded and rendered only in Summary
- Concurrent processor execution. `All()` is sequential and this design assumes it
- Interactive plan approval before apply. Dry run and validate cover the need
- Scrollbar dragging in the viewport
