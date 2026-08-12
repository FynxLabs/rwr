package processors

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/system"
)

// ProcessScripts executes scripts defined in blueprint data, supporting inline content
// and external files with various interpreters (bash, python, ruby, etc.).
func ProcessScripts(blueprintData []byte, blueprintDir string, format string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var scriptData types.ScriptData
	var err error

	// Unmarshal the blueprint data
	err = helpers.DecodeBlueprintInto(blueprintData, format, types.BlueprintTypeScripts,
		helpers.TreeSchemaVersion(initConfig), &scriptData)
	if err != nil {
		log.Errorf("Error unmarshaling scripts blueprint: %v", err)
		return fmt.Errorf("error unmarshaling scripts blueprint: %w", err)
	}

	log.Debugf("Unmarshaled scripts: %+v", scriptData.Scripts)

	// Process imports and merge imported scripts
	allScripts, err := processScriptImports(scriptData.Scripts, blueprintDir, format, helpers.TreeSchemaVersion(initConfig))
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
	track := newProgress(types.BlueprintTypeScripts)
	track.expect("", len(scripts))
	for _, script := range scripts {
		log.Debugf("Processing script: %+v", script)

		if system.IsDryRun() {
			log.Infof("[DRY-RUN] Would run script: %s (exec: %s)", script.Name, script.Exec)
			track.item("", script.Name, script.Action, types.StatusPlanned, "dry-run", 0)
			continue
		}

		if script.Action == "run" {
			started := time.Now()
			err := runScript(script, osInfo, initConfig, blueprintDir)
			if err != nil {
				log.Errorf("Error running script %s: %v", script.Name, err)
				track.item("", script.Name, script.Action, types.StatusFailed, err.Error(), time.Since(started))
				// Scripts stop at the first failure, where packages, files and
				// the rest record the failure and carry on. That difference is
				// deliberate. A script is arbitrary code, the scripts in a file
				// are usually ordered, and one that fails has very likely left
				// the next one's prerequisites unmet. Returning also hands the
				// error to All(), which is what offers the operator
				// retry/skip/abort - and a retry re-runs the whole file, so
				// stopping early is what keeps the already-succeeded scripts
				// from running a second time. Nothing here can assume the
				// idempotence that makes retry cheap for the other processors.
				return fmt.Errorf("error running script %s: %w", script.Name, err)
			}
			log.Infof("Script %s executed successfully", script.Name)
			track.item("", script.Name, script.Action, types.StatusOK, "", time.Since(started))
		} else {
			log.Errorf("Unsupported action for script %s: %s", script.Name, script.Action)
			track.item("", script.Name, script.Action, types.StatusFailed, "unsupported action", 0)
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
		// Write the script content to a temporary file, named for the executor
		// that will run it. The extension used to be .sh unconditionally, and
		// `powershell -File` refuses any file not ending in .ps1 - so every
		// inline script on Windows (where powershell is the default executor)
		// failed before its first line ran.
		tempFile, err := os.CreateTemp("", fmt.Sprintf("%s-*%s", script.Name, scriptTempExtension(script.Exec)))
		if err != nil {
			return fmt.Errorf("error creating temporary file for script: %v", err)
		}
		defer os.Remove(tempFile.Name()) //nolint:errcheck

		err = os.WriteFile(tempFile.Name(), []byte(script.Content), 0755) // #nosec G306 -- TODO(PR8): create with target mode instead of chmod-after
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
		err := os.Chmod(scriptPath, 0755) // #nosec G302 -- TODO(PR8): create with target mode instead of chmod-after
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

	// Append the script arguments. The blueprint writes `args` either as a
	// string, which types.ScriptArgs splits on whitespace the way it always
	// has, or as a list, whose elements are taken verbatim so an argument
	// containing a space can be expressed at all.
	if len(script.Args) > 0 {
		log.Debugf("Adding script arguments: %v", []string(script.Args))
		scriptCmd.Args = append(scriptCmd.Args, script.Args...)
	}

	// Set the log name
	scriptCmd.LogName = script.Log

	// Set the elevated flag
	scriptCmd.Elevated = script.Elevated

	// buildCommand checks Elevated first, so asking for both silently ignored one
	// of them. Say which one won rather than letting the script run as the wrong
	// account.
	if script.AsUser != "" {
		if script.Elevated {
			log.Warnf("Script %s declares both elevated and asUser: running elevated and ignoring asUser %q", script.Name, script.AsUser)
		} else {
			scriptCmd.AsUser = script.AsUser
		}
	}

	// Set the interactive flag using per-blueprint override or global default
	scriptCmd.Interactive = helpers.ResolveInteractive(script.Interactive, initConfig.Variables.Flags.Interactive)

	log.Debugf("Running script command: %+v", scriptCmd)

	// Run the script
	err := system.RunCommand(scriptCmd, initConfig.Variables.Flags.Debug)
	if err != nil {
		return fmt.Errorf("error running script: %v", err)
	}

	return nil
}

// scriptTempExtension names an inline script's staging file for the executor
// that will run it. Only powershell actually enforces its extension
// (`-File` refuses non-.ps1 files); the rest are for the reader and for any
// tooling that keys off the suffix.
func scriptTempExtension(executor string) string {
	switch executor {
	case "powershell":
		return ".ps1"
	case "python":
		return ".py"
	case "ruby":
		return ".rb"
	case "perl":
		return ".pl"
	case "lua":
		return ".lua"
	default: // bash, /bin/bash, self
		return ".sh"
	}
}

func processScriptImports(items []types.Script, blueprintDir string, format string, treeVersion int) ([]types.Script, error) {
	return helpers.ResolveImports(items, blueprintDir,
		func(item types.Script) string { return item.Import },
		func(data []byte, fileFormat string) ([]types.Script, error) {
			var d types.ScriptData
			if err := helpers.DecodeBlueprintInto(data, fileFormat, types.BlueprintTypeScripts, treeVersion, &d); err != nil {
				return nil, err
			}
			return d.Scripts, nil
		}, format)
}
