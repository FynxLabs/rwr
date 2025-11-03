package processors

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/system"
)

func ProcessScripts(blueprintData []byte, blueprintDir string, format string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var scriptData types.ScriptData
	var err error

	// Unmarshal the blueprint data
	err = helpers.UnmarshalBlueprint(blueprintData, format, &scriptData)
	if err != nil {
		log.Errorf("Error unmarshaling scripts blueprint: %v", err)
		return fmt.Errorf("error unmarshaling scripts blueprint: %w", err)
	}

	log.Debugf("Unmarshaled scripts: %+v", scriptData.Scripts)

	// Process imports and merge imported scripts
	allScripts, err := processScriptImports(scriptData.Scripts, blueprintDir, format)
	if err != nil {
		return fmt.Errorf("error processing script imports: %w", err)
	}
	scriptData.Scripts = allScripts

	// Filter scripts based on active profiles
	filteredScripts := helpers.FilterByProfiles(scriptData.Scripts, initConfig.Variables.Flags.Profiles)

	log.Debugf("Filtering scripts: %d total, %d matching active profiles %v",
		len(scriptData.Scripts), len(filteredScripts), initConfig.Variables.Flags.Profiles)

	// Process the filtered scripts
	err = processScripts(filteredScripts, osInfo, initConfig, blueprintDir)
	if err != nil {
		log.Errorf("Error processing scripts: %v", err)
		return fmt.Errorf("error processing scripts: %w", err)
	}

	return nil
}

func processScripts(scripts []types.Script, osInfo *types.OSInfo, initConfig *types.InitConfig, blueprintDir string) error {
	for _, script := range scripts {
		log.Debugf("Processing script: %+v", script)

		if script.Action == "run" {
			err := runScript(script, osInfo, initConfig, blueprintDir)
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

func runScript(script types.Script, osInfo *types.OSInfo, initConfig *types.InitConfig, blueprintDir string) error {
	var scriptCmd types.Command

	log.Debugf("Running script: %s", script.Name)

	// Set default executor if not specified
	if script.Exec == "" {
		switch osInfo.System.OS {
		case "linux", "darwin":
			script.Exec = "bash"
		case "windows":
			script.Exec = "powershell"
		default:
			return fmt.Errorf("unsupported OS for default script executor: %s", osInfo.System.OS)
		}
	}

	// Determine the script source (from file or content)
	var scriptPath string
	if script.Source != "" {
		scriptPath = filepath.Join(blueprintDir, script.Source, script.Name)
	} else if script.Content != "" {
		// Write the script content to a temporary file
		tempFile, err := os.CreateTemp("", fmt.Sprintf("%s-*.sh", script.Name))
		if err != nil {
			return fmt.Errorf("error creating temporary file for script: %v", err)
		}
		defer os.Remove(tempFile.Name())

		err = os.WriteFile(tempFile.Name(), []byte(script.Content), 0755)
		if err != nil {
			return fmt.Errorf("error writing script content to temporary file: %v", err)
		}

		scriptPath = tempFile.Name()
	} else {
		return fmt.Errorf("either source or content must be provided for script %s", script.Name)
	}

	// Determine the script executor based on the "exec" field
	switch script.Exec {
	case "self":
		log.Debugf("Using 'self' executor for script: %s", script.Name)
		// Make the script executable
		err := os.Chmod(scriptPath, 0755)
		if err != nil {
			return fmt.Errorf("error setting script as executable: %v", err)
		}
		scriptCmd = types.Command{
			Exec: scriptPath,
			Args: []string{},
		}
	case "bash", "/bin/bash":
		log.Debugf("Using 'bash' executor for script: %s", script.Name)
		scriptCmd = types.Command{
			Exec: osInfo.Tools.Bash.Bin,
			Args: []string{scriptPath},
		}
	case "python":
		log.Debugf("Using 'python' executor for script: %s", script.Name)
		scriptCmd = types.Command{
			Exec: osInfo.Tools.Python.Bin,
			Args: []string{scriptPath},
		}
	case "ruby":
		log.Debugf("Using 'ruby' executor for script: %s", script.Name)
		scriptCmd = types.Command{
			Exec: osInfo.Tools.Ruby.Bin,
			Args: []string{scriptPath},
		}
	case "perl":
		log.Debugf("Using 'perl' executor for script: %s", script.Name)
		scriptCmd = types.Command{
			Exec: osInfo.Tools.Perl.Bin,
			Args: []string{scriptPath},
		}
	case "lua":
		log.Debugf("Using 'lua' executor for script: %s", script.Name)
		scriptCmd = types.Command{
			Exec: osInfo.Tools.Lua.Bin,
			Args: []string{scriptPath},
		}
	case "powershell":
		log.Debugf("Using 'powershell' executor for script: %s", script.Name)
		scriptCmd = types.Command{
			Exec: osInfo.Tools.PowerShell.Bin,
			Args: []string{"-File", scriptPath},
		}
	default:
		return fmt.Errorf("unsupported script executor: %s", script.Exec)
	}

	// Append the script arguments
	if script.Args != "" {
		log.Debugf("Adding script arguments: %s", script.Args)
		scriptCmd.Args = append(scriptCmd.Args, script.Args)
	}

	// Set the log name
	scriptCmd.LogName = script.Log

	// Set the elevated flag
	scriptCmd.Elevated = script.Elevated

	log.Debugf("Running script command: %+v", scriptCmd)

	// Run the script
	err := system.RunCommand(scriptCmd, initConfig.Variables.Flags.Debug)
	if err != nil {
		return fmt.Errorf("error running script: %v", err)
	}

	return nil
}

func processScriptImports(scripts []types.Script, blueprintDir string, format string) ([]types.Script, error) {
	allScripts := make([]types.Script, 0)
	visited := make(map[string]bool)

	for _, script := range scripts {
		if script.Import != "" {
			log.Debugf("Processing script import: %s", script.Import)

			importPath := filepath.Join(blueprintDir, script.Import)
			absPath, err := filepath.Abs(importPath)
			if err != nil {
				return nil, fmt.Errorf("error resolving import path %s: %w", importPath, err)
			}

			if visited[absPath] {
				log.Warnf("Circular import detected, skipping: %s", absPath)
				continue
			}
			visited[absPath] = true

			importData, err := os.ReadFile(importPath)
			if err != nil {
				return nil, fmt.Errorf("error reading import file %s: %w", importPath, err)
			}

			fileFormat := format
			if fileFormat == "" {
				ext := filepath.Ext(importPath)
				fileFormat = ext
			}

			var importedScriptData types.ScriptData
			if err := helpers.UnmarshalBlueprint(importData, fileFormat, &importedScriptData); err != nil {
				return nil, fmt.Errorf("error unmarshaling import file %s: %w", importPath, err)
			}

			allScripts = append(allScripts, importedScriptData.Scripts...)
			log.Debugf("Imported %d scripts from %s", len(importedScriptData.Scripts), script.Import)
		} else {
			allScripts = append(allScripts, script)
		}
	}

	return allScripts, nil
}
