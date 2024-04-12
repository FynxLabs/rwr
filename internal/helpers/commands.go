package helpers

import (
	"github.com/spf13/viper"
	"os"
	"os/exec"
	"runtime"

	"github.com/charmbracelet/log"
)

func RunWithElevatedPrivileges(command string, logName string, args ...string) error {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		// Prepend "runas" to the command and arguments
		runasArgs := append([]string{"/c", command}, args...)
		cmd := exec.Command("cmd", runasArgs...)

		// Set the standard output and error streams
		setOutputStreams(cmd, logName)
	} else {
		// Prepend "sudo" to the command and arguments
		sudoArgs := append([]string{command}, args...)
		cmd := exec.Command("sudo", sudoArgs...)

		// Set the standard output and error streams
		setOutputStreams(cmd, logName)
	}

	// Run the command
	err := cmd.Run()
	if err != nil {
		log.Errorf("Error running command: %v", err)
		return err
	}

	return nil
}

func RunCommand(command string, logName string, args ...string) error {
	cmd := exec.Command(command, args...)

	// Set the standard output and error streams
	setOutputStreams(cmd, logName)

	// Run the command
	err := cmd.Run()
	if err != nil {
		log.Errorf("Error running command: %v", err)
		return err
	}

	return nil
}

// setOutputStreams sets the standard output and error streams for a command based on the log level and log name.
func setOutputStreams(cmd *exec.Cmd, logName string) {
	if lvl, err := log.ParseLevel(viper.GetString("log.level")); err == nil && lvl == log.DebugLevel {
		log.Debugf("Debug set, configuring output streams for command: %v", cmd.Path)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
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
		cmd.Stderr = file
	}
}

// CommandExists Checks if a command exists in the system.
func CommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
