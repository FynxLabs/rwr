// Package system provides system-level operations and abstractions for rwr.
// It handles command execution, operating system detection, package manager
// discovery, file operations, and cross-platform compatibility. The package
// supports Linux, macOS, and Windows systems, providing utilities for path
// management, embedded file handling, and interactive user prompts.
package system

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/reporting"
	"github.com/fynxlabs/rwr/internal/types"
	"golang.org/x/term"
)

// buildCommand creates an *exec.Cmd from the given types.Command.
//
// Commands are built as argv and handed directly to the kernel - no shell is
// interposed. Every element of cmd.Args reaches the target program as exactly one
// argument, so blueprint-supplied values (package names, paths, comments, password
// hashes) cannot be reinterpreted as shell syntax and arguments containing spaces
// survive intact.
//
// A shell is still available when a provider genuinely wants one; it just has to
// say so explicitly, e.g. exec = "sh" with args = ["-c", "cd /tmp/paru && makepkg"].
//
// Elevation is unchanged in policy - it remains per-command via Elevated/AsUser -
// only the spawn differs. "--" terminates sudo's own option parsing so a command
// whose name begins with a dash cannot be absorbed as a sudo flag. Windows has no
// sudo: Elevated is a no-op there and the process must already be elevated, which
// matches the previous behavior of running through `cmd /C` without any elevation.
func buildCommand(cmd types.Command) *exec.Cmd {
	built := spawn(cmd)
	// Cancelling the run kills the command's whole process group, not just the
	// process rwr spawned: `brew install` is a shell that forks curl and git,
	// and `sudo pacman` is sudo with pacman underneath. Killing only the direct
	// child orphans the real work, which carries on holding the terminal it
	// inherited.
	intoOwnProcessGroup(built)
	built.Cancel = func() error { return terminateProcessGroup(built) }
	// Bound how long a killed command's pipes are held. Without it, a child
	// that leaks a descriptor to a grandchild keeps Wait blocked and the
	// cancellation the operator asked for never completes.
	built.WaitDelay = 5 * time.Second
	return built
}

// spawn builds the *exec.Cmd for a command, before cancellation is wired onto
// it. Every path goes through CommandContext so the run's context can kill it.
func spawn(cmd types.Command) *exec.Cmd {
	ctx := RunContext()
	if runtime.GOOS != "windows" {
		if cmd.Elevated {
			log.Debugf("Running command as sudo - Running Command: %v %v", cmd.Exec, cmd.LogArgs())
			return exec.CommandContext(ctx, "sudo", append([]string{"--", cmd.Exec}, cmd.Args...)...) // #nosec G204 -- argv, not a shell string: args are passed as discrete arguments
		}
		if cmd.AsUser != "" {
			log.Debugf("Running command as user: %v - Running Command: %v %v", cmd.AsUser, cmd.Exec, cmd.LogArgs())
			return exec.CommandContext(ctx, "sudo", append([]string{"-u", cmd.AsUser, "--", cmd.Exec}, cmd.Args...)...) // #nosec G204 -- argv, not a shell string: args are passed as discrete arguments
		}
	} else if cmd.Elevated {
		log.Debugf("Elevated requested on Windows; running in-process (no sudo equivalent): %v %v", cmd.Exec, cmd.LogArgs())
	}

	log.Debugf("Running command: %v %v", cmd.Exec, cmd.LogArgs())
	return exec.CommandContext(ctx, cmd.Exec, cmd.Args...) // #nosec G204 -- argv, not a shell string: args are passed as discrete arguments
}

// setupCommandEnvironment configures the environment variables and PATH for the
// given *exec.Cmd. It copies the current process environment, appends any additional
// variables defined in the types.Command, and enhances the PATH with common binary
// directories.
func setupCommandEnvironment(command *exec.Cmd, cmd types.Command) {
	// Get the current environment variables
	env := os.Environ()

	// Append the additional variables from cmd.Variables
	for key, value := range cmd.Variables {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	// Add common paths to the PATH environment variable
	updatedPath := AddCommonPaths()
	env = append(env, fmt.Sprintf("PATH=%s", updatedPath))

	// Set the environment variables for the command
	command.Env = env
}

// sudoValidatedAt throttles credential validation: sudo's cache lives for
// minutes, so re-checking before every one of hundreds of elevated commands
// would be pure overhead.
var (
	sudoValidateMu  sync.Mutex
	sudoValidatedAt time.Time
)

// ensureSudoCredentials tries to make sure sudo can run without prompting.
// When the cached credentials have expired and a real terminal exists, the
// password prompt runs as its own terminal handover (`sudo -v`), so under
// the TUI the dashboard suspends cleanly instead of painting over the
// prompt. Best effort by design: without a tty (CI), or when validation
// fails (a command-scoped NOPASSWD rule makes `sudo -v` itself want a
// password), the command proceeds and fails or succeeds on its own terms,
// exactly as it did before validation existed.
func ensureSudoCredentials() {
	sudoValidateMu.Lock()
	defer sudoValidateMu.Unlock()

	if time.Since(sudoValidatedAt) < time.Minute {
		return
	}

	// Cheap probe: -n never prompts, it just reports whether the cache holds.
	probe := exec.Command("sudo", "-n", "-v")
	if err := probe.Run(); err == nil {
		sudoValidatedAt = time.Now()
		return
	}

	// No terminal, no prompt: leave it to the command (CI runs are
	// NOPASSWD or root, where sudo never needed a tty in the first place).
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return
	}

	// Cache expired: ask for the password on the real terminal.
	if err := runOnTerminal(exec.Command("sudo", "-v")); err != nil {
		log.Debugf("sudo validation inconclusive (%v); running the command anyway", err)
		return
	}
	sudoValidatedAt = time.Now()
}

