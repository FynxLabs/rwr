package helpers

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fynxlabs/rwr/internal/types"

	"github.com/charmbracelet/log"
)

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

// RunCommandOutput runs a command and returns the output as a string.
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

// CommandExists Checks if a command exists in the system.
func CommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func GetBinPath(binName string) (string, error) {
	path, err := exec.LookPath(binName)
	if err != nil {
		return "", err
	}
	return filepath.Clean(path), nil
}
