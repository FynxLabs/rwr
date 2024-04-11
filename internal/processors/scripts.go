package processors

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/helpers"
	"github.com/thefynx/rwr/internal/processors/types"
)

func ProcessScriptsFromFile(blueprintFile string, osInfo types.OSInfo) error {
	var scripts []types.Script

	// Read the blueprint file
	blueprintData, err := os.ReadFile(blueprintFile)
	if err != nil {
		log.Errorf("Error reading blueprint file: %v", err)
		return fmt.Errorf("error reading blueprint file: %w", err)
	}

	// Unmarshal the blueprint data
	err = helpers.UnmarshalBlueprint(blueprintData, filepath.Ext(blueprintFile), &scripts)
	if err != nil {
		log.Errorf("Error unmarshaling scripts blueprint: %v", err)
		return fmt.Errorf("error unmarshaling scripts blueprint: %w", err)
	}

	// Process the scripts
	err = ProcessScripts(scripts, osInfo)
	if err != nil {
		log.Errorf("Error processing scripts: %v", err)
		return fmt.Errorf("error processing scripts: %w", err)
	}

	return nil
}

func ProcessScriptsFromData(blueprintData []byte, initConfig *types.InitConfig, osInfo types.OSInfo) error {
	var scripts []types.Script

	// Unmarshal the resolved blueprint data
	err := helpers.UnmarshalBlueprint(blueprintData, initConfig.Init.Format, &scripts)
	if err != nil {
		log.Errorf("Error unmarshaling scripts blueprint data: %v", err)
		return fmt.Errorf("error unmarshaling scripts blueprint data: %w", err)
	}

	// Process the scripts
	err = ProcessScripts(scripts, osInfo)
	if err != nil {
		log.Errorf("Error processing scripts: %v", err)
		return fmt.Errorf("error processing scripts: %w", err)
	}

	return nil
}

func ProcessScripts(scripts []types.Script, osInfo types.OSInfo) error {
	for _, script := range scripts {
		if script.Action == "run" {
			err := runScript(script, osInfo)
			if err != nil {
				log.Errorf("Error running script %s: %v", script.Name, err)
				return fmt.Errorf("error running script %s: %w", script.Name, err)
			}
			log.Infof("Script %s executed successfully", script.Name)
		} else {
			log.Errorf("Unsupported action for script %s: %s", script.Name, script.Action)
			return fmt.Errorf("unsupported action for script %s: %s", script.Name, script.Action)
		}
	}
	return nil
}

func runScript(script types.Script, osInfo types.OSInfo) error {
	var cmdArgs []string

	// Determine the script executor based on the "exec" field
	var executor string
	switch script.Exec {
	case "self":
		executor = script.Source
	case "bash", "/bin/bash":
		executor = osInfo.Tools.Bash.Bin
		cmdArgs = append(cmdArgs, script.Source)
	case "python":
		executor = osInfo.Tools.Python.Bin
		cmdArgs = append(cmdArgs, script.Source)
	case "ruby":
		executor = osInfo.Tools.Ruby.Bin
		cmdArgs = append(cmdArgs, script.Source)
	case "perl":
		executor = osInfo.Tools.Perl.Bin
		cmdArgs = append(cmdArgs, script.Source)
	case "lua":
		executor = osInfo.Tools.Lua.Bin
		cmdArgs = append(cmdArgs, script.Source)
	case "powershell":
		executor = osInfo.Tools.PowerShell.Bin
		cmdArgs = append(cmdArgs, "-File", script.Source)
	default:
		return fmt.Errorf("unsupported script executor: %s", script.Exec)
	}

	// Append the script arguments
	if script.Args != "" {
		cmdArgs = append(cmdArgs, script.Args)
	}

	// Run the script
	if script.Elevated {
		err := helpers.RunWithElevatedPrivileges(executor, script.Log, cmdArgs...)
		if err != nil {
			return fmt.Errorf("error running script with elevated privileges: %v", err)
		}
	} else {
		err := helpers.RunCommand(executor, script.Log, cmdArgs...)
		if err != nil {
			return fmt.Errorf("error running script: %v", err)
		}
	}

	return nil
}