// runOnTerminal hands cmd the real terminal via the display layer, falling
// back to direct wiring if the dashboard dies before servicing the request -
// bubbletea drops Sends after its context cancels, and a dropped TerminalReq
// would otherwise block its waiter forever. The claim makes the request
// run-once: losing it means a servicer is executing the command and its
// callback will write done regardless of the program's lifetime, so waiting
// is safe - and Run() is never called twice on the same *exec.Cmd.
func runOnTerminal(command *exec.Cmd) error {
	done := make(chan error, 1)
	claim := &atomic.Bool{}
	reporting.Emit(reporting.TerminalReq{Cmd: command, Done: done, Claim: claim})
	select {
	case err := <-done:
		return err
	case <-reporting.TerminalLost():
		if claim.CompareAndSwap(false, true) {
			// Nobody serviced it; the terminal is free now - wire it
			// directly, exactly as the LogReporter would have.
			if command.Stdin == nil {
				command.Stdin = os.Stdin
			}
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr
			return command.Run()
		}
		return <-done
	}
}

// dryRunMode controls whether commands are actually executed.
// When true, RunCommand and RunCommandOutput log the command but skip execution.
var dryRunMode bool

// SetDryRun enables or disables dry-run mode globally.
func SetDryRun(enabled bool) {
	dryRunMode = enabled
}

// IsDryRun returns whether dry-run mode is enabled.
func IsDryRun() bool {
	return dryRunMode
}

// RunCommand executes a system command with the specified configuration.
// It handles elevated (sudo) execution, running commands as specific users,
// setting environment variables, and configuring input/output streams based
// on the interactive flag and debug mode. Returns an error if the command fails.
// In dry-run mode, it logs the command without executing it.
func RunCommand(cmd types.Command, debug bool) error {
	return current.Run(cmd, debug)
}

// runCommand is the real implementation behind osExecutor.Run.
func runCommand(cmd types.Command, debug bool) error {
	// Refuse before spawning, so a cancelled run stops rather than launching
	// every remaining command only to have each one killed.
	if Cancelled() {
		return ErrCancelled
	}
	if dryRunMode {
		log.Infof("[DRY-RUN] Would execute: %s %s", cmd.Exec, strings.Join(cmd.LogArgs(), " "))
		return nil
	}

	command := buildCommand(cmd)
	setupCommandEnvironment(command, cmd)

	if cmd.Stdin != "" {
		// Supplied input wins over the terminal even for an interactive command:
		// the caller is feeding the tool something specific, and inheriting
		// os.Stdin instead would hang waiting for a human.
		command.Stdin = strings.NewReader(cmd.Stdin)
	}

	if cmd.Interactive {
		// Interactive commands must reach the terminal directly - capturing
		// stderr would swallow sudo's password prompt and hang the run. The
		// display layer owns the terminal, so the handoff goes through the
		// reporter: LogReporter wires os.Std* exactly as this code used to;
		// a TUI suspends itself around the child instead. This path also
		// serves per-item `interactive: true` inside non-interactive runs.
		if err := runOnTerminal(command); err != nil {
			log.Errorf("Error running command: %v (stderr above)", err)
			return err
		}
		return nil
	}

	// A captured elevated command can still make sudo prompt: sudo reads the
	// password from /dev/tty, straight past the captured pipes - under the
	// TUI the prompt and the dashboard then fight over the terminal and the
	// dashboard eats half the password. Validate credentials first, through
	// a proper terminal handover when the cache has expired. Advisory only:
	// a failed validation must not fail the command - `sudo -v` is not
	// command-scoped, so a NOPASSWD rule for just this command makes
	// validation want a password the command itself never needs.
	if (cmd.Elevated || cmd.AsUser != "") && runtime.GOOS != "windows" {
		ensureSudoCredentials()
	}
	{
		// Under the TUI, captured stderr streams into the log view (the `≫`
		// lines); headless it streams to the real stderr like the pre-TUI
		// wiring did. No buffer: the error path points at the stream, and
		// accumulating hundreds of KB of compiler progress just to discard
		// it unread was pure memory overhead.
		if w := reporting.CommandOutputWriter(reporting.SrcStderr); w != nil {
			command.Stderr = w
		} else {
			command.Stderr = os.Stderr
		}
		logFile, err := setOutputStreams(command, debug, cmd.LogName)
		if err != nil {
			return err
		}
		if logFile != nil {
			// Closed after Run, not before it: the command writes to this
			// descriptor while it executes.
			defer func() {
				if cerr := logFile.Close(); cerr != nil {
					log.Errorf("Error closing log file: %v", cerr)
				}
			}()
		}
	}

	if err := command.Run(); err != nil {
		// A command killed by cancellation exits non-zero like any failure.
		// Reporting it as one would fill the summary with "signal: killed"
		// for work the operator deliberately stopped.
		if Cancelled() {
			return ErrCancelled
		}
		// stderr was streamed live (to the log view under the TUI, to the
		// real stderr headless); repeating the blob here renders it twice.
		log.Errorf("Error running command: %v (stderr above)", err)
		return err
	}

	return nil
}

