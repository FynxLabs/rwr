package actions

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
	"os"
	"os/user"
	"runtime"
	"strings"
)

type InitConfig struct {
	Blueprints struct {
		Format   string   `mapstructure:"format"`
		Location string   `mapstructure:"location"`
		Order    []string `mapstructure:"order"`
	} `mapstructure:"blueprints"`
	PackageManagers []struct {
		Name   string `mapstructure:"name"`
		Action string `mapstructure:"action"`
	} `mapstructure:"packageManagers"`
	Variables map[string]interface{} `mapstructure:"variables"`
}

func Initialize(initFilePath string) (*InitConfig, error) {
	var initConfig InitConfig

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
		os.Setenv(envKey, value)
	}

	log.Info("Initialization completed")

	return &initConfig, nil
}
