package processors

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
)

func ProcessConfiguration(blueprintData []byte, format string, initConfig *types.InitConfig) error {

	var configData types.ConfigData

	err := helpers.UnmarshalBlueprint(blueprintData, format, &configData)
	if err != nil {
		return fmt.Errorf("error unmarshaling configuration blueprint: %w", err)
	}

	for _, config := range configData.Configurations {
		switch config.Tool {
		case "dconf":
			err = processDconf(config, initConfig)
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
			return err
		}
	}

	return nil
}

func processDconf(config types.Configuration, initConfig *types.InitConfig) error {
	dconf, ok := config.Options["dconf"].(types.DconfConfiguration)
	if !ok {
		return fmt.Errorf("invalid dconf configuration")
	}

	if dconf.RunOnce {
		configDir := viper.GetString("rwr.configdir")
		bootstrapFile := filepath.Join(configDir, "dconf_bootstrap")
		if _, err := os.Stat(bootstrapFile); err == nil {
			log.Infof("Dconf configuration already applied, skipping")
			return nil
		}
	}

	cmd := exec.Command("dconf", "load", "/")
	cmd.Stdin, _ = os.Open(dconf.File)

	if dconf.Elevated {
		cmd = exec.Command("sudo", append([]string{"-S"}, cmd.Args...)...)
	}

	err := helpers.RunCommand(types.Command{
		Exec:     cmd.Path,
		Args:     cmd.Args[1:],
		Elevated: dconf.Elevated,
	}, initConfig.Variables.Flags.Debug)

	if err != nil {
		return fmt.Errorf("error applying dconf configuration: %w", err)
	}

	if dconf.RunOnce {
		configDir := viper.GetString("rwr.configdir")
		bootstrapFile := filepath.Join(configDir, "dconf_bootstrap")
		if err := os.WriteFile(bootstrapFile, []byte{}, 0644); err != nil {
			log.Warnf("Failed to create dconf bootstrap file: %v", err)
		}
	}

	return nil
}

func processGSettings(config types.Configuration, initConfig *types.InitConfig) error {
	gsettings, ok := config.Options["gsettings"].(types.GSettingsConfiguration)
	if !ok {
		return fmt.Errorf("invalid gsettings configuration")
	}

	args := []string{"set"}
	if gsettings.Path != "" {
		args = append(args, fmt.Sprintf("%s:%s", gsettings.Schema, gsettings.Path))
	} else {
		args = append(args, gsettings.Schema)
	}
	args = append(args, gsettings.Key, gsettings.Value)

	cmd := exec.Command("gsettings", args...)

	if gsettings.Elevated {
		cmd = exec.Command("sudo", append([]string{"-S"}, cmd.Args...)...)
	}

	err := helpers.RunCommand(types.Command{
		Exec:     cmd.Path,
		Args:     cmd.Args[1:],
		Elevated: gsettings.Elevated,
	}, initConfig.Variables.Flags.Debug)

	if err != nil {
		return fmt.Errorf("error applying gsettings configuration: %w", err)
	}

	return nil
}

func processMacOSDefaults(config types.Configuration, initConfig *types.InitConfig) error {
	defaults, ok := config.Options["macos_defaults"].(types.MacOSDefaultsConfiguration)
	if !ok {
		return fmt.Errorf("invalid macOS defaults configuration")
	}

	args := []string{"write"}
	if defaults.Domain != "" {
		args = append(args, defaults.Domain)
	} else {
		args = append(args, "NSGlobalDomain")
	}
	args = append(args, defaults.Key, fmt.Sprintf("-%s", defaults.Kind), fmt.Sprintf("%v", defaults.Value))

	cmd := exec.Command("defaults", args...)

	if defaults.Elevated {
		cmd = exec.Command("sudo", append([]string{"-S"}, cmd.Args...)...)
	}

	err := helpers.RunCommand(types.Command{
		Exec:     cmd.Path,
		Args:     cmd.Args[1:],
		Elevated: defaults.Elevated,
	}, initConfig.Variables.Flags.Debug)

	if err != nil {
		return fmt.Errorf("error applying macOS defaults configuration: %w", err)
	}

	return nil
}

func processWindowsRegistry(config types.Configuration, initConfig *types.InitConfig) error {
	regConfig, ok := config.Options["windows_registry"].(types.WindowsRegistryConfiguration)
	if !ok {
		return fmt.Errorf("invalid Windows registry configuration")
	}

	var psCommand string
	switch strings.ToLower(regConfig.Type) {
	case "string":
		psCommand = fmt.Sprintf("Set-ItemProperty -Path 'HKLM:\\%s' -Name '%s' -Value '%s' -Type String", regConfig.Path, regConfig.Key, regConfig.Value)
	case "expandstring":
		psCommand = fmt.Sprintf("Set-ItemProperty -Path 'HKLM:\\%s' -Name '%s' -Value '%s' -Type ExpandString", regConfig.Path, regConfig.Key, regConfig.Value)
	case "binary":
		// For binary, we'll need to convert the []byte to a comma-separated string
		byteSlice, ok := regConfig.Value.([]byte)
		if !ok {
			return fmt.Errorf("invalid binary value for registry key")
		}
		byteString := strings.Trim(strings.Join(strings.Fields(fmt.Sprintf("%d", byteSlice)), ","), "[]")
		psCommand = fmt.Sprintf("Set-ItemProperty -Path 'HKLM:\\%s' -Name '%s' -Value ([byte[]]@(%s)) -Type Binary", regConfig.Path, regConfig.Key, byteString)
	case "dword":
		psCommand = fmt.Sprintf("Set-ItemProperty -Path 'HKLM:\\%s' -Name '%s' -Value %d -Type DWord", regConfig.Path, regConfig.Key, regConfig.Value)
	case "qword":
		psCommand = fmt.Sprintf("Set-ItemProperty -Path 'HKLM:\\%s' -Name '%s' -Value %d -Type QWord", regConfig.Path, regConfig.Key, regConfig.Value)
	default:
		return fmt.Errorf("unsupported registry value type: %s", regConfig.Type)
	}

	args := []string{"-Command", psCommand}

	cmd := exec.Command("powershell", args...)

	if regConfig.Elevated {
		// For elevated privileges, we need to run PowerShell as administrator
		// This might require additional setup or prompt the user for elevation
		cmd = exec.Command("powershell", append([]string{"-Command", "Start-Process", "powershell", "-Verb", "RunAs", "-ArgumentList"}, args...)...)
	}

	err := helpers.RunCommand(types.Command{
		Exec:     cmd.Path,
		Args:     cmd.Args[1:],
		Elevated: regConfig.Elevated,
	}, initConfig.Variables.Flags.Debug)

	if err != nil {
		return fmt.Errorf("error applying Windows registry configuration: %w", err)
	}

	return nil
}
