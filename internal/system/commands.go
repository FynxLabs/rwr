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

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/types"
)

// RunCommand executes a system command with the specified configuration.
// It handles elevated (sudo) execution, running commands as specific users,
// setting environment variables, and configuring input/output streams based
// on the interactive flag and debug mode. Returns an error if the command fails.
func RunCommand(cmd types.Command, debug bool) error {
	var command *exec.Cmd

	if cmd.Elevated {
		if runtime.GOOS == "windows" {
			log.Debugf("Running command as elevated - Running Command: %v %v", cmd.Exec, cmd.Args)
			command = exec.Command("cmd", "/C", fmt.Sprintf("%s %s", cmd.Exec, strings.Join(cmd.Args, " ")))
		} else {
			log.Debugf("Running command as sudo - Running Command: %v %v", cmd.Exec, cmd.Args)
			command = exec.Command("sudo", "sh", "-c", fmt.Sprintf("%s %s", cmd.Exec, strings.Join(cmd.Args, " ")))
		}
	} else if cmd.AsUser != "" {
		log.Debugf("Running command as user: %v - Running Command: %v %v", cmd.AsUser, cmd.Exec, cmd.Args)
		command = exec.Command("sudo", "-u", cmd.AsUser, "sh", "-c", fmt.Sprintf("%s %s", cmd.Exec, strings.Join(cmd.Args, " ")))
	} else {
		log.Debugf("Running command: %v %v", cmd.Exec, cmd.Args)
		command = exec.Command("sh", "-c", fmt.Sprintf("%s %s", cmd.Exec, strings.Join(cmd.Args, " ")))
	}

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

	var stderr bytes.Buffer
	command.Stderr = &stderr

	if cmd.Interactive {
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
	} else {
		setOutputStreams(command, debug, cmd.LogName)
	}

	err := command.Run()
	if err != nil {
		errMsg := fmt.Sprintf("Error running command: %v\nStderr: %s", err, stderr.String())
		log.Error(errMsg)
		return err
	}

	return nil
}

// RunCommandOutput executes a system command and returns its output as a string.
// It handles elevated execution and running as specific users, similar to RunCommand,
// but captures and returns stdout instead of streaming it. Returns the command output
// and an error if the command fails.
func RunCommandOutput(cmd types.Command, debug bool) (string, error) {
	var command *exec.Cmd

	if cmd.Elevated {
		if runtime.GOOS == "windows" {
			log.Debugf("Running command as elevated - Running Command: %v %v", cmd.Exec, cmd.Args)
			command = exec.Command("cmd", "/C", fmt.Sprintf("%s %s", cmd.Exec, strings.Join(cmd.Args, " ")))
		} else {
			log.Debugf("Running command as sudo - Running Command: %v %v", cmd.Exec, cmd.Args)
			command = exec.Command("sudo", "sh", "-c", fmt.Sprintf("%s %s", cmd.Exec, strings.Join(cmd.Args, " ")))
		}
	} else if cmd.AsUser != "" {
		log.Debugf("Running command as user: %v - Running Command: %v %v", cmd.AsUser, cmd.Exec, cmd.Args)
		command = exec.Command("sudo", "-u", cmd.AsUser, "sh", "-c", fmt.Sprintf("%s %s", cmd.Exec, strings.Join(cmd.Args, " ")))
	} else {
		log.Debugf("Running command: %v %v", cmd.Exec, cmd.Args)
		command = exec.Command("sh", "-c", fmt.Sprintf("%s %s", cmd.Exec, strings.Join(cmd.Args, " ")))
	}

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

	var stdout bytes.Buffer
	var stderr bytes.Buffer
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

// setOutputStreams sets the standard output stream for a command based on the log level and log name.
func setOutputStreams(cmd *exec.Cmd, debug bool, logName string) {
	log.Debugf("Debug: %v", debug)
	log.Debugf("Log Name: %v", logName)
	if debug {
		log.Debugf("Debug set, configuring stdout for command: %v", cmd.Path)
		cmd.Stdout = os.Stdout
	} else if logName != "" {
		file, err := os.OpenFile(logName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Errorf("Error opening log file: %v", err)
			return
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				log.Errorf("Error closing log file: %v", err)
			}
		}(file)

		cmd.Stdout = file
	}
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
