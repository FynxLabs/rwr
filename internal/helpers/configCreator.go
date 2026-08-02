package helpers

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
)

// CreateDefaultConfig interactively prompts the user to configure rwr settings
// such as GitHub token, SSH key, and package managers, then writes the config file.
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
	if err := EnsureConfigDir(configDir); err != nil {
		return err
	}

	// Set the configuration file path
	configFilePath := filepath.Join(configDir, "config.yaml")
	viper.SetConfigFile(configFilePath)

	// Prompt for GitHub API Token.
	//
	// The stored value is redacted rather than shown as the default: this prompt
	// used to print the live PAT and the Base64 private key to the terminal, where
	// they stayed in scrollback and in any recorded session.
	fmt.Printf("Enter GitHub API Token (press enter to keep current) [%s]: ", types.Redact(viper.GetString("repository.gh_api_token")))
	ghApiTokenInput, _ := reader.ReadString('\n') //nolint:errcheck
	ghApiTokenInput = strings.TrimSpace(ghApiTokenInput)
	if ghApiTokenInput != "" {
		viper.Set("repository.gh_api_token", ghApiTokenInput)
	}

	// Prompt for SSH Private Key
	fmt.Printf("Enter SSH Private Key (file path or Base64 encoded) (press enter to keep current) [%s]: ", types.Redact(viper.GetString("repository.ssh_private_key")))
	sshPrivateKeyInput, _ := reader.ReadString('\n') //nolint:errcheck
	sshPrivateKeyInput = strings.TrimSpace(sshPrivateKeyInput)
	if sshPrivateKeyInput != "" {
		viper.Set("repository.ssh_private_key", sshPrivateKeyInput)
	}

	// Prompt for Skip Version Check
	fmt.Printf("Skip version check? (true/false) (press enter to keep default) [%t]: ", viper.GetBool("rwr.skipVersionCheck"))
	skipVersionCheckInput, _ := reader.ReadString('\n') //nolint:errcheck
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
	logLevelInput, _ := reader.ReadString('\n') //nolint:errcheck
	logLevelInput = strings.TrimSpace(logLevelInput)
	if logLevelInput != "" {
		viper.Set("log.level", logLevelInput)
	} else {
		viper.Set("log.level", defaultLogLevel) // Set to default if no input is provided
	}

	// Prompt for Repository Configuration
	fmt.Println("Repository Configuration:")

	// Prompt for Init File Location
	fmt.Printf("Enter the location of the init file (local or url) (press enter to keep default) [%s]: ", viper.GetString("repository.init-file"))
	initFileLocationInput, _ := reader.ReadString('\n') //nolint:errcheck
	initFileLocationInput = strings.TrimSpace(initFileLocationInput)
	if initFileLocationInput != "" {
		viper.Set("repository.init-file", initFileLocationInput)
	}

	// The file holds a GitHub token and an SSH private key, and viper writes
	// at 0644 with no way to ask for less. Pre-creating it at 0600 means the
	// credentials are never on disk world-readable, not even briefly.
	if err := PrecreateSecureConfigFile(configFilePath); err != nil {
		return err
	}

	// Write the configuration to the specified file
	if err := viper.WriteConfig(); err != nil {
		return err
	}

	// Belt and braces: verify nothing widened it.
	if err := SecureConfigFile(configFilePath); err != nil {
		return err
	}

	fmt.Println("Configuration saved to:", configFilePath)
	return nil
}
