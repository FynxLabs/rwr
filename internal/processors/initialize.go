package processors

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

func Initialize(initFilePath string, flags types.Flags) (*types.InitConfig, error) {
	var initConfig types.InitConfig
	var err error
	var fileExt string
	var tempInitFile string

	// Create a temporary directory for downloaded or processed init files
	tempDir, err := os.MkdirTemp("", "rwr-init-")
	if err != nil {
		return nil, fmt.Errorf("error creating temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Handle URL or local file
	if strings.HasPrefix(initFilePath, "http://") || strings.HasPrefix(initFilePath, "https://") {
		log.Debugf("Init File is a Web URL, Downloading %s", initFilePath)

		fileExt = filepath.Ext(initFilePath)
		tempInitFile = filepath.Join(tempDir, "init"+fileExt)

		if strings.Contains(initFilePath, "/blob/") {
			log.Debugf("Treating init file as Github Blob URL")
			parts := strings.Split(initFilePath, "/")
			blobSplit := strings.Split(initFilePath, "/blob/")

			rawUrl := "https://raw.githubusercontent.com/" + parts[3] + "/" + parts[4] + "/" + blobSplit[1]

			log.Debugf("Created Raw URL: %s", rawUrl)

			err = system.DownloadFile(rawUrl, tempInitFile, false)
			if err != nil {
				return nil, fmt.Errorf("error downloading init file: %w", err)
			}

		} else {
			log.Debugf("Treating init file as Raw URL")
			log.Debugf("Setting downloaded file as temp: %s", tempInitFile)
			err = system.DownloadFile(initFilePath, tempInitFile, false)
			if err != nil {
				return nil, fmt.Errorf("error downloading init file: %w", err)
			}
		}
	} else {
		log.Debugf("Init File is local path: %s", initFilePath)
		if _, err := os.Stat(initFilePath); os.IsNotExist(err) {
			return nil, fmt.Errorf("init file not found at path: %s", initFilePath)
		}
		fileExt = filepath.Ext(initFilePath)
		tempInitFile = initFilePath
		log.Debugf("Using init file: %s", tempInitFile)
	}

	log.Debugf("Reading in temporary Init File: %s", tempInitFile)

	// Read the init file
	initFileData, err := os.ReadFile(tempInitFile)
	if err != nil {
		return nil, fmt.Errorf("error reading init file %s: %w", tempInitFile, err)
	}

	// Set default variables
	variables, err := setDefaultVariables()
	if err != nil {
		return nil, err
	}

	// Process the init file as a template
	processedInit, err := helpers.ResolveTemplate(initFileData, variables)
	if err != nil {
		return nil, fmt.Errorf("error processing init file as a template: %w", err)
	}

	// Convert TOML to YAML if necessary
	if fileExt == ".toml" {
		processedInit, fileExt, err = convertTomlToYaml(processedInit)
		if err != nil {
			return nil, err
		}
	}

	// Write the processed init file to the temporary directory
	processedInitFile := filepath.Join(tempDir, "init-processed"+fileExt)
	err = os.WriteFile(processedInitFile, processedInit, 0644)
	if err != nil {
		return nil, fmt.Errorf("error writing processed init file: %w", err)
	}

	log.Debugf("Processed Init File Path: %s", processedInitFile)

	// Read the processed init file with Viper
	viper.SetConfigFile(processedInitFile)
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading init file into viper: %w", err)
	}

	// Unmarshal the init file into the InitConfig struct
	if err := viper.Unmarshal(&initConfig); err != nil {
		return nil, fmt.Errorf("error unmarshaling %s: %w", processedInitFile, err)
	}

	// Set additional config values
	initConfig.Variables = variables
	initConfig.Variables.Flags = flags

	// Set the blueprints location
	setBlueprintsLocation(&initConfig, initFilePath)

	// Set user-defined variables and environment variables
	setUserDefinedAndEnvVariables(&initConfig)

	log.Debugf("Initialized initConfig: %v", initConfig)

	return &initConfig, nil
}

func setDefaultVariables() (types.Variables, error) {
	currentUser, err := user.Current()
	if err != nil {
		return types.Variables{}, fmt.Errorf("error retrieving current user information: %w", err)
	}

	names := strings.Fields(currentUser.Name)
	firstName, lastName := "", ""
	if len(names) > 0 {
		firstName = names[0]
	}
	if len(names) > 1 {
		lastName = names[len(names)-1]
	}

	groupName := ""
	if runtime.GOOS != "windows" {
		group, err := user.LookupGroupId(currentUser.Gid)
		if err != nil {
			log.With("err", err).Warnf("Error retrieving primary group name for user %s", currentUser.Username)
		} else {
			groupName = group.Name
		}
	}

	return types.Variables{
		User: types.UserInfo{
			Username:  currentUser.Username,
			FirstName: firstName,
			LastName:  lastName,
			FullName:  currentUser.Name,
			GroupName: groupName,
			Home:      currentUser.HomeDir,
			Shell:     os.Getenv("SHELL"),
		},
		UserDefined: make(map[string]interface{}),
	}, nil
}

func convertTomlToYaml(data []byte) ([]byte, string, error) {
	log.Debugf("TOML Format detected, converting to yaml for viper")
	var tempMap map[string]interface{}
	if _, err := toml.Decode(string(data), &tempMap); err != nil {
		return nil, "", fmt.Errorf("error decoding TOML: %w", err)
	}
	yamlData, err := yaml.Marshal(tempMap)
	if err != nil {
		return nil, "", fmt.Errorf("error converting TOML to YAML: %w", err)
	}
	return yamlData, ".yaml", nil
}

func setBlueprintsLocation(initConfig *types.InitConfig, initFilePath string) {
	// Handle Git target setup if needed
	if initConfig.Init.Git != nil && initConfig.Init.Git.Target != "" {
		resolvedTarget := system.ExpandPath(initConfig.Init.Git.Target)
		if err := os.MkdirAll(resolvedTarget, 0755); err != nil {
			log.Warnf("Failed to create blueprint directory: %v", err)
		}
	}

	// Set location based on init file rules
	if initConfig.Init.Location == "" || initConfig.Init.Location == "." {
		initConfig.Init.Location = filepath.Dir(initFilePath)
	} else if initConfig.Init.Location == "~" || strings.HasPrefix(initConfig.Init.Location, "~/") {
		homeDir, _ := os.UserHomeDir()
		initConfig.Init.Location = filepath.Join(homeDir, initConfig.Init.Location[2:])
	} else if !filepath.IsAbs(initConfig.Init.Location) {
		initConfig.Init.Location = filepath.Join(filepath.Dir(initFilePath), initConfig.Init.Location)
	}

	log.Debugf("Blueprints location set to: %s", initConfig.Init.Location)
}

func setUserDefinedAndEnvVariables(initConfig *types.InitConfig) {
	for key, value := range initConfig.Variables.UserDefined {
		initConfig.Variables.UserDefined[key] = value
	}

	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "RWR_") {
			parts := strings.SplitN(env, "=", 2)
			key := strings.TrimPrefix(parts[0], "RWR_")
			initConfig.Variables.UserDefined[key] = parts[1]
		}
	}

	for _, key := range viper.AllKeys() {
		value := viper.GetString(key)
		envKey := fmt.Sprintf("RWR_VAR_%s", strings.ToUpper(strings.ReplaceAll(key, ".", "_")))
		os.Setenv(envKey, value)
	}
}
