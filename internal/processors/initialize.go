package processors

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
)

func Initialize(initFilePath string, flags types.Flags) (*types.InitConfig, error) {
	var initConfig types.InitConfig
	var err error
	var fileExt string

	// Set default variables
	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("error retrieving current user information: %w", err)
	}

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

	userInfo := types.UserInfo{
		Username:  currentUser.Username,
		FirstName: firstName,
		LastName:  lastName,
		FullName:  currentUser.Name,
		GroupName: groupName,
		Home:      currentUser.HomeDir,
		Shell:     os.Getenv("SHELL"),
	}

	variables := types.Variables{
		User: userInfo,
	}

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
			err = helpers.DownloadFile(initFilePath, "init"+fileExt, false)
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

	// Get the directory of the init file
	initFileDir, err := filepath.Abs(filepath.Dir(initFilePath))
	if err != nil {
		return nil, fmt.Errorf("error getting absolute path of init file directory: %w", err)
	}
	log.Debugf("Initializing system information with init file: %s", initFilePath)
	log.Debugf("Init file directory: %s", initFileDir)

	// Check if init templates are enabled
	if flags.InitTemplatesEnabled {
		// Create a temporary directory for the processed init file
		log.Debugf("Processing init file as a template")
		tempDir, err := os.MkdirTemp("", "rwr-init-")
		if err != nil {
			return nil, fmt.Errorf("error creating temporary directory: %w", err)
		}
		defer func() {
			err := os.RemoveAll(tempDir)
			if err != nil {
				log.Errorf("error removing temporary directory: %v", err)
			}
		}()

		// Generate the processed init file path in the temporary directory
		processedInitFile := filepath.Join(tempDir, "init-processed"+fileExt)

		// Process the init file as a template
		initFileData, err := os.ReadFile(initFilePath)
		if err != nil {
			return nil, fmt.Errorf("error reading init file: %w", err)
		}

		processedInit, err := processTemplates(initFileData, initFileDir, &types.InitConfig{Variables: variables})
		if err != nil {
			log.Errorf("error processing init file as a template: %v", err)
			return nil, err
		}

		// Write the processed init file to the temporary directory
		err = os.WriteFile(processedInitFile, processedInit, 0644)
		if err != nil {
			log.Errorf("error writing processed init file: %v", err)
			return nil, err
		}

		initFilePath = processedInitFile
	}

	viper.SetConfigFile(initFilePath)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading init file: %w", err)
	}

	// Unmarshal the init file into the InitConfig struct
	if err := viper.Unmarshal(&initConfig); err != nil {
		return nil, fmt.Errorf("error unmarshaling init.yaml: %w", err)
	}

	// Set default values
	initConfig.Variables.User = userInfo

	initConfig.Variables.UserDefined = make(map[string]interface{})

	initConfig.Variables.Flags = flags

	// Set the default location if not specified
	if initConfig.Init.Location == "" {
		log.Debugf("Location not specified in init file. Using directory of the init file")
		initConfig.Init.Location = initFileDir
	} else if initConfig.Init.Location == "." {
		// If the location is ".", set it to the directory of the init file
		log.Debugf("Location set to current directory. Using directory of the init file")
		initConfig.Init.Location = initFileDir
	} else if initConfig.Init.Location == "~" || strings.HasPrefix(initConfig.Init.Location, "~/") {
		// If the location is "~" or starts with "~/", expand it to the user's home directory
		log.Debugf("Location is relative to the user's home directory. Expanding it")
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("error expanding home directory: %v", err)
		}
		initConfig.Init.Location = filepath.Join(homeDir, initConfig.Init.Location[2:])
	} else if !filepath.IsAbs(initConfig.Init.Location) {
		// If the location is relative, make it an absolute path relative to the init file path
		log.Debugf("Location is relative. Converting it to an absolute path relative to the init file directory")
		initConfig.Init.Location = filepath.Join(initFileDir, initConfig.Init.Location)
	}

	log.Debugf("Init file location: %s", initConfig.Init.Location)

	// Set user-defined variables from init.yaml
	for key, value := range initConfig.Variables.UserDefined {
		initConfig.Variables.UserDefined[key] = value
	}

	// Read environment variables with RWR_ prefix and set them in userDefined
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "RWR_") {
			parts := strings.SplitN(env, "=", 2)
			key := strings.TrimPrefix(parts[0], "RWR_")
			initConfig.Variables.UserDefined[key] = parts[1]
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
