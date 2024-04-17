package processors

import (
	"fmt"
	"github.com/thefynx/rwr/internal/types"
	"os"
	"path/filepath"
	"runtime"

	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/helpers"
)

func ProcessUsersFromFile(blueprintFile string, initConfig *types.InitConfig) error {
	// Read the blueprint file
	blueprintData, err := os.ReadFile(blueprintFile)
	if err != nil {
		log.Errorf("Error reading blueprint file: %v", err)
		return fmt.Errorf("error reading blueprint file: %w", err)
	}

	var usersData types.UsersData

	// Unmarshal the blueprint data
	err = helpers.UnmarshalBlueprint(blueprintData, filepath.Ext(blueprintFile), &usersData)
	if err != nil {
		log.Errorf("Error unmarshaling users blueprint: %v", err)
		return fmt.Errorf("error unmarshaling users blueprint: %w", err)
	}

	// Process the groups
	err = ProcessGroups(usersData.Groups, initConfig)
	if err != nil {
		log.Errorf("Error processing groups: %v", err)
		return fmt.Errorf("error processing groups: %w", err)
	}

	// Process the users
	err = ProcessUsers(usersData.Users, initConfig)
	if err != nil {
		log.Errorf("Error processing users: %v", err)
		return fmt.Errorf("error processing users: %w", err)
	}

	return nil
}

func ProcessUsersFromData(blueprintData []byte, initConfig *types.InitConfig) error {

	var usersData types.UsersData

	// Unmarshal the resolved blueprint data
	err := helpers.UnmarshalBlueprint(blueprintData, initConfig.Init.Format, &usersData)
	if err != nil {
		log.Errorf("Error unmarshaling users blueprint data: %v", err)
		return fmt.Errorf("error unmarshaling users blueprint data: %w", err)
	}

	// Process the groups
	err = ProcessGroups(usersData.Groups, initConfig)
	if err != nil {
		log.Errorf("Error processing groups: %v", err)
		return fmt.Errorf("error processing groups: %w", err)
	}

	// Process the users
	err = ProcessUsers(usersData.Users, initConfig)
	if err != nil {
		log.Errorf("Error processing users: %v", err)
		return fmt.Errorf("error processing users: %w", err)
	}

	return nil
}

func ProcessGroups(groups []types.Group, initConfig *types.InitConfig) error {
	for _, group := range groups {
		if group.Action == "create" {
			err := createGroup(group, initConfig)
			if err != nil {
				log.Errorf("Error creating group %s: %v", group.Name, err)
				return fmt.Errorf("error creating group %s: %w", group.Name, err)
			}
			log.Infof("Group %s created successfully", group.Name)
		} else {
			log.Errorf("Unsupported action for group %s: %s", group.Name, group.Action)
			return fmt.Errorf("unsupported action for group %s: %s", group.Name, group.Action)
		}
	}
	return nil
}

func ProcessUsers(users []types.User, initConfig *types.InitConfig) error {
	for _, user := range users {
		if user.Action == "create" {
			err := createUser(user, initConfig)
			if err != nil {
				log.Errorf("Error creating user %s: %v", user.Name, err)
				return fmt.Errorf("error creating user %s: %w", user.Name, err)
			}
			log.Infof("User %s created successfully", user.Name)
		} else {
			log.Errorf("Unsupported action for user %s: %s", user.Name, user.Action)
			return fmt.Errorf("unsupported action for user %s: %s", user.Name, user.Action)
		}
	}
	return nil
}

func createGroup(group types.Group, initConfig *types.InitConfig) error {
	switch runtime.GOOS {
	case "linux", "darwin":
		createGroupCmd := types.Command{
			Exec: "groupadd",
			Args: []string{group.Name},
		}
		err := helpers.RunCommand(createGroupCmd, initConfig.Variables.Flags.Debug)
		if err != nil {
			return fmt.Errorf("error creating group: %v", err)
		}
	case "windows":
		// Not supported on Windows
		log.Warnf("Creating groups is not supported on Windows")
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
	return nil
}

func createUser(user types.User, initConfig *types.InitConfig) error {
	switch runtime.GOOS {
	case "linux", "darwin":
		createUserCmd := types.Command{
			Exec: "useradd",
			Args: []string{
				"--create-home",
				"--password", user.Password,
				"--shell", user.Shell,
				"--home-dir", user.Home,
				user.Name,
			},
		}
		for _, group := range user.Groups {
			createUserCmd.Args = append(createUserCmd.Args, "--groups", group)
		}
		err := helpers.RunCommand(createUserCmd, initConfig.Variables.Flags.Debug)
		if err != nil {
			return fmt.Errorf("error creating user: %v", err)
		}
	case "windows":
		// Not supported on Windows
		log.Warnf("Creating users is not supported on Windows")
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
	return nil
}
