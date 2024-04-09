package processors

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/helpers"
	"github.com/thefynx/rwr/internal/processors/types"
)

func ProcessUsersFromFile(blueprintFile string) error {
	var usersData struct {
		Groups []types.Group `yaml:"groups"`
		Users  []types.User  `yaml:"users"`
	}

	// Read the blueprint file
	blueprintData, err := os.ReadFile(blueprintFile)
	if err != nil {
		log.Errorf("Error reading blueprint file: %v", err)
		return fmt.Errorf("error reading blueprint file: %w", err)
	}

	// Unmarshal the blueprint data
	err = helpers.UnmarshalBlueprint(blueprintData, filepath.Ext(blueprintFile), &usersData)
	if err != nil {
		log.Errorf("Error unmarshaling users blueprint: %v", err)
		return fmt.Errorf("error unmarshaling users blueprint: %w", err)
	}

	// Process the groups
	err = ProcessGroups(usersData.Groups)
	if err != nil {
		log.Errorf("Error processing groups: %v", err)
		return fmt.Errorf("error processing groups: %w", err)
	}

	// Process the users
	err = ProcessUsers(usersData.Users)
	if err != nil {
		log.Errorf("Error processing users: %v", err)
		return fmt.Errorf("error processing users: %w", err)
	}

	return nil
}

func ProcessUsersFromData(blueprintData []byte, initConfig *types.InitConfig) error {
	var usersData struct {
		Groups []types.Group `yaml:"groups"`
		Users  []types.User  `yaml:"users"`
	}

	// Unmarshal the resolved blueprint data
	err := helpers.UnmarshalBlueprint(blueprintData, initConfig.Blueprint.Format, &usersData)
	if err != nil {
		log.Errorf("Error unmarshaling users blueprint data: %v", err)
		return fmt.Errorf("error unmarshaling users blueprint data: %w", err)
	}

	// Process the groups
	err = ProcessGroups(usersData.Groups)
	if err != nil {
		log.Errorf("Error processing groups: %v", err)
		return fmt.Errorf("error processing groups: %w", err)
	}

	// Process the users
	err = ProcessUsers(usersData.Users)
	if err != nil {
		log.Errorf("Error processing users: %v", err)
		return fmt.Errorf("error processing users: %w", err)
	}

	return nil
}

func ProcessGroups(groups []types.Group) error {
	for _, group := range groups {
		if group.Action == "create" {
			err := createGroup(group)
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

func ProcessUsers(users []types.User) error {
	for _, user := range users {
		if user.Action == "create" {
			err := createUser(user)
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

func createGroup(group types.Group) error {
	switch runtime.GOOS {
	case "linux", "darwin":
		err := helpers.RunCommand("groupadd", group.Name)
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

func createUser(user types.User) error {
	switch runtime.GOOS {
	case "linux", "darwin":
		args := []string{
			"--create-home",
			"--password", user.Password,
			"--shell", user.Shell,
			"--home-dir", user.Home,
			user.Name,
		}
		for _, group := range user.Groups {
			args = append(args, "--groups", group)
		}
		err := helpers.RunCommand("useradd", args...)
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
