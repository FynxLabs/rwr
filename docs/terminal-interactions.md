# Terminal interactions

The dashboard owns the terminal throughout a run. All operator interaction goes
through `internal/reporting`, so processors and credential providers share the
same popup behavior, cancellation handling, and headless fallback.

| Interaction | API |
| --- | --- |
| Choices, text entry, masked credentials, confirmations | `reporting.RunForm(form)` |
| Instructions while an asynchronous task runs, such as OAuth | `reporting.RunFormWithTask(form, task)` |
| Administrator password | `reporting.RequestSecretContext(ctx, prompt)` |
| File overwrite with diff and apply-to-all choices | `reporting.RequestConfirmationAllWithDetails(prompt, details)` |
| External tool with its own prompts, such as MFA | `reporting.RunInteractiveCommand(ctx, command, secrets, private)` |

Forms render inside the dashboard's modal frame and use its colors. Esc or Ctrl-C
cancels a form; a waiting task receives a canceled context and must stop its
network requests. Form submission never quits the parent dashboard.

When dialogs overlap, masked administrator entry takes priority over confirmations
and forms for both rendering and keyboard/paste input. Closing a dialog restores
any waiting interaction underneath it. OAuth completion cannot hide a pending
administrator password prompt.

A successful sudo readiness probe does not guarantee that a later command can
reuse the same authorization: the command can have a different session or popup
TTY. Under the dashboard, that sudo process uses managed askpass instead of
disabling password entry with `-n`. If it needs a password, the masked popup
appears; a valid ticket or password-free policy runs without an extra prompt.
The bridge binds access to the actual child session after startup in both
captured and interactive execution. Supplied command input stays separate from
askpass authentication. Each run starts with fresh readiness state.

External tools run in a private pseudoterminal displayed inside a popup. Their
output, cursor movement, choices, and password echo settings stay inside that
pane. Keyboard input and paste go to the tool. Resizing updates the pane and its
pseudoterminal. Ctrl-C or Ctrl-Esc cancels the tool; Esc is available to the tool's
own menus. Unix cancellation kills the child process group.

Unix commands retain a TTY on stdin when initial input is supplied. Initial
lines are queued with echo disabled before the child or popup input starts;
normal terminal settings are then restored for the child to control. Initial
input must be newline-terminated text, at most 4 KiB, and fit intact in the
terminal's input buffer; other input requires non-interactive execution.
Windows ConPTY commands currently require unsupplied stdin; commands that provide
stdin must use non-interactive execution there.

List known credentials in `secrets` so output is scrubbed across write boundaries.
Set `private` for authentication commands: their output is displayed but never
sent to the run log. Password echo is controlled by the tool's pseudoterminal
settings. RWR's own credential fields always use masked input.

Canceling a choice returns an error, never its default selection. A dismissed
package-manager chooser cannot approve installation of the default manager.

In headless mode these APIs use the existing terminal interaction. During a
dashboard run, `WithTerminal` refuses raw handoffs. New code must not call
standalone form `Run`, `tea.Exec`, `tea.ExecProcess`, or read `os.Stdin` directly.
The architecture regression test guards terminal handoffs and standalone forms;
the approved raw-input paths are headless fallbacks and pre-dashboard CLI setup.

Tests run real Bubble Tea event loops and local tool fixtures. They cover
selection, token validation and masking, OAuth cancellation, a tool requiring a
controlling TTY, private output, redaction, resize, and child cancellation.
