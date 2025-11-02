package processors

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/fynxlabs/rwr/internal/system"
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

	// Process imports and merge imported users and groups
	blueprintDir := initConfig.Init.Location
	allGroups, err := processGroupImports(usersData.Groups, blueprintDir, format)
	if err != nil {
		return fmt.Errorf("error processing group imports: %w", err)
	}
	usersData.Groups = allGroups

	allUsers, err := processUserImports(usersData.Users, blueprintDir, format)
	if err != nil {
		return fmt.Errorf("error processing user imports: %w", err)
	}
	usersData.Users = allUsers

	// Filter groups based on active profiles
	filteredGroups := helpers.FilterByProfiles(usersData.Groups, initConfig.Variables.Flags.Profiles)
	log.Debugf("Filtering groups: %d total, %d matching active profiles %v",
		len(usersData.Groups), len(filteredGroups), initConfig.Variables.Flags.Profiles)

	// Filter users based on active profiles
	filteredUsers := helpers.FilterByProfiles(usersData.Users, initConfig.Variables.Flags.Profiles)
	log.Debugf("Filtering users: %d total, %d matching active profiles %v",
		len(usersData.Users), len(filteredUsers), initConfig.Variables.Flags.Profiles)

	// Process the filtered groups
	err = processGroups(filteredGroups, initConfig)
	if err != nil {
		log.Errorf("Error processing groups: %v", err)
		return fmt.Errorf("error processing groups: %w", err)
	}

	// Process the filtered users
	err = processUsers(filteredUsers, initConfig)
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
		err := system.RunCommand(checkGroupCmd, initConfig.Variables.Flags.Debug)
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
		err = system.RunCommand(createGroupCmd, initConfig.Variables.Flags.Debug)
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
		err := system.RunCommand(createUserCmd, initConfig.Variables.Flags.Debug)
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

		err := system.RunCommand(modifyGroupCmd, initConfig.Variables.Flags.Debug)
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

		err := system.RunCommand(modifyUserCmd, initConfig.Variables.Flags.Debug)
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

		err := system.RunCommand(removeUserCmd, initConfig.Variables.Flags.Debug)
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

func processGroupImports(groups []types.Group, blueprintDir string, format string) ([]types.Group, error) {
	allGroups := make([]types.Group, 0)
	visited := make(map[string]bool)

	for _, group := range groups {
		if group.Import != "" {
			log.Debugf("Processing group import: %s", group.Import)

			importPath := filepath.Join(blueprintDir, group.Import)
			absPath, err := filepath.Abs(importPath)
			if err != nil {
				return nil, fmt.Errorf("error resolving import path %s: %w", importPath, err)
			}

			if visited[absPath] {
				log.Warnf("Circular import detected, skipping: %s", absPath)
				continue
			}
			visited[absPath] = true

			importData, err := os.ReadFile(importPath)
			if err != nil {
				return nil, fmt.Errorf("error reading import file %s: %w", importPath, err)
			}

			fileFormat := format
			if fileFormat == "" {
				ext := filepath.Ext(importPath)
				fileFormat = ext
			}

			var importedUsersData types.UsersData
			if err := helpers.UnmarshalBlueprint(importData, fileFormat, &importedUsersData); err != nil {
				return nil, fmt.Errorf("error unmarshaling import file %s: %w", importPath, err)
			}

			allGroups = append(allGroups, importedUsersData.Groups...)
			log.Debugf("Imported %d groups from %s", len(importedUsersData.Groups), group.Import)
		} else {
			allGroups = append(allGroups, group)
		}
	}

	return allGroups, nil
}

func processUserImports(users []types.User, blueprintDir string, format string) ([]types.User, error) {
	allUsers := make([]types.User, 0)
	visited := make(map[string]bool)

	for _, user := range users {
		if user.Import != "" {
			log.Debugf("Processing user import: %s", user.Import)

			importPath := filepath.Join(blueprintDir, user.Import)
			absPath, err := filepath.Abs(importPath)
			if err != nil {
				return nil, fmt.Errorf("error resolving import path %s: %w", importPath, err)
			}

			if visited[absPath] {
				log.Warnf("Circular import detected, skipping: %s", absPath)
				continue
			}
			visited[absPath] = true

			importData, err := os.ReadFile(importPath)
			if err != nil {
				return nil, fmt.Errorf("error reading import file %s: %w", importPath, err)
			}

			fileFormat := format
			if fileFormat == "" {
				ext := filepath.Ext(importPath)
				fileFormat = ext
			}

			var importedUsersData types.UsersData
			if err := helpers.UnmarshalBlueprint(importData, fileFormat, &importedUsersData); err != nil {
				return nil, fmt.Errorf("error unmarshaling import file %s: %w", importPath, err)
			}

			allUsers = append(allUsers, importedUsersData.Users...)
			log.Debugf("Imported %d users from %s", len(importedUsersData.Users), user.Import)
		} else {
			allUsers = append(allUsers, user)
		}
	}

	return allUsers, nil
}
