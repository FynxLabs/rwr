package processors

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
	"github.com/thefynx/rwr/internal/helpers"
	"github.com/thefynx/rwr/internal/processors/types"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

func Initialize(initFilePath string) (*types.InitConfig, error) {
	var initConfig types.InitConfig
	var err error
	var fileExt string

	// Check if the init file path is a URL
	if strings.HasPrefix(initFilePath, "http://") || strings.HasPrefix(initFilePath, "https://") {
		// Extract the repository URL and file path from the GitHub URL
		if strings.Contains(initFilePath, "/blob/") {
			parts := strings.Split(initFilePath, "/blob/")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid GitHub URL format")
			}
			repoURL := parts[0]
			filePath := parts[1]

			// Determine the file extension based on the file path
			fileExt = filepath.Ext(filePath)

			// Download the init file from GitHub
			err = helpers.HandleGitFileDownload(types.GitOptions{
				URL:    repoURL + "/blob/" + filePath,
				Target: "init" + fileExt, // Save the file with the original extension
			})
			if err != nil {
				return nil, fmt.Errorf("error downloading init file from GitHub: %w", err)
			}

			initFilePath = "init" + fileExt // Update the init file path to the downloaded file
		} else {
			// Determine the file extension based on the URL
			fileExt = filepath.Ext(initFilePath)

			// Download the raw init file
			err = helpers.DownloadFile(initFilePath, "init"+fileExt)
			if err != nil {
				return nil, fmt.Errorf("error downloading init file: %w", err)
			}

			initFilePath = "init" + fileExt // Update the init file path to the downloaded file
		}
	} else {
		// Check if the init file exists at the specified path
		if _, err := os.Stat(initFilePath); os.IsNotExist(err) {
			return nil, fmt.Errorf("init file not found at path: %s", initFilePath)
		}
		fileExt = filepath.Ext(initFilePath)
	}

	viper.SetConfigFile(initFilePath)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading init file: %w", err)
	}

	if err := viper.Unmarshal(&initConfig); err != nil {
		return nil, fmt.Errorf("error unmarshaling init.yaml: %w", err)
	}

	// Set default variables
	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("error retrieving current user information: %w", err)
	}

	viper.SetDefault("user.username", currentUser.Username)
	viper.SetDefault("user.home", currentUser.HomeDir)
	viper.SetDefault("user.shell", os.Getenv("SHELL"))

	// Retrieve user's full name, first name, and last name
	fullName := currentUser.Name
	names := strings.Fields(fullName)
	firstName := ""
	lastName := ""
	if len(names) > 0 {
		firstName = names[0]
	}
	if len(names) > 1 {
		lastName = names[len(names)-1]
	}

	viper.SetDefault("user.fullName", fullName)
	viper.SetDefault("user.firstName", firstName)
	viper.SetDefault("user.lastName", lastName)

	// Set user's group name
	groupName := ""
	if runtime.GOOS != "windows" {
		// Retrieve the user's primary group name on Unix-like systems
		group, err := user.LookupGroupId(currentUser.Gid)
		if err != nil {
			log.With("err", err).Warnf("Error retrieving primary group name for user %s", currentUser.Username)
		} else {
			groupName = group.Name
		}
	}

	viper.SetDefault("user.groupName", groupName)

	// Set user-defined variables from init.yaml
	for key, value := range initConfig.Variables {
		viper.Set(fmt.Sprintf("userDefined.%s", key), value)
	}

	// Read environment variables with RWR_ prefix and set them in userDefined
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "RWR_") {
			parts := strings.SplitN(env, "=", 2)
			key := strings.TrimPrefix(parts[0], "RWR_")
			viper.Set(fmt.Sprintf("userDefined.%s", key), parts[1])
		}
	}

	// Export all variables to the shell environment
	for _, key := range viper.AllKeys() {
		value := viper.GetString(key)
		envKey := fmt.Sprintf("RWR_VAR_%s", strings.ToUpper(strings.ReplaceAll(key, ".", "_")))
		err := os.Setenv(envKey, value)
		if err != nil {
			return nil, err
		}
	}

	return &initConfig, nil
}
