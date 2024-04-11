package helpers

import (
	"os"
	"os/exec"
	"runtime"

	"github.com/charmbracelet/log"
)

func RunWithElevatedPrivileges(command string, logName string, args ...string) error {
	if runtime.GOOS == "windows" {
		// Prepend "runas" to the command and arguments
		runasArgs := append([]string{"/c", command}, args...)
		cmd := exec.Command("cmd", runasArgs...)

		// Set the standard output and error streams
		if logName != "" {
			file, err := os.OpenFile(logName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Errorf("Error opening log file: %v", err)
				return err
			}
			defer file.Close()

			cmd.Stdout = file
			cmd.Stderr = file
		} else {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}

		// Run the command with runas
		err := cmd.Run()
		if err != nil {
			log.Errorf("Error running command with runas: %v", err)
			return err
		}
	} else {
		// Prepend "sudo" to the command and arguments
		sudoArgs := append([]string{command}, args...)
		cmd := exec.Command("sudo", sudoArgs...)

		// Set the standard output and error streams
		if logName != "" {
			file, err := os.OpenFile(logName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Errorf("Error opening log file: %v", err)
				return err
			}
			defer file.Close()

			cmd.Stdout = file
			cmd.Stderr = file
		} else {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}

		// Run the command with sudo
		err := cmd.Run()
		if err != nil {
			log.Errorf("Error running command with sudo: %v", err)
			return err
		}
	}

	return nil
}

func RunCommand(command string, logName string, args ...string) error {
	cmd := exec.Command(command, args...)

	// Set the standard output and error streams
	if logName != "" {
		file, err := os.OpenFile(logName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Errorf("Error opening log file: %v", err)
			return err
		}
		defer file.Close()

		cmd.Stdout = file
		cmd.Stderr = file
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	// Run the command
	err := cmd.Run()
	if err != nil {
		log.Errorf("Error running command: %v", err)
		return err
	}

	return nil
}

// CommandExists Checks if a command exists in the system.
func CommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
