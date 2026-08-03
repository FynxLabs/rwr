package processors

import (
	"fmt"
	"regexp"
	"runtime"
	"time"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// userGOOS names the platform the user/group commands are built for. It is a
// variable rather than a direct runtime.GOOS read so the darwin branch can be
// exercised from a Linux CI runner, which is the only place these argv shapes
// can be asserted without a Mac.
var userGOOS = runtime.GOOS

// commandExists reports whether a helper binary is on PATH. Indirected for the
// same reason as userGOOS: sysadminctl is never present on the test runner, so
// without a seam only the fallback path would ever be covered.
var commandExists = system.CommandExists

// macOS reserves everything below 500 for the system; Apple's own tools start
// handing out local accounts and groups at 501.
const macMinID = 501

// macDefaultPrimaryGroupID is "staff", the primary group Directory Utility gives
// a local account. types.User has no primary-group field, so new accounts land
// here unless the blueprint adds them to other groups explicitly.
const macDefaultPrimaryGroupID = "20"

// ProcessUsers creates, modifies, or removes system users and groups as defined
// in blueprint data.
//
// Linux goes through shadow-utils (useradd/usermod/userdel, groupadd/groupmod,
// gpasswd); macOS goes through Open Directory (dscl, sysadminctl, dseditgroup,
// pwpolicy) — none of the shadow-utils binaries exist there. Windows has no
// implementation and logs a warning per entry.
func ProcessUsers(blueprintData []byte, blueprintDir string, format string, initConfig *types.InitConfig) error {
	var usersData types.UsersData
	var err error

	// Unmarshal the blueprint data
	err = helpers.DecodeBlueprintInto(blueprintData, format, types.BlueprintTypeUsers,
		helpers.TreeSchemaVersion(initConfig), &usersData)
	if err != nil {
		log.Errorf("Error unmarshaling users blueprint: %v", err)
		return fmt.Errorf("error unmarshaling users blueprint: %w", err)
	}

	// Process imports and merge imported users and groups
	allGroups, err := processGroupImports(usersData.Groups, blueprintDir, format, helpers.TreeSchemaVersion(initConfig))
	if err != nil {
		return fmt.Errorf("error processing group imports: %w", err)
	}
	usersData.Groups = allGroups

	allUsers, err := processUserImports(usersData.Users, blueprintDir, format, helpers.TreeSchemaVersion(initConfig))
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

	// One tracker across both loops: users and groups share the processor's
	// single lane, and a second tracker would reset its done/total counts.
	track := newProgress(types.BlueprintTypeUsers)

	// Process the filtered groups
	err = processGroups(filteredGroups, initConfig, track)
	if err != nil {
		log.Errorf("Error processing groups: %v", err)
		return fmt.Errorf("error processing groups: %w", err)
	}

	// Process the filtered users
	err = processUsers(filteredUsers, initConfig, track)
	if err != nil {
		log.Errorf("Error processing users: %v", err)
		return fmt.Errorf("error processing users: %w", err)
	}

	return nil
}

func processGroups(groups []types.Group, initConfig *types.InitConfig, track *progress) error {
	track.expect("", len(groups))
	for _, group := range groups {
		if system.IsDryRun() {
			log.Infof("[DRY-RUN] Would %s group: %s", group.Action, group.Name)
			track.item("", group.Name, group.Action, types.StatusPlanned, "dry-run", 0)
			continue
		}
		started := time.Now()
		switch group.Action {
		case types.UserActionCreate:
			err := createGroup(group, initConfig)
			if err != nil {
				log.Errorf("Error creating group %s: %v", group.Name, err)
				track.item("", group.Name, group.Action, types.StatusFailed, err.Error(), time.Since(started))
				return fmt.Errorf("error creating group %s: %w", group.Name, err)
			}
			log.Infof("Group %s processed successfully", group.Name)
		case types.UserActionModify:
			err := modifyGroup(group, initConfig)
			if err != nil {
				log.Errorf("Error modifying group %s: %v", group.Name, err)
				track.item("", group.Name, group.Action, types.StatusFailed, err.Error(), time.Since(started))
				return fmt.Errorf("error modifying group %s: %w", group.Name, err)
			}
			log.Infof("Group %s modified successfully", group.Name)
		case types.UserActionRemove, types.UserActionDelete:
			err := removeGroup(group, initConfig)
			if err != nil {
				log.Errorf("Error removing group %s: %v", group.Name, err)
				track.item("", group.Name, group.Action, types.StatusFailed, err.Error(), time.Since(started))
				return fmt.Errorf("error removing group %s: %w", group.Name, err)
			}
			log.Infof("Group %s removed successfully", group.Name)
		default:
			log.Errorf("Unsupported action for group %s: %s", group.Name, group.Action)
			track.item("", group.Name, group.Action, types.StatusFailed, "unsupported action", 0)
			return fmt.Errorf("unsupported action for group %s: %s", group.Name, group.Action)
		}
		if userGOOS == "windows" {
			// The per-action helpers warn and no-op on Windows.
			track.item("", group.Name, group.Action, types.StatusSkipped, "not supported on Windows", 0)
		} else {
			track.item("", group.Name, group.Action, types.StatusOK, "", time.Since(started))
		}
	}
	return nil
}

func processUsers(users []types.User, initConfig *types.InitConfig, track *progress) error {
	track.expect("", len(users))
	for _, user := range users {
		if system.IsDryRun() {
			log.Infof("[DRY-RUN] Would %s user: %s", user.Action, user.Name)
			track.item("", user.Name, user.Action, types.StatusPlanned, "dry-run", 0)
			continue
		}
		started := time.Now()
		switch user.Action {
		case types.UserActionCreate:
			err := createUser(user, initConfig)
			if err != nil {
				log.Errorf("Error creating user %s: %v", user.Name, err)
				track.item("", user.Name, user.Action, types.StatusFailed, err.Error(), time.Since(started))
				return fmt.Errorf("error creating user %s: %w", user.Name, err)
			}
			log.Infof("User %s created successfully", user.Name)
		case types.UserActionModify:
			err := modifyUser(user, initConfig)
			if err != nil {
				log.Errorf("Error modifying user %s: %v", user.Name, err)
				track.item("", user.Name, user.Action, types.StatusFailed, err.Error(), time.Since(started))
				return fmt.Errorf("error modifying user %s: %w", user.Name, err)
			}
			log.Infof("User %s modified successfully", user.Name)
		case types.UserActionRemove, types.UserActionDelete:
			err := removeUser(user, initConfig)
			if err != nil {
				log.Errorf("Error removing user %s: %v", user.Name, err)
				track.item("", user.Name, user.Action, types.StatusFailed, err.Error(), time.Since(started))
				return fmt.Errorf("error removing user %s: %w", user.Name, err)
			}
			log.Infof("User %s removed successfully", user.Name)
		default:
			log.Errorf("Unsupported action for user %s: %s", user.Name, user.Action)
			track.item("", user.Name, user.Action, types.StatusFailed, "unsupported action", 0)
			return fmt.Errorf("unsupported action for user %s: %s", user.Name, user.Action)
		}
		if userGOOS == "windows" {
			// The per-action helpers warn and no-op on Windows.
			track.item("", user.Name, user.Action, types.StatusSkipped, "not supported on Windows", 0)
		} else {
			track.item("", user.Name, user.Action, types.StatusOK, "", time.Since(started))
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// existence probes
//
// Every mutating path is guarded by one of these so a second `rwr all` converges
// instead of aborting: useradd exits non-zero on "user already exists", and that
// error used to propagate all the way up and abandon the rest of the run.
// ---------------------------------------------------------------------------

// groupExists probes for a group. In dry-run mode nothing is executed, so the
// probe returns assumeDryRun — callers pass whichever answer keeps the dry run
// printing the work it would do (absent before a create, present before a
// remove) instead of a skip message.
func groupExists(name string, initConfig *types.InitConfig, assumeDryRun bool) bool {
	if system.IsDryRun() {
		return assumeDryRun
	}
	var cmd types.Command
	switch userGOOS {
	case "darwin":
		cmd = types.Command{Exec: "dscl", Args: []string{".", "-read", "/Groups/" + name}}
	default:
		cmd = types.Command{Exec: "getent", Args: []string{"group", name}}
	}
	return system.RunCommand(cmd, initConfig.Variables.Flags.Debug) == nil
}

// userExists probes for a user; see groupExists for assumeDryRun.
func userExists(name string, initConfig *types.InitConfig, assumeDryRun bool) bool {
	if system.IsDryRun() {
		return assumeDryRun
	}
	var cmd types.Command
	switch userGOOS {
	case "darwin":
		cmd = types.Command{Exec: "dscl", Args: []string{".", "-read", "/Users/" + name}}
	default:
		cmd = types.Command{Exec: "getent", Args: []string{"passwd", name}}
	}
	return system.RunCommand(cmd, initConfig.Variables.Flags.Debug) == nil
}

// ---------------------------------------------------------------------------
// groups
// ---------------------------------------------------------------------------

func createGroup(group types.Group, initConfig *types.InitConfig) error {
	switch userGOOS {
	case "linux":
		return createGroupLinux(group, initConfig)
	case "darwin":
		return createGroupDarwin(group, initConfig)
	case "windows":
		log.Warnf("Creating groups is not supported on Windows")
	default:
		return fmt.Errorf("unsupported operating system: %s", userGOOS)
	}
	return nil
}

func modifyGroup(group types.Group, initConfig *types.InitConfig) error {
	switch userGOOS {
	case "linux":
		return modifyGroupLinux(group, initConfig)
	case "darwin":
		return modifyGroupDarwin(group, initConfig)
	case "windows":
		log.Warnf("Modifying groups is not supported on Windows")
	default:
		return fmt.Errorf("unsupported operating system: %s", userGOOS)
	}
	return nil
}

func removeGroup(group types.Group, initConfig *types.InitConfig) error {
	switch userGOOS {
	case "linux":
		if !groupExists(group.Name, initConfig, true) {
			log.Infof("Group %s does not exist, nothing to remove", group.Name)
			return nil
		}
		cmd := types.Command{Exec: "groupdel", Args: []string{group.Name}, Elevated: true}
		if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error removing group: %v", err)
		}
	case "darwin":
		if !groupExists(group.Name, initConfig, true) {
			log.Infof("Group %s does not exist, nothing to remove", group.Name)
			return nil
		}
		cmd := types.Command{Exec: "dscl", Args: []string{".", "-delete", "/Groups/" + group.Name}, Elevated: true}
		if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error removing group: %v", err)
		}
	case "windows":
		log.Warnf("Removing groups is not supported on Windows")
	default:
		return fmt.Errorf("unsupported operating system: %s", userGOOS)
	}
	return nil
}

// ---------------------------------------------------------------------------
// users
// ---------------------------------------------------------------------------

func createUser(user types.User, initConfig *types.InitConfig) error {
	switch userGOOS {
	case "linux":
		return createUserLinux(user, initConfig)
	case "darwin":
		return createUserDarwin(user, initConfig)
	case "windows":
		log.Warnf("Creating users is not supported on Windows")
	default:
		return fmt.Errorf("unsupported operating system: %s", userGOOS)
	}
	return nil
}

func modifyUser(user types.User, initConfig *types.InitConfig) error {
	switch userGOOS {
	case "linux":
		return modifyUserLinux(user, initConfig)
	case "darwin":
		return modifyUserDarwin(user, initConfig)
	case "windows":
		log.Warnf("Modifying users is not supported on Windows")
	default:
		return fmt.Errorf("unsupported operating system: %s", userGOOS)
	}
	return nil
}

func removeUser(user types.User, initConfig *types.InitConfig) error {
	switch userGOOS {
	case "linux":
		return removeUserLinux(user, initConfig)
	case "darwin":
		return removeUserDarwin(user, initConfig)
	case "windows":
		log.Warnf("Removing users is not supported on Windows")
	default:
		return fmt.Errorf("unsupported operating system: %s", userGOOS)
	}
	return nil
}

// ---------------------------------------------------------------------------
// password handling
// ---------------------------------------------------------------------------

// modernCryptHash matches the "$id$" prefix of every crypt(3) scheme glibc and
// libxcrypt support ($1$, $5$, $6$, $2b$, $y$, $gy$, ...).
var modernCryptHash = regexp.MustCompile(`^\$[A-Za-z0-9_-]{1,12}\$`)

// desCryptHash matches the legacy 13-character DES hash.
var desCryptHash = regexp.MustCompile(`^[./A-Za-z0-9]{13}$`)

func processGroupImports(items []types.Group, blueprintDir string, format string, treeVersion int) ([]types.Group, error) {
	return helpers.ResolveImports(items, blueprintDir,
		func(item types.Group) string { return item.Import },
		func(data []byte, fileFormat string) ([]types.Group, error) {
			var d types.UsersData
			if err := helpers.DecodeBlueprintInto(data, fileFormat, types.BlueprintTypeUsers, treeVersion, &d); err != nil {
				return nil, err
			}
			return d.Groups, nil
		}, format)
}

func processUserImports(items []types.User, blueprintDir string, format string, treeVersion int) ([]types.User, error) {
	return helpers.ResolveImports(items, blueprintDir,
		func(item types.User) string { return item.Import },
		func(data []byte, fileFormat string) ([]types.User, error) {
			var d types.UsersData
			if err := helpers.DecodeBlueprintInto(data, fileFormat, types.BlueprintTypeUsers, treeVersion, &d); err != nil {
				return nil, err
			}
			return d.Users, nil
		}, format)
}
