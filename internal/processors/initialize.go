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

	// Get the directory of the init file
	initFileDir := filepath.Dir(initFilePath)
	log.Debugf("Initializing system information with init file: %s", initFilePath)
	log.Debugf("Init file directory: %s", initFileDir)

	// Check if init templates are enabled
	if viper.GetBool("rwr.initTemplatesEnabled") {
		// Create a temporary directory for the processed init file
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
		err = processTemplate(types.Template{
			Source:    initFilePath,
			Target:    processedInitFile,
			Variables: viper.GetStringMap("user"),
		})
		if err != nil {
			return nil, fmt.Errorf("error processing init file as template: %w", err)
		}
		initFilePath = processedInitFile
	}

	viper.SetConfigFile(initFilePath)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading init file: %w", err)
	}

	if err := viper.Unmarshal(&initConfig); err != nil {
		return nil, fmt.Errorf("error unmarshaling init.yaml: %w", err)
	}

	// Set the default location if not specified
	if initConfig.Init.Location == "" {
		log.Debugf("Location not specified in init file. Using directory of the init file")
		initConfig.Init.Location = initFileDir
	} else if initConfig.Init.Location == "." {
		// If the location is ".", set it to the directory of the init file
		log.Debugf("Location set to current directory. Using directory of the init file")
		initConfig.Init.Location = initFileDir
	} else if !filepath.IsAbs(initConfig.Init.Location) {
		// If the location is relative, make it relative to the init file path
		log.Debugf("Location is relative. Making it relative to the init file directory")
		initConfig.Init.Location = filepath.Join(initFileDir, initConfig.Init.Location)
	}

	log.Debugf("Init file location: %s", initConfig.Init.Location)

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
