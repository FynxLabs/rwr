// Package system provides system-level operations and abstractions for rwr.
// It handles command execution, operating system detection, package manager
// discovery, file operations, and cross-platform compatibility. The package
// supports Linux, macOS, and Windows systems, providing utilities for path
// management, embedded file handling, and interactive user prompts.
package system

import (
	"bytes"
	"context"
	"errors"
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
	configureCommandCancellation(built)
	return built
}

// buildAuthenticatedCommand binds a freshly-read password to the exact sudo
// process that runs the requested command. Some sudo policies do not make a
// ticket created by a separate `sudo -v` available to the later process.
func buildAuthenticatedCommand(cmd types.Command) *exec.Cmd {
	ctx := RunContext()
	var args []string
	if cmd.Elevated {
		args = append([]string{"-S", "-p", "", "--", cmd.Exec}, cmd.Args...)
	} else {
		args = append([]string{"-S", "-p", "", "-u", cmd.AsUser, "--", cmd.Exec}, cmd.Args...)
	}
	built := exec.CommandContext(ctx, "sudo", args...) // #nosec G204 -- fixed sudo flags followed by discrete command argv
	configureCommandCancellation(built)
	return built
}

func configureCommandCancellation(built *exec.Cmd) {
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
}

// spawn builds the *exec.Cmd for a command, before cancellation is wired onto
// it. Every path goes through CommandContext so the run's context can kill it.
func spawn(cmd types.Command) *exec.Cmd {
	ctx := RunContext()
	if runtime.GOOS != "windows" {
		if cmd.Elevated {
			log.Debugf("Running command as sudo - Running Command: %v %v", cmd.Exec, cmd.LogArgs())
			return exec.CommandContext(ctx, "sudo", append([]string{"-n", "--", cmd.Exec}, cmd.Args...)...) // #nosec G204 -- argv, not a shell string: args are passed as discrete arguments
		}
		if cmd.AsUser != "" {
			log.Debugf("Running command as user: %v - Running Command: %v %v", cmd.AsUser, cmd.Exec, cmd.LogArgs())
			return exec.CommandContext(ctx, "sudo", append([]string{"-n", "-u", cmd.AsUser, "--", cmd.Exec}, cmd.Args...)...) // #nosec G204 -- argv, not a shell string: args are passed as discrete arguments
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

// The latest probe result is cached briefly whether it was warm or cold. A
// failed policy lookup is no more likely to change between adjacent packages
// than a successful one, and retrying a ten-second timeout for every package
// turns the bound into an unbounded run-level delay. A cached cold result still
// sends every potentially escalating command to the terminal, so a password
// prompt made by the command itself remains visible.
var (
	sudoValidateMu sync.Mutex
	sudoProbedAt   time.Time
	sudoProbeWarm  bool
	// sudoWarmFromPrompt distinguishes a ticket RWR created for a
	// self-escalating command from an ambient ticket proven by `sudo -n -v`.
	// Some macOS sudo policies let the original command use the former but do
	// not let a later, separately spawned sudo process reuse it.
	sudoWarmFromPrompt bool
	// Retained only for this run so separately-scoped sudo processes do not
	// ask for the same password repeatedly. Callers receive disposable copies.
	sudoRunPassword []byte
)

// ErrSudoAuthentication reports that a command requiring sudo could not be
// authenticated through the controlled masked prompt.
var ErrSudoAuthentication = errors.New("sudo authentication failed")

var errNoSudoTerminal = errors.New("sudo credentials are not cached and no terminal is available")

// Escalates is the case the elevation flags alone miss: a command rwr runs
// unprivileged that calls sudo itself. brew refuses to run as root, so
// Elevated is false and validation never ran, and then a cask install shelled
// out to sudo, prompted on /dev/tty behind the dashboard, and hung the run
// with no visible prompt.
func mayPromptForSudo(cmd types.Command) bool {
	if runtime.GOOS == "windows" {
		return false
	}
	return cmd.Elevated || cmd.AsUser != "" || cmd.Escalates
}

// sudoCredentialsCached reports whether sudo would run without asking.
//
// It never prompts. A cold result is handled separately by the controlled
// masked authentication path, and only for a command known to need or invoke
// sudo. Homebrew formulae are classified before this point and never reach it.
//
// sudoProbeArgs is the probe rwr runs. -n is the whole contract: it makes sudo
// answer from the credential cache or fail, and never prompt. Losing it would
// turn the probe back into the thing this replaced.
var sudoProbeArgs = []string{"sudo", "-n", "-v"}

// sudoProbeTimeout bounds the probe.
//
// -n means sudo will not prompt, but that is not the same as sudo returning.
// A policy lookup can block underneath it - PAM against a directory server,
// NSS resolving a group over the network - and the probe holds
// sudoValidateMu while it runs, so a stall there is not one slow command but
// every command after it, waiting on a mutex that is never released.
const sudoProbeTimeout = 10 * time.Second

// sudoProbe runs the probe. Its context is rooted in the run context so ctrl-c
// interrupts a blocked PAM/NSS lookup immediately instead of waiting for the
// probe deadline. A var lets tests substitute a probe without touching the
// host's sudo policy or credential state.
var sudoProbe = func(ctx context.Context) error {
	return exec.CommandContext(ctx, sudoProbeArgs[0], sudoProbeArgs[1:]...).Run() // #nosec G204 -- fixed argv, not input
}

func sudoValidationCommand(ctx context.Context, input []byte) *exec.Cmd {
	command := exec.CommandContext(ctx, "sudo", "-S", "-p", "", "-v") // #nosec G204 -- fixed argv
	command.Stdin = bytes.NewReader(input)
	command.Stdout = io.Discard
	command.Stderr = os.Stderr
	return command
}

var sudoValidate = func(ctx context.Context, input []byte) error {
	return sudoValidationCommand(ctx, input).Run()
}

var sudoPassword = func() ([]byte, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, errNoSudoTerminal
	}
	var password []byte
	err := reporting.WithTerminal(func() error {
		_, _ = fmt.Fprint(os.Stderr, "sudo password: ")
		var err error
		password, err = term.ReadPassword(int(os.Stdin.Fd()))
		_, _ = fmt.Fprintln(os.Stderr)
		if err != nil {
			return fmt.Errorf("reading sudo password: %w", err)
		}
		return nil
	})
	return password, err
}

// SetSudoProbeForTest substitutes the probe and returns the restore func.
func SetSudoProbeForTest(probe func(context.Context) error) (restore func()) {
	previous := sudoProbe
	sudoProbe = probe
	return func() { sudoProbe = previous }
}

func SetSudoPasswordForTest(prompt func() ([]byte, error)) (restore func()) {
	previous := sudoPassword
	sudoPassword = prompt
	return func() { sudoPassword = previous }
}

func SetSudoValidateForTest(validate func(context.Context, []byte) error) (restore func()) {
	previous := sudoValidate
	sudoValidate = validate
	return func() { sudoValidate = previous }
}

// resetSudoThrottleForTest clears the probe throttle so a test observes the
// probe rather than a cached answer from an earlier one.
func resetSudoThrottleForTest() {
	sudoValidateMu.Lock()
	defer sudoValidateMu.Unlock()
	sudoProbedAt = time.Time{}
	sudoProbeWarm = false
	sudoWarmFromPrompt = false
	zeroBytes(sudoRunPassword)
	sudoRunPassword = nil
}

func cacheSudoPassword(password []byte) {
	sudoValidateMu.Lock()
	defer sudoValidateMu.Unlock()
	zeroBytes(sudoRunPassword)
	sudoRunPassword = append([]byte(nil), password...)
}

func cachedSudoPassword() []byte {
	sudoValidateMu.Lock()
	defer sudoValidateMu.Unlock()
	return append([]byte(nil), sudoRunPassword...)
}

func clearSudoPassword() {
	sudoValidateMu.Lock()
	defer sudoValidateMu.Unlock()
	zeroBytes(sudoRunPassword)
	sudoRunPassword = nil
}

func sudoCredentialsCached() bool {
	sudoValidateMu.Lock()
	defer sudoValidateMu.Unlock()

	if time.Since(sudoProbedAt) < time.Minute {
		return sudoProbeWarm
	}

	ctx, cancel := context.WithTimeout(RunContext(), sudoProbeTimeout)
	defer cancel()
	sudoProbeWarm = sudoProbe(ctx) == nil
	sudoWarmFromPrompt = false
	sudoProbedAt = time.Now()
	return sudoProbeWarm
}

func sudoCredentialsReusableByManagedCommand() bool {
	if !sudoCredentialsCached() {
		return false
	}
	sudoValidateMu.Lock()
	defer sudoValidateMu.Unlock()
	return !sudoWarmFromPrompt
}

func ensureSudoCredentials() error {
	if sudoCredentialsCached() {
		return nil
	}
	password, err := sudoPassword()
	if err != nil {
		return err
	}
	input := passwordInput(password, "")
	zeroBytes(password)
	defer zeroBytes(input)
	if err := sudoValidate(RunContext(), input); err != nil {
		return err
	}
	cacheSudoPassword(input[:len(input)-1])

	markSudoWarm(true)
	return nil
}

func markSudoWarm(fromPrompt bool) {
	sudoValidateMu.Lock()
	sudoProbeWarm = true
	sudoWarmFromPrompt = fromPrompt
	sudoProbedAt = time.Now()
	sudoValidateMu.Unlock()
}

func passwordInput(password []byte, stdin string) []byte {
	input := make([]byte, 0, len(password)+1+len(stdin))
	input = append(input, password...)
	input = append(input, '\n')
	input = append(input, stdin...)
	return input
}

func zeroBytes(value []byte) {
	for i := range value {
		value[i] = 0
	}
}

func commandForRun(cmd types.Command) (*exec.Cmd, []byte, error) {
	if runtime.GOOS == "windows" || (!cmd.Elevated && cmd.AsUser == "") || sudoCredentialsReusableByManagedCommand() {
		return buildCommand(cmd), nil, nil
	}
	password := cachedSudoPassword()
	if len(password) == 0 {
		var err error
		password, err = sudoPassword()
		if err != nil {
			if errors.Is(err, errNoSudoTerminal) {
				return buildCommand(cmd), nil, nil
			}
			return nil, nil, fmt.Errorf("%w: %v", ErrSudoAuthentication, err)
		}
		cacheSudoPassword(password)
	}
	input := passwordInput(password, cmd.Stdin)
	zeroBytes(password)
	return buildAuthenticatedCommand(cmd), input, nil
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

	command, sudoInput, err := commandForRun(cmd)
	if err != nil {
		return err
	}
	defer zeroBytes(sudoInput)
	setupCommandEnvironment(command, cmd)

	if sudoInput != nil {
		command.Stdin = bytes.NewReader(sudoInput)
		if cmd.Interactive {
			command.Stdin = io.MultiReader(command.Stdin, os.Stdin)
		}
	} else if cmd.Stdin != "" {
		// Supplied input wins over the terminal even for an interactive command:
		// the caller is feeding the tool something specific, and inheriting
		// os.Stdin instead would hang waiting for a human.
		command.Stdin = strings.NewReader(cmd.Stdin)
	}
	if cmd.Escalates && !cmd.Elevated && cmd.AsUser == "" {
		if err := ensureSudoCredentials(); err != nil {
			if Cancelled() {
				return ErrCancelled
			}
			if !errors.Is(err, errNoSudoTerminal) {
				return fmt.Errorf("%w: %v", ErrSudoAuthentication, err)
			}
		}
	}

	if cmd.Interactive {
		// Interactive commands must reach the terminal directly - capturing
		// stderr would swallow sudo's password prompt and hang the run. The
		// display layer owns the terminal, so the handoff goes through the
		// reporter: LogReporter wires os.Std* exactly as this code used to;
		// a TUI suspends itself around the child instead. This path also
		// serves per-item `interactive: true` inside non-interactive runs.
		if err := runOnTerminal(command); err != nil {
			if sudoInput != nil {
				clearSudoPassword()
			}
			// An interactive command reaches the terminal directly, so a
			// cancelled one comes back as its kill status rather than through
			// the context. Without this it reports "signal: killed" and gets
			// logged as an error, which is the same contract break the
			// captured path guards against below.
			if Cancelled() {
				return ErrCancelled
			}
			log.Errorf("Error running command: %v (stderr above)", err)
			return err
		}
		if sudoInput != nil {
			markSudoWarm(true)
		}
		return nil
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
		if sudoInput != nil {
			clearSudoPassword()
		}
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
	if sudoInput != nil {
		markSudoWarm(true)
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
	if cmd.Escalates && !cmd.Elevated && cmd.AsUser == "" {
		if err := ensureSudoCredentials(); err != nil {
			if Cancelled() {
				return "", ErrCancelled
			}
			if !errors.Is(err, errNoSudoTerminal) {
				return "", fmt.Errorf("%w: %v", ErrSudoAuthentication, err)
			}
		}
	}

	command, sudoInput, err := commandForRun(cmd)
	if err != nil {
		return "", err
	}
	defer zeroBytes(sudoInput)
	setupCommandEnvironment(command, cmd)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if sudoInput != nil {
		command.Stdin = bytes.NewReader(sudoInput)
	} else if cmd.Stdin != "" {
		command.Stdin = strings.NewReader(cmd.Stdin)
	}
	command.Stdout = &stdout
	command.Stderr = &stderr

	err = command.Run()
	if err != nil {
		if sudoInput != nil {
			clearSudoPassword()
		}
		if Cancelled() {
			return "", ErrCancelled
		}
		errMsg := fmt.Sprintf("Error running command: %v\nStderr: %s", err, stderr.String())
		log.Error(errMsg)
		return "", err
	}
	if sudoInput != nil {
		markSudoWarm(true)
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
