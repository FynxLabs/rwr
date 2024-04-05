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

	// Specify and set the output file path for the configuration file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configFilePath := filepath.Join(homeDir, ".rwr.yaml")
	viper.SetConfigFile(configFilePath)

	// Prompt for GitHub API Token
	fmt.Printf("Enter GitHub API Token (press enter to keep default) [%s]: ", viper.GetString("repository.gh_api_token"))
	ghApiTokenInput, _ := reader.ReadString('\n')
	ghApiTokenInput = strings.TrimSpace(ghApiTokenInput)
	if ghApiTokenInput != "" {
		viper.Set("repository.gh_api_token", ghApiTokenInput)
	}

	// Prompt for Output format
	fmt.Printf("Set the output format (json/yaml/raw) (press enter to keep default) [%s]: ", viper.GetString("rwr.output"))
	outputInput, _ := reader.ReadString('\n')
	outputInput = strings.TrimSpace(outputInput)
	if outputInput != "" {
		viper.Set("rwr.output", outputInput)
	}

	// Prompt for Skip Version Check
	fmt.Printf("Skip version check? (true/false) (press enter to keep default) [%t]: ", viper.GetBool("rwr.skipVersionCheck"))
	skipVersionCheckInput, _ := reader.ReadString('\n')
	skipVersionCheckInput = strings.TrimSpace(skipVersionCheckInput)
	if skipVersionCheckInput != "" {
		viper.Set("rwr.skipVersionCheck", skipVersionCheckInput == "true")
	}

	// Prompt for Highlighting
	fmt.Printf("Enable highlighting? (true/false) (press enter to keep default) [%t]: ", viper.GetBool("rwr.highlight"))
	highlightInput, _ := reader.ReadString('\n')
	highlightInput = strings.TrimSpace(highlightInput)
	if highlightInput != "" {
		viper.Set("rwr.highlight", highlightInput == "true")
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

	// Prompt for Repository Configuration
	fmt.Println("Repository Configuration:")

	// Prompt for Blueprints Local Path
	defaultLocalPath := filepath.Join(homeDir, ".config", "rwr", "blueprints")
	fmt.Printf("Enter the local path for blueprints (press enter to keep default) [%s]: ", defaultLocalPath)
	localPathInput, _ := reader.ReadString('\n')
	localPathInput = strings.TrimSpace(localPathInput)
	if localPathInput == "" {
		localPathInput = defaultLocalPath
	}
	viper.Set("repository.blueprints.localPath", localPathInput)

	// Prompt for Remote Store Type
	fmt.Print("Enter the remote store type (git/s3/local) (press enter to keep default) [local]: ")
	remoteStoreTypeInput, _ := reader.ReadString('\n')
	remoteStoreTypeInput = strings.TrimSpace(remoteStoreTypeInput)
	if remoteStoreTypeInput == "" {
		remoteStoreTypeInput = "local"
	}
	viper.Set("repository.blueprints.remoteStoreType", remoteStoreTypeInput)

	// Prompt for Remote Store URL if Remote Store Type is not local
	if remoteStoreTypeInput != "local" {
		fmt.Print("Enter the remote store URL: ")
		remoteStoreURLInput, _ := reader.ReadString('\n')
		remoteStoreURLInput = strings.TrimSpace(remoteStoreURLInput)
		viper.Set("repository.blueprints.remoteStoreURL", remoteStoreURLInput)
	}

	// Write the configuration to the specified file
	if err := viper.WriteConfig(); err != nil {
		return err
	}

	fmt.Println("Configuration saved to:", configFilePath)
	return nil
}
