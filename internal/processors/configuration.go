package processors

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
)

func ProcessConfiguration(blueprintData []byte, blueprintDir string, format string, initConfig *types.InitConfig) error {
	var configData types.ConfigData

	err := helpers.UnmarshalBlueprint(blueprintData, format, &configData)
	if err != nil {
		return fmt.Errorf("error unmarshaling configuration blueprint: %w", err)
	}

	for _, config := range configData.Configurations {
		var err error
		switch config.Tool {
		case "dconf":
			err = processDconf(blueprintDir, config, initConfig)
		case "gsettings":
			err = processGSettings(config, initConfig)
		case "macos_defaults":
			err = processMacOSDefaults(config, initConfig)
		case "windows_registry":
			err = processWindowsRegistry(config, initConfig)
		default:
			err = fmt.Errorf("unsupported configuration tool: %s", config.Tool)
		}

		if err != nil {
			log.Errorf("Error processing configuration %s: %v", config.Name, err)
			return fmt.Errorf("error processing configuration %s: %w", config.Name, err)
		}
	}

	return nil
}

func processDconf(blueprintDir string, config types.Configuration, initConfig *types.InitConfig) error {
	log.Debugf("Processing Dconf file: %s", config.File)

	// Resolve the file path relative to the blueprint directory
	file := filepath.Join(blueprintDir, config.File)

	log.Debugf("Dconf file set for path: %s", file)

	boostrapfileName := "configuration_" + config.Name + "_bootstrap"

	bootstrapFile := filepath.Join(initConfig.Variables.Flags.RunOnceLocation, boostrapfileName)

	if config.RunOnce {
		log.Debugf("RunOnce Set: Checking for %s to see if already ran", bootstrapFile)
		if _, err := os.Stat(bootstrapFile); err == nil {
			log.Infof("Dconf configuration already applied, skipping")
			return nil
		}
	}

	cmd := types.Command{
		Exec:     "dconf",
		Args:     []string{"load", "/"},
		Elevated: config.Elevated,
	}

	// Using a pipe to provide input
	cmd.Variables = map[string]string{
		"DCONF_INPUT": file,
	}
	cmd.Args = append(cmd.Args, fmt.Sprintf("< $DCONF_INPUT"))

	err := helpers.RunCommand(cmd, initConfig.Variables.Flags.Debug)

	if err != nil {
		return fmt.Errorf("error applying dconf configuration: %w", err)
	}

	if config.RunOnce {
		log.Debugf("RunOnce Set: Write Bootstrap File %s", bootstrapFile)
		if err := os.WriteFile(bootstrapFile, []byte{}, 0644); err != nil {
			log.Warnf("Failed to create dconf bootstrap file: %v", err)
		}
	}

	return nil
}

func processGSettings(config types.Configuration, initConfig *types.InitConfig) error {
	args := []string{"set"}
	if config.Path != "" {
		args = append(args, fmt.Sprintf("%s:%s", config.Schema, config.Path))
	} else {
		args = append(args, config.Schema)
	}

	cvalue := fmt.Sprintf("%v", config.Value)
	args = append(args, config.Key, cvalue)

	cmd := types.Command{
		Exec:     "gsettings",
		Args:     args,
		Elevated: config.Elevated,
	}

	err := helpers.RunCommand(cmd, initConfig.Variables.Flags.Debug)

	if err != nil {
		return fmt.Errorf("error applying gsettings configuration: %w", err)
	}

	return nil
}

func processMacOSDefaults(config types.Configuration, initConfig *types.InitConfig) error {
	args := []string{"write"}
	if config.Domain != "" {
		args = append(args, config.Domain)
	} else {
		args = append(args, "NSGlobalDomain")
	}
	args = append(args, config.Key, fmt.Sprintf("-%s", config.Kind), fmt.Sprintf("%v", config.Value))

	cmd := types.Command{
		Exec:     "defaults",
		Args:     args,
		Elevated: config.Elevated,
	}

	err := helpers.RunCommand(cmd, initConfig.Variables.Flags.Debug)

	if err != nil {
		return fmt.Errorf("error applying macOS defaults configuration: %w", err)
	}

	return nil
}

func processWindowsRegistry(config types.Configuration, initConfig *types.InitConfig) error {
	var psCommand string
	switch strings.ToLower(config.Type) {
	case "string":
		psCommand = fmt.Sprintf("Set-ItemProperty -Path 'HKLM:\\%s' -Name '%s' -Value '%s' -Type String", config.Path, config.Key, config.Value)
	case "expandstring":
		psCommand = fmt.Sprintf("Set-ItemProperty -Path 'HKLM:\\%s' -Name '%s' -Value '%s' -Type ExpandString", config.Path, config.Key, config.Value)
	case "binary":
		// For binary, we'll need to convert the []byte to a comma-separated string
		byteSlice, ok := config.Value.([]byte)
		if !ok {
			return fmt.Errorf("invalid binary value for registry key")
		}
		byteString := strings.Trim(strings.Join(strings.Fields(fmt.Sprintf("%d", byteSlice)), ","), "[]")
		psCommand = fmt.Sprintf("Set-ItemProperty -Path 'HKLM:\\%s' -Name '%s' -Value ([byte[]]@(%s)) -Type Binary", config.Path, config.Key, byteString)
	case "dword":
		psCommand = fmt.Sprintf("Set-ItemProperty -Path 'HKLM:\\%s' -Name '%s' -Value %d -Type DWord", config.Path, config.Key, config.Value)
	case "qword":
		psCommand = fmt.Sprintf("Set-ItemProperty -Path 'HKLM:\\%s' -Name '%s' -Value %d -Type QWord", config.Path, config.Key, config.Value)
	default:
		return fmt.Errorf("unsupported registry value type: %s", config.Type)
	}

	cmd := types.Command{
		Exec:     "powershell",
		Args:     []string{"-Command", psCommand},
		Elevated: config.Elevated,
	}

	if config.Elevated {
		// For elevated privileges, we need to run PowerShell as administrator
		cmd.Args = []string{"-Command", "Start-Process", "powershell", "-Verb", "RunAs", "-ArgumentList", fmt.Sprintf("-Command %s", psCommand)}
	}

	err := helpers.RunCommand(cmd, initConfig.Variables.Flags.Debug)

	if err != nil {
		return fmt.Errorf("error applying Windows registry configuration: %w", err)
	}

	return nil
}
