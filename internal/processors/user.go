package processors

import (
	"fmt"
	"runtime"

	"github.com/fynxlabs/rwr/internal/types"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
)

func ProcessUsers(blueprintData []byte, format string, initConfig *types.InitConfig) error {
	var usersData types.UsersData
	var err error

	// Unmarshal the blueprint data
	err = helpers.UnmarshalBlueprint(blueprintData, format, &usersData)
	if err != nil {
		log.Errorf("Error unmarshaling users blueprint: %v", err)
		return fmt.Errorf("error unmarshaling users blueprint: %w", err)
	}

	// Process the groups
	err = processGroups(usersData.Groups, initConfig)
	if err != nil {
		log.Errorf("Error processing groups: %v", err)
		return fmt.Errorf("error processing groups: %w", err)
	}

	// Process the users
	err = processUsers(usersData.Users, initConfig)
	if err != nil {
		log.Errorf("Error processing users: %v", err)
		return fmt.Errorf("error processing users: %w", err)
	}

	return nil
}

func processGroups(groups []types.Group, initConfig *types.InitConfig) error {
	for _, group := range groups {
		switch group.Action {
		case "create":
			err := createGroup(group, initConfig)
			if err != nil {
				log.Errorf("Error creating group %s: %v", group.Name, err)
				return fmt.Errorf("error creating group %s: %w", group.Name, err)
			}
			log.Infof("Group %s processed successfully", group.Name)
		case "modify":
			err := modifyGroup(group, initConfig)
			if err != nil {
				log.Errorf("Error modifying group %s: %v", group.Name, err)
				return fmt.Errorf("error modifying group %s: %w", group.Name, err)
			}
			log.Infof("Group %s modified successfully", group.Name)
		default:
			log.Errorf("Unsupported action for group %s: %s", group.Name, group.Action)
			return fmt.Errorf("unsupported action for group %s: %s", group.Name, group.Action)
		}
	}
	return nil
}

func processUsers(users []types.User, initConfig *types.InitConfig) error {
	for _, user := range users {
		switch user.Action {
		case "create":
			err := createUser(user, initConfig)
			if err != nil {
				log.Errorf("Error creating user %s: %v", user.Name, err)
				return fmt.Errorf("error creating user %s: %w", user.Name, err)
			}
			log.Infof("User %s created successfully", user.Name)
		case "modify":
			err := modifyUser(user, initConfig)
			if err != nil {
				log.Errorf("Error modifying user %s: %v", user.Name, err)
				return fmt.Errorf("error modifying user %s: %w", user.Name, err)
			}
			log.Infof("User %s modified successfully", user.Name)
		case "remove":
			err := removeUser(user, initConfig)
			if err != nil {
				log.Errorf("Error removing user %s: %v", user.Name, err)
				return fmt.Errorf("error removing user %s: %w", user.Name, err)
			}
			log.Infof("User %s removed successfully", user.Name)
		default:
			log.Errorf("Unsupported action for user %s: %s", user.Name, user.Action)
			return fmt.Errorf("unsupported action for user %s: %s", user.Name, user.Action)
		}
	}
	return nil
}

func createGroup(group types.Group, initConfig *types.InitConfig) error {
	switch runtime.GOOS {
	case "linux", "darwin":
		// Check if the group already exists
		checkGroupCmd := types.Command{
			Exec:     "getent",
			Args:     []string{"group", group.Name},
			Elevated: false,
		}
		err := helpers.RunCommand(checkGroupCmd, initConfig.Variables.Flags.Debug)
		if err == nil {
			// Group already exists, log a message and return without error
			log.Infof("Group %s already exists, skipping creation", group.Name)
			return nil
		}

		// If the group doesn't exist, create it
		createGroupCmd := types.Command{
			Exec:     "groupadd",
			Args:     []string{group.Name},
			Elevated: true,
		}
		err = helpers.RunCommand(createGroupCmd, initConfig.Variables.Flags.Debug)
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
			Elevated: true,
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

func modifyGroup(group types.Group, initConfig *types.InitConfig) error {
	switch runtime.GOOS {
	case "linux", "darwin":
		modifyGroupCmd := types.Command{
			Exec:     "groupmod",
			Args:     []string{group.Name},
			Elevated: true,
		}
		if group.NewName != "" {
			modifyGroupCmd.Args = append(modifyGroupCmd.Args, "--new-name", group.NewName)
		}
		// TODO: More groupmod options

		err := helpers.RunCommand(modifyGroupCmd, initConfig.Variables.Flags.Debug)
		if err != nil {
			return fmt.Errorf("error modifying group: %v", err)
		}
	case "windows":
		// Not supported on Windows
		log.Warnf("Modifying groups is not supported on Windows")
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
	return nil
}

func modifyUser(user types.User, initConfig *types.InitConfig) error {
	switch runtime.GOOS {
	case "linux", "darwin":
		modifyUserCmd := types.Command{
			Exec:     "usermod",
			Args:     []string{user.Name},
			Elevated: true,
		}
		if user.NewName != "" {
			modifyUserCmd.Args = append(modifyUserCmd.Args, "--login", user.NewName)
		}
		if user.NewHome != "" {
			modifyUserCmd.Args = append(modifyUserCmd.Args, "--move-home", "--home", user.NewHome)
		}
		if user.NewShell != "" {
			modifyUserCmd.Args = append(modifyUserCmd.Args, "--shell", user.NewShell)
		}
		if len(user.AddGroups) > 0 {
			for _, group := range user.AddGroups {
				modifyUserCmd.Args = append(modifyUserCmd.Args, "--append", "--groups", group)
			}
		}
		// TODO: More usermod options

		err := helpers.RunCommand(modifyUserCmd, initConfig.Variables.Flags.Debug)
		if err != nil {
			return fmt.Errorf("error modifying user: %v", err)
		}
	case "windows":
		// Not supported on Windows
		log.Warnf("Modifying users is not supported on Windows")
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
	return nil
}

func removeUser(user types.User, initConfig *types.InitConfig) error {
	switch runtime.GOOS {
	case "linux", "darwin":
		removeUserCmd := types.Command{
			Exec:     "userdel",
			Args:     []string{user.Name},
			Elevated: true,
		}
		if user.RemoveHome {
			removeUserCmd.Args = append(removeUserCmd.Args, "--remove")
		}
		// Add other options for removing users here

		err := helpers.RunCommand(removeUserCmd, initConfig.Variables.Flags.Debug)
		if err != nil {
			return fmt.Errorf("error removing user: %v", err)
		}
	case "windows":
		// Not supported on Windows
		log.Warnf("Removing users is not supported on Windows")
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
	return nil
}