// RunCommandOutput executes a system command and returns its output as a string.
// It handles elevated execution and running as specific users, similar to RunCommand,
// but captures and returns stdout instead of streaming it. Returns the command output
// and an error if the command fails.
func RunCommandOutput(cmd types.Command, debug bool) (string, error) {
	return current.Output(cmd, debug)
}

// runCommandOutput is the real implementation behind osExecutor.Output.
func runCommandOutput(cmd types.Command, debug bool) (string, error) {
	if Cancelled() {
		return "", ErrCancelled
	}
	if dryRunMode {
		log.Infof("[DRY-RUN] Would execute: %s %s", cmd.Exec, strings.Join(cmd.LogArgs(), " "))
		return "", nil
	}

	command := buildCommand(cmd)
	setupCommandEnvironment(command, cmd)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if cmd.Stdin != "" {
		command.Stdin = strings.NewReader(cmd.Stdin)
	}
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	if err != nil {
		if Cancelled() {
			return "", ErrCancelled
		}
		errMsg := fmt.Sprintf("Error running command: %v\nStderr: %s", err, stderr.String())
		log.Error(errMsg)
		return "", err
	}

	return stdout.String(), nil
}

// setOutputStreams points the command's stdout at the terminal (debug) or at the
// blueprint's log file. When a log file is opened it is returned so the caller can
// close it *after* the command has run - closing it here would hand the command a
// dead descriptor and silently discard everything it wrote.
func setOutputStreams(cmd *exec.Cmd, debug bool, logName string) (*os.File, error) {
	log.Debugf("Debug: %v", debug)
	log.Debugf("Log Name: %v", logName)

	// Under the TUI, command stdout is captured into the store so it renders
	// in the log view at debug display level, instead of tearing the frame.
	// Headless (no sink), it streams to the real stdout - the pre-TUI
	// contract is that `rwr all > install.log` carries the package-manager
	// output, and a nil writer here silently threw it away.
	captured := reporting.CommandOutputWriter(reporting.SrcStdout)

	if debug {
		log.Debugf("Debug set, configuring stdout for command: %v", cmd.Path)
		if captured != nil {
			cmd.Stdout = captured
		} else {
			cmd.Stdout = os.Stdout
		}
		return nil, nil
	}

	if logName == "" {
		if captured != nil {
			cmd.Stdout = captured
		} else {
			cmd.Stdout = os.Stdout
		}
		return nil, nil
	}

	// The log path comes from the blueprint (`log:` on a script), so it is
	// attacker-influenced whenever the blueprint is. Appending through a symlink
	// would let command output be written into ~/.ssh/authorized_keys or ~/.bashrc
	// without the blueprint containing anything that looks like it writes a file.
	// Refuse to follow a symlink, and create at 0600 - command output routinely
	// contains more than the operator expects.
	if info, err := os.Lstat(logName); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("refusing to write command log through symlink %q", logName)
	}

	file, err := os.OpenFile(logName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600) // #nosec G304 -- path is operator-supplied blueprint input; symlinks refused above
	if err != nil {
		log.Errorf("Error opening log file: %v", err)
		return nil, fmt.Errorf("error opening log file %q: %w", logName, err)
	}

	if captured != nil {
		cmd.Stdout = io.MultiWriter(file, captured)
	} else {
		cmd.Stdout = file
	}
	return file, nil
}

// CommandExists checks if a command exists in the system's PATH.
// It returns true if the command is found, false otherwise.
func CommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// GetBinPath returns the absolute path to the specified binary.
// It searches for the binary in the system's PATH and returns a cleaned
// absolute path. Returns an error if the binary is not found.
func GetBinPath(binName string) (string, error) {
	path, err := exec.LookPath(binName)
	if err != nil {
		return "", err
	}
	return filepath.Clean(path), nil
}
