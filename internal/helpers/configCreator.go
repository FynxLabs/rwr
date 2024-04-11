package helpers

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

func CreateDefaultConfig() error {
	reader := bufio.NewReader(os.Stdin)

	// Get the configuration directory from viper
	configDir := viper.GetString("rwr.configdir")
	if configDir == "" {
		// If not set, use the default path
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configDir = filepath.Join(homeDir, ".config", "rwr")
	}

	// Create the configuration directory if it doesn't exist
	err := os.MkdirAll(configDir, os.ModePerm)
	if err != nil {
		return err
	}

	// Set the configuration file path
	configFilePath := filepath.Join(configDir, "config.yaml")
	viper.SetConfigFile(configFilePath)

	// Prompt for GitHub API Token
	fmt.Printf("Enter GitHub API Token (press enter to keep default) [%s]: ", viper.GetString("repository.gh_api_token"))
	ghApiTokenInput, _ := reader.ReadString('\n')
	ghApiTokenInput = strings.TrimSpace(ghApiTokenInput)
	if ghApiTokenInput != "" {
		viper.Set("repository.gh_api_token", ghApiTokenInput)
	}

	// Prompt for SSH Private Key
	fmt.Printf("Enter SSH Private Key Base64 Encoded (press enter to keep default) [%s]: ", viper.GetString("repository.ssh_private_key"))
	sshPrivateKeyInput, _ := reader.ReadString('\n')
	sshPrivateKeyInput = strings.TrimSpace(sshPrivateKeyInput)
	if sshPrivateKeyInput != "" {
		viper.Set("repository.ssh_private_key", sshPrivateKeyInput)
	}

	// Prompt for Skip Version Check
	fmt.Printf("Skip version check? (true/false) (press enter to keep default) [%t]: ", viper.GetBool("rwr.skipVersionCheck"))
	skipVersionCheckInput, _ := reader.ReadString('\n')
	skipVersionCheckInput = strings.TrimSpace(skipVersionCheckInput)
	if skipVersionCheckInput != "" {
		viper.Set("rwr.skipVersionCheck", skipVersionCheckInput == "true")
	}

	// Prompt for Log Level
	defaultLogLevel := viper.GetString("log.level")
	if defaultLogLevel == "" {
		defaultLogLevel = "info" // Assuming "info" as a safe default log level
	}
	fmt.Printf("Enter Log Level (debug, info, warn, error) (press enter to keep default) [%s]: ", defaultLogLevel)
	logLevelInput, _ := reader.ReadString('\n')
	logLevelInput = strings.TrimSpace(logLevelInput)
	if logLevelInput != "" {
		viper.Set("log.level", logLevelInput)
	} else {
		viper.Set("log.level", defaultLogLevel) // Set to default if no input is provided
	}

	// Prompt for Default Package Manager on Linux
	fmt.Printf("Set the default package manager for Linux (press enter to keep default) [%s]: ", viper.GetString("packageManager.linux.default"))
	linuxDefaultPMInput, _ := reader.ReadString('\n')
	linuxDefaultPMInput = strings.TrimSpace(linuxDefaultPMInput)
	if linuxDefaultPMInput != "" {
		viper.Set("packageManager.linux.default", linuxDefaultPMInput)
	}

	// Prompt for Default Package Manager on macOS
	fmt.Printf("Set the default package manager for macOS (press enter to keep default) [%s]: ", viper.GetString("packageManager.macos.default"))
	macOSDefaultPMInput, _ := reader.ReadString('\n')
	macOSDefaultPMInput = strings.TrimSpace(macOSDefaultPMInput)
	if macOSDefaultPMInput != "" {
		viper.Set("packageManager.macos.default", macOSDefaultPMInput)
	}

	// Prompt for Default Package Manager on Windows
	fmt.Printf("Set the default package manager for Windows (press enter to keep default) [%s]: ", viper.GetString("packageManager.windows.default"))
	windowsDefaultPMInput, _ := reader.ReadString('\n')
	windowsDefaultPMInput = strings.TrimSpace(windowsDefaultPMInput)
	if windowsDefaultPMInput != "" {
		viper.Set("packageManager.windows.default", windowsDefaultPMInput)
	}

	// Prompt for Init Templates Enabled
	fmt.Printf("Enable templates for the init file? (true/false) (press enter to keep default) [%t]: ", viper.GetBool("rwr.initTemplatesEnabled"))
	initTemplatesEnabledInput, _ := reader.ReadString('\n')
	initTemplatesEnabledInput = strings.TrimSpace(initTemplatesEnabledInput)
	if initTemplatesEnabledInput != "" {
		viper.Set("rwr.initTemplatesEnabled", initTemplatesEnabledInput == "true")
	}

	// Prompt for Repository Configuration
	fmt.Println("Repository Configuration:")

	// Prompt for Init File Location
	fmt.Printf("Enter the location of the init file (local or url) (press enter to keep default) [%s]: ", viper.GetString("repository.init-file"))
	initFileLocationInput, _ := reader.ReadString('\n')
	initFileLocationInput = strings.TrimSpace(initFileLocationInput)
	if initFileLocationInput != "" {
		viper.Set("repository.init-file", initFileLocationInput)
	}

	// Write the configuration to the specified file
	if err := viper.WriteConfig(); err != nil {
		return err
	}

	fmt.Println("Configuration saved to:", configFilePath)
	return nil
}
