package processors

import (
	"fmt"
	"github.com/thefynx/rwr/internal/types"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/helpers"
)

func ProcessScriptsFromFile(blueprintFile string, blueprintDir string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
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
	err = ProcessScripts(scripts, osInfo, initConfig)
	if err != nil {
		log.Errorf("Error processing scripts: %v", err)
		return fmt.Errorf("error processing scripts: %w", err)
	}

	return nil
}

func ProcessScriptsFromData(blueprintData []byte, blueprintDir string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var scripts []types.Script

	// Unmarshal the resolved blueprint data
	err := helpers.UnmarshalBlueprint(blueprintData, initConfig.Init.Format, &scripts)
	if err != nil {
		log.Errorf("Error unmarshaling scripts blueprint data: %v", err)
		return fmt.Errorf("error unmarshaling scripts blueprint data: %w", err)
	}

	// Process the scripts
	err = ProcessScripts(scripts, osInfo, initConfig)
	if err != nil {
		log.Errorf("Error processing scripts: %v", err)
		return fmt.Errorf("error processing scripts: %w", err)
	}

	return nil
}

func ProcessScripts(scripts []types.Script, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	for _, script := range scripts {
		if script.Action == "run" {
			err := runScript(script, osInfo, initConfig)
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

func runScript(script types.Script, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var scriptCmd types.Command

	// Determine the script executor based on the "exec" field
	switch script.Exec {
	case "self":
		scriptCmd = types.Command{
			Exec: script.Source,
			Args: []string{},
		}
	case "bash", "/bin/bash":
		scriptCmd = types.Command{
			Exec: osInfo.Tools.Bash.Bin,
			Args: []string{script.Source},
		}
	case "python":
		scriptCmd = types.Command{
			Exec: osInfo.Tools.Python.Bin,
			Args: []string{script.Source},
		}
	case "ruby":
		scriptCmd = types.Command{
			Exec: osInfo.Tools.Ruby.Bin,
			Args: []string{script.Source},
		}
	case "perl":
		scriptCmd = types.Command{
			Exec: osInfo.Tools.Perl.Bin,
			Args: []string{script.Source},
		}
	case "lua":
		scriptCmd = types.Command{
			Exec: osInfo.Tools.Lua.Bin,
			Args: []string{script.Source},
		}
	case "powershell":
		scriptCmd = types.Command{
			Exec: osInfo.Tools.PowerShell.Bin,
			Args: []string{"-File", script.Source},
		}
	default:
		return fmt.Errorf("unsupported script executor: %s", script.Exec)
	}

	// Append the script arguments
	if script.Args != "" {
		scriptCmd.Args = append(scriptCmd.Args, script.Args)
	}

	// Set the log name
	scriptCmd.LogName = script.Log

	// Set the elevated flag
	scriptCmd.Elevated = script.Elevated

	// Run the script
	err := helpers.RunCommand(scriptCmd, initConfig.Variables.Flags.Debug)
	if err != nil {
		return fmt.Errorf("error running script: %v", err)
	}

	return nil
}
