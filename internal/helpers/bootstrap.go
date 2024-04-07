package helpers

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

func IsBootstrapped() bool {
	// Get the configuration directory from viper
	configDir := viper.GetString("rwr.configdir")
	if configDir == "" {
		// If not set, use the default path
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Errorf("Error getting user home directory: %v", err)
			return false
		}
		configDir = filepath.Join(homeDir, ".config", "rwr")
	}

	// Check if bootstrap file exists
	bootstrapFile := filepath.Join(configDir, "bootstrap")
	if _, err := os.Stat(bootstrapFile); os.IsNotExist(err) {
		return false
	}
	return true
}

func Bootstrap() error {
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

	// Create the bootstrap file
	bootstrapFile := filepath.Join(configDir, "bootstrap")
	file, err := os.Create(bootstrapFile)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Errorf("Error closing file: %v", err)
		}
	}(file)
	return nil
}
