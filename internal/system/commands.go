// Package system provides system-level operations and abstractions for rwr.
// It handles command execution, operating system detection, package manager
// discovery, file operations, and cross-platform compatibility. The package
// supports Linux, macOS, and Windows systems, providing utilities for path
// management, embedded file handling, and interactive user prompts.
package system

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/reporting"
	"github.com/fynxlabs/rwr/internal/types"
)

// buildCommand creates an *exec.Cmd from the given types.Command.
//
// Commands are built as argv and handed directly to the kernel — no shell is
// interposed. Every element of cmd.Args reaches the target program as exactly one
// argument, so blueprint-supplied values (package names, paths, comments, password
// hashes) cannot be reinterpreted as shell syntax and arguments containing spaces
// survive intact.
//
// A shell is still available when a provider genuinely wants one; it just has to
// say so explicitly, e.g. exec = "sh" with args = ["-c", "cd /tmp/paru && makepkg"].
//
// Elevation is unchanged in policy — it remains per-command via Elevated/AsUser —
// only the spawn differs. "--" terminates sudo's own option parsing so a command
// whose name begins with a dash cannot be absorbed as a sudo flag. Windows has no
// sudo: Elevated is a no-op there and the process must already be elevated, which
// matches the previous behavior of running through `cmd /C` without any elevation.
func buildCommand(cmd types.Command) *exec.Cmd {
	if runtime.GOOS != "windows" {
		if cmd.Elevated {
			log.Debugf("Running command as sudo - Running Command: %v %v", cmd.Exec, cmd.LogArgs())
			return exec.Command("sudo", append([]string{"--", cmd.Exec}, cmd.Args...)...) // #nosec G204 -- argv, not a shell string: args are passed as discrete arguments
		}
		if cmd.AsUser != "" {
			log.Debugf("Running command as user: %v - Running Command: %v %v", cmd.AsUser, cmd.Exec, cmd.LogArgs())
			return exec.Command("sudo", append([]string{"-u", cmd.AsUser, "--", cmd.Exec}, cmd.Args...)...) // #nosec G204 -- argv, not a shell string: args are passed as discrete arguments
		}
	} else if cmd.Elevated {
		log.Debugf("Elevated requested on Windows; running in-process (no sudo equivalent): %v %v", cmd.Exec, cmd.LogArgs())
	}

	log.Debugf("Running command: %v %v", cmd.Exec, cmd.LogArgs())
	return exec.Command(cmd.Exec, cmd.Args...) // #nosec G204 -- argv, not a shell string: args are passed as discrete arguments
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
	if dryRunMode {
		log.Infof("[DRY-RUN] Would execute: %s %s", cmd.Exec, strings.Join(cmd.LogArgs(), " "))
		return nil
	}

	command := buildCommand(cmd)
	setupCommandEnvironment(command, cmd)

	var stderr bytes.Buffer

	if cmd.Stdin != "" {
		// Supplied input wins over the terminal even for an interactive command:
		// the caller is feeding the tool something specific, and inheriting
		// os.Stdin instead would hang waiting for a human.
		command.Stdin = strings.NewReader(cmd.Stdin)
	}

	if cmd.Interactive {
		// Interactive commands must reach the terminal directly — capturing
		// stderr would swallow sudo's password prompt and hang the run. The
		// display layer owns the terminal, so the handoff goes through the
		// reporter: LogReporter wires os.Std* exactly as this code used to;
		// a TUI suspends itself around the child instead. This path also
		// serves per-item `interactive: true` inside non-interactive runs.
		done := make(chan error, 1)
		reporting.Emit(reporting.TerminalReq{Cmd: command, Done: done})
		if err := <-done; err != nil {
			log.Errorf("Error running command: %v\nStderr: %s", err, stderr.String())
			return err
		}
		return nil
	}
	{
		command.Stderr = &stderr
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
		log.Errorf("Error running command: %v\nStderr: %s", err, stderr.String())
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
		errMsg := fmt.Sprintf("Error running command: %v\nStderr: %s", err, stderr.String())
		log.Error(errMsg)
		return "", err
	}

	return stdout.String(), nil
}

// setOutputStreams points the command's stdout at the terminal (debug) or at the
// blueprint's log file. When a log file is opened it is returned so the caller can
// close it *after* the command has run — closing it here would hand the command a
// dead descriptor and silently discard everything it wrote.
func setOutputStreams(cmd *exec.Cmd, debug bool, logName string) (*os.File, error) {
	log.Debugf("Debug: %v", debug)
	log.Debugf("Log Name: %v", logName)

	if debug {
		log.Debugf("Debug set, configuring stdout for command: %v", cmd.Path)
		cmd.Stdout = os.Stdout
		return nil, nil
	}

	if logName == "" {
		return nil, nil
	}

	// The log path comes from the blueprint (`log:` on a script), so it is
	// attacker-influenced whenever the blueprint is. Appending through a symlink
	// would let command output be written into ~/.ssh/authorized_keys or ~/.bashrc
	// without the blueprint containing anything that looks like it writes a file.
	// Refuse to follow a symlink, and create at 0600 — command output routinely
	// contains more than the operator expects.
	if info, err := os.Lstat(logName); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("refusing to write command log through symlink %q", logName)
	}

	file, err := os.OpenFile(logName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600) // #nosec G304 -- path is operator-supplied blueprint input; symlinks refused above
	if err != nil {
		log.Errorf("Error opening log file: %v", err)
		return nil, fmt.Errorf("error opening log file %q: %w", logName, err)
	}

	cmd.Stdout = file
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
