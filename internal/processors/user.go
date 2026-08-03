package processors

import (
	"fmt"
	"regexp"
	"runtime"
	"strconv"
	"strings"
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

// macUserInGroup uses dseditgroup's own membership query. dseditgroup -o edit -a
// fails when the user is already a member, so the add has to be conditional.
func macUserInGroup(user, group string, initConfig *types.InitConfig) bool {
	if system.IsDryRun() {
		return false
	}
	cmd := types.Command{
		Exec: "dseditgroup",
		Args: []string{"-o", "checkmember", "-m", user, group},
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

func createGroupLinux(group types.Group, initConfig *types.InitConfig) error {
	if groupExists(group.Name, initConfig, false) {
		log.Infof("Group %s already exists, converging attributes instead of creating", group.Name)
		return modifyGroup(types.Group{Name: group.Name, GID: group.GID}, initConfig)
	}

	args := []string{}
	if group.GID != "" {
		args = append(args, "--gid", group.GID)
	}
	if group.System {
		args = append(args, "--system")
	}
	args = append(args, group.Name)

	createGroupCmd := types.Command{
		Exec:     "groupadd",
		Args:     args,
		Elevated: true,
	}
	if err := system.RunCommand(createGroupCmd, initConfig.Variables.Flags.Debug); err != nil {
		return fmt.Errorf("error creating group: %v", err)
	}
	return nil
}

func createGroupDarwin(group types.Group, initConfig *types.InitConfig) error {
	if group.System {
		log.Warnf("Group %s: 'system' is ignored on macOS; GIDs below %d are reserved by Apple and are not allocated by rwr", group.Name, macMinID)
	}

	if groupExists(group.Name, initConfig, false) {
		log.Infof("Group %s already exists, converging attributes instead of creating", group.Name)
		return modifyGroup(types.Group{Name: group.Name, GID: group.GID}, initConfig)
	}

	gid := group.GID
	if gid == "" {
		allocated, err := macNextFreeID("/Groups", "PrimaryGroupID", initConfig)
		if err != nil {
			return fmt.Errorf("error allocating GID for group %s: %w", group.Name, err)
		}
		gid = allocated
	}

	record := "/Groups/" + group.Name
	steps := [][]string{
		{".", "-create", record},
		{".", "-create", record, "PrimaryGroupID", gid},
		{".", "-create", record, "RealName", group.Name},
	}
	for _, args := range steps {
		cmd := types.Command{Exec: "dscl", Args: args, Elevated: true}
		if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error creating group: %v", err)
		}
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

func modifyGroupLinux(group types.Group, initConfig *types.InitConfig) error {
	if group.NewName == "" && group.GID == "" {
		log.Debugf("Group %s: nothing to modify", group.Name)
		return nil
	}

	modifyGroupCmd := types.Command{
		Exec:     "groupmod",
		Args:     []string{group.Name},
		Elevated: true,
	}
	if group.NewName != "" {
		modifyGroupCmd.Args = append(modifyGroupCmd.Args, "--new-name", group.NewName)
	}
	if group.GID != "" {
		modifyGroupCmd.Args = append(modifyGroupCmd.Args, "--gid", group.GID)
	}

	if err := system.RunCommand(modifyGroupCmd, initConfig.Variables.Flags.Debug); err != nil {
		return fmt.Errorf("error modifying group: %v", err)
	}
	return nil
}

func modifyGroupDarwin(group types.Group, initConfig *types.InitConfig) error {
	record := "/Groups/" + group.Name

	if group.GID != "" {
		cmd := types.Command{
			Exec:     "dscl",
			Args:     []string{".", "-create", record, "PrimaryGroupID", group.GID},
			Elevated: true,
		}
		if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error modifying group: %v", err)
		}
	}

	// Renaming last: every earlier edit addresses the group by its old record name.
	if group.NewName != "" {
		cmd := types.Command{
			Exec:     "dscl",
			Args:     []string{".", "-change", record, "RecordName", group.Name, group.NewName},
			Elevated: true,
		}
		if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error renaming group: %v", err)
		}
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

func createUserLinux(user types.User, initConfig *types.InitConfig) error {
	if userExists(user.Name, initConfig, false) {
		log.Infof("User %s already exists, converging declared attributes instead of creating", user.Name)
		return convergeUserLinux(user, initConfig)
	}

	args := []string{"--create-home"}
	if user.UID != "" {
		args = append(args, "--uid", user.UID)
	}
	if user.Shell != "" {
		args = append(args, "--shell", user.Shell)
	}
	if user.Home != "" {
		args = append(args, "--home-dir", user.Home)
	}
	if user.Comment != "" {
		args = append(args, "--comment", user.Comment)
	}
	if user.System {
		args = append(args, "--system")
	}
	if user.Expire != "" {
		args = append(args, "--expiredate", user.Expire)
	}
	for _, group := range user.Groups {
		args = append(args, "--groups", group)
	}
	args = append(args, user.Name)

	createUserCmd := types.Command{
		Exec:        "useradd",
		Args:        args,
		Elevated:    true,
		Interactive: helpers.ResolveInteractive(user.Interactive, initConfig.Variables.Flags.Interactive),
	}
	if err := system.RunCommand(createUserCmd, initConfig.Variables.Flags.Debug); err != nil {
		return fmt.Errorf("error creating user: %v", err)
	}
	return setLinuxPassword(user, initConfig)
}

// convergeUserLinux applies a "create" declaration to an account that already
// exists. usermod is used rather than the NewX fields of modifyUser because the
// declared home is the intended home, not a request to relocate an existing one.
func convergeUserLinux(user types.User, initConfig *types.InitConfig) error {
	args := []string{}
	if user.UID != "" {
		args = append(args, "--uid", user.UID)
	}
	if user.Shell != "" {
		args = append(args, "--shell", user.Shell)
	}
	if user.Home != "" {
		args = append(args, "--home", user.Home)
	}
	if user.Comment != "" {
		args = append(args, "--comment", user.Comment)
	}
	if user.Expire != "" {
		args = append(args, "--expiredate", user.Expire)
	}
	for _, group := range user.Groups {
		args = append(args, "--append", "--groups", group)
	}
	if len(args) == 0 {
		log.Debugf("User %s already exists and declares no attributes to converge", user.Name)
		return setLinuxPassword(user, initConfig)
	}
	args = append(args, user.Name)

	cmd := types.Command{
		Exec:        "usermod",
		Args:        args,
		Elevated:    true,
		Interactive: helpers.ResolveInteractive(user.Interactive, initConfig.Variables.Flags.Interactive),
	}
	if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
		return fmt.Errorf("error converging existing user: %v", err)
	}
	return setLinuxPassword(user, initConfig)
}

func createUserDarwin(user types.User, initConfig *types.InitConfig) error {
	warnUnsupportedMacFields(user)

	if userExists(user.Name, initConfig, false) {
		log.Infof("User %s already exists, converging declared attributes instead of creating", user.Name)
		return convergeUserDarwin(user, initConfig)
	}

	uid := user.UID
	if uid == "" {
		allocated, err := macNextFreeID("/Users", "UniqueID", initConfig)
		if err != nil {
			return fmt.Errorf("error allocating UID for user %s: %w", user.Name, err)
		}
		uid = allocated
	}

	home := user.Home
	if home == "" {
		home = "/Users/" + user.Name
	}
	shell := user.Shell
	if shell == "" {
		shell = "/bin/zsh"
	}
	realName := user.Comment
	if realName == "" {
		realName = user.Name
	}

	interactive := helpers.ResolveInteractive(user.Interactive, initConfig.Variables.Flags.Interactive)

	if commandExists("sysadminctl") {
		cmd := types.Command{
			Exec: "sysadminctl",
			Args: []string{
				"-addUser", user.Name,
				"-fullName", realName,
				"-UID", uid,
				"-shell", shell,
				"-home", home,
			},
			Elevated:    true,
			Interactive: interactive,
		}
		if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error creating user: %v", err)
		}
	} else {
		record := "/Users/" + user.Name
		steps := [][]string{
			{".", "-create", record},
			{".", "-create", record, "UserShell", shell},
			{".", "-create", record, "RealName", realName},
			{".", "-create", record, "UniqueID", uid},
			{".", "-create", record, "PrimaryGroupID", macDefaultPrimaryGroupID},
			{".", "-create", record, "NFSHomeDirectory", home},
		}
		for _, args := range steps {
			cmd := types.Command{Exec: "dscl", Args: args, Elevated: true, Interactive: interactive}
			if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error creating user: %v", err)
			}
		}
	}

	// sysadminctl populates the home directory itself only in some releases, and
	// the dscl path never does; createhomedir is idempotent, so run it either way.
	createHomeCmd := types.Command{
		Exec:     "createhomedir",
		Args:     []string{"-c", "-u", user.Name},
		Elevated: true,
	}
	if err := system.RunCommand(createHomeCmd, initConfig.Variables.Flags.Debug); err != nil {
		return fmt.Errorf("error creating home directory for user %s: %v", user.Name, err)
	}

	if err := macSetPassword(user, initConfig); err != nil {
		return err
	}

	return macAddGroups(user.Name, user.Groups, initConfig)
}

func convergeUserDarwin(user types.User, initConfig *types.InitConfig) error {
	record := "/Users/" + user.Name
	attrs := [][2]string{}
	if user.Shell != "" {
		attrs = append(attrs, [2]string{"UserShell", user.Shell})
	}
	if user.Comment != "" {
		attrs = append(attrs, [2]string{"RealName", user.Comment})
	}
	if user.UID != "" {
		attrs = append(attrs, [2]string{"UniqueID", user.UID})
	}
	if user.Home != "" {
		attrs = append(attrs, [2]string{"NFSHomeDirectory", user.Home})
	}

	for _, attr := range attrs {
		cmd := types.Command{
			Exec:     "dscl",
			Args:     []string{".", "-create", record, attr[0], attr[1]},
			Elevated: true,
		}
		if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error converging existing user: %v", err)
		}
	}

	if err := macSetPassword(user, initConfig); err != nil {
		return err
	}

	return macAddGroups(user.Name, user.Groups, initConfig)
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

func modifyUserLinux(user types.User, initConfig *types.InitConfig) error {
	modifyUserCmd := types.Command{
		Exec:        "usermod",
		Args:        []string{user.Name},
		Elevated:    true,
		Interactive: helpers.ResolveInteractive(user.Interactive, initConfig.Variables.Flags.Interactive),
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
	if user.Comment != "" {
		modifyUserCmd.Args = append(modifyUserCmd.Args, "--comment", user.Comment)
	}
	if user.UID != "" {
		modifyUserCmd.Args = append(modifyUserCmd.Args, "--uid", user.UID)
	}
	if user.Expire != "" {
		modifyUserCmd.Args = append(modifyUserCmd.Args, "--expiredate", user.Expire)
	}
	if user.Lock {
		modifyUserCmd.Args = append(modifyUserCmd.Args, "--lock")
	}
	if user.Unlock {
		modifyUserCmd.Args = append(modifyUserCmd.Args, "--unlock")
	}
	if len(user.AddGroups) > 0 {
		for _, group := range user.AddGroups {
			modifyUserCmd.Args = append(modifyUserCmd.Args, "--append", "--groups", group)
		}
	}

	err := system.RunCommand(modifyUserCmd, initConfig.Variables.Flags.Debug)
	if err != nil {
		return fmt.Errorf("error modifying user: %v", err)
	}

	if err := setLinuxPassword(user, initConfig); err != nil {
		return err
	}

	// Remove groups via gpasswd (usermod doesn't support removing individual groups)
	for _, group := range user.RemoveGroups {
		removeGroupCmd := types.Command{
			Exec:     "gpasswd",
			Args:     []string{"--delete", user.Name, group},
			Elevated: true,
		}
		if err := system.RunCommand(removeGroupCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error removing user %s from group %s: %v", user.Name, group, err)
		}
	}
	return nil
}

func modifyUserDarwin(user types.User, initConfig *types.InitConfig) error {
	warnUnsupportedMacFields(user)

	record := "/Users/" + user.Name
	attrs := [][2]string{}
	if user.NewShell != "" {
		attrs = append(attrs, [2]string{"UserShell", user.NewShell})
	} else if user.Shell != "" {
		attrs = append(attrs, [2]string{"UserShell", user.Shell})
	}
	if user.Comment != "" {
		attrs = append(attrs, [2]string{"RealName", user.Comment})
	}
	if user.UID != "" {
		attrs = append(attrs, [2]string{"UniqueID", user.UID})
	}
	if user.NewHome != "" {
		attrs = append(attrs, [2]string{"NFSHomeDirectory", user.NewHome})
	} else if user.Home != "" {
		attrs = append(attrs, [2]string{"NFSHomeDirectory", user.Home})
	}

	interactive := helpers.ResolveInteractive(user.Interactive, initConfig.Variables.Flags.Interactive)

	for _, attr := range attrs {
		cmd := types.Command{
			Exec:        "dscl",
			Args:        []string{".", "-create", record, attr[0], attr[1]},
			Elevated:    true,
			Interactive: interactive,
		}
		if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error modifying user: %v", err)
		}
	}

	if user.NewHome != "" {
		// dscl only rewrites the record; the directory itself still has to exist.
		log.Warnf("User %s: macOS does not move an existing home directory; NFSHomeDirectory now points at %s but its contents were not relocated", user.Name, user.NewHome)
	}

	if err := macSetPassword(user, initConfig); err != nil {
		return err
	}

	if user.Lock {
		cmd := types.Command{Exec: "pwpolicy", Args: []string{"-u", user.Name, "-disableuser"}, Elevated: true}
		if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error locking user: %v", err)
		}
	}
	if user.Unlock {
		cmd := types.Command{Exec: "pwpolicy", Args: []string{"-u", user.Name, "-enableuser"}, Elevated: true}
		if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error unlocking user: %v", err)
		}
	}

	if err := macAddGroups(user.Name, user.AddGroups, initConfig); err != nil {
		return err
	}
	for _, group := range user.RemoveGroups {
		cmd := types.Command{
			Exec:     "dseditgroup",
			Args:     []string{"-o", "edit", "-d", user.Name, "-t", "user", group},
			Elevated: true,
		}
		if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error removing user %s from group %s: %v", user.Name, group, err)
		}
	}

	// Renaming last: every edit above addresses the account by its old record name.
	if user.NewName != "" {
		cmd := types.Command{
			Exec:     "dscl",
			Args:     []string{".", "-change", record, "RecordName", user.Name, user.NewName},
			Elevated: true,
		}
		if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error renaming user: %v", err)
		}
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

func removeUserLinux(user types.User, initConfig *types.InitConfig) error {
	if !userExists(user.Name, initConfig, true) {
		log.Infof("User %s does not exist, nothing to remove", user.Name)
		return nil
	}

	removeUserCmd := types.Command{
		Exec:        "userdel",
		Args:        []string{user.Name},
		Elevated:    true,
		Interactive: helpers.ResolveInteractive(user.Interactive, initConfig.Variables.Flags.Interactive),
	}
	if user.RemoveHome {
		removeUserCmd.Args = append(removeUserCmd.Args, "--remove")
	}

	if err := system.RunCommand(removeUserCmd, initConfig.Variables.Flags.Debug); err != nil {
		return fmt.Errorf("error removing user: %v", err)
	}
	return nil
}

func removeUserDarwin(user types.User, initConfig *types.InitConfig) error {
	if !userExists(user.Name, initConfig, true) {
		log.Infof("User %s does not exist, nothing to remove", user.Name)
		return nil
	}

	interactive := helpers.ResolveInteractive(user.Interactive, initConfig.Variables.Flags.Interactive)

	if commandExists("sysadminctl") {
		// sysadminctl deletes the home directory unless told otherwise, which is
		// the opposite of userdel's default; -keepHome restores parity with the
		// blueprint's remove_home flag.
		args := []string{"-deleteUser", user.Name}
		if !user.RemoveHome {
			args = append(args, "-keepHome")
		}
		cmd := types.Command{Exec: "sysadminctl", Args: args, Elevated: true, Interactive: interactive}
		if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error removing user: %v", err)
		}
		return nil
	}

	cmd := types.Command{
		Exec:        "dscl",
		Args:        []string{".", "-delete", "/Users/" + user.Name},
		Elevated:    true,
		Interactive: interactive,
	}
	if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
		return fmt.Errorf("error removing user: %v", err)
	}
	if user.RemoveHome {
		log.Warnf("User %s: remove_home requires sysadminctl, which is not available; the home directory was left in place", user.Name)
	}
	return nil
}

// ---------------------------------------------------------------------------
// macOS helpers
// ---------------------------------------------------------------------------

// warnUnsupportedMacFields reports the blueprint fields that have no Open
// Directory equivalent, rather than dropping them silently and letting the run
// look like it applied them.
func warnUnsupportedMacFields(user types.User) {
	if user.System {
		log.Warnf("User %s: 'system' is ignored on macOS; UIDs below %d are reserved by Apple and are not allocated by rwr", user.Name, macMinID)
	}
	if user.Expire != "" {
		log.Warnf("User %s: 'expire' is ignored on macOS; local Open Directory accounts have no expiration date field", user.Name)
	}
}

func macAddGroups(name string, groups []string, initConfig *types.InitConfig) error {
	for _, group := range groups {
		if macUserInGroup(name, group, initConfig) {
			log.Debugf("User %s is already a member of group %s", name, group)
			continue
		}
		cmd := types.Command{
			Exec:     "dseditgroup",
			Args:     []string{"-o", "edit", "-a", name, "-t", "user", group},
			Elevated: true,
		}
		if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error adding user %s to group %s: %v", name, group, err)
		}
	}
	return nil
}

// macSetPassword sets the account password through dscl.
//
// macOS stores a salted SHA-512 PBKDF2 blob it computes itself, so unlike Linux
// it needs the cleartext password — there is no hash to pre-compute. That value
// therefore appears in the argv of a sudo'd process, visible in `ps` to every
// local user and recorded in sudo's syslog entry.
//
// TODO: feed the password to `dscl . -passwd` on stdin instead.
// types.Command.Stdin and its RunCommand wiring exist now (this file already
// uses them for the Linux chpasswd path below); what remains is confirming
// dscl reads a password from stdin non-interactively on current macOS.
func macSetPassword(user types.User, initConfig *types.InitConfig) error {
	if user.Password == "" {
		return nil
	}
	log.Warnf("User %s: the password is passed as a command argument on macOS and is briefly visible to other local users in the process list", user.Name)

	cmd := types.Command{
		Exec:     "dscl",
		Args:     []string{".", "-passwd", "/Users/" + user.Name, user.Password},
		Elevated: true,
		// dscl offers no way to read the password from stdin, so it cannot be kept
		// out of argv the way chpasswd allows on Linux. It can at least be kept out
		// of the debug and dry-run log lines, which print the whole argv.
		Secrets: []string{user.Password},
	}
	if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
		// The error is deliberately not wrapped with anything derived from the
		// password value.
		return fmt.Errorf("error setting password for user %s: %v", user.Name, err)
	}
	return nil
}

// macNextFreeID picks the lowest unused numeric ID at or above macMinID by
// reading the IDs Open Directory already hands out.
func macNextFreeID(path, attribute string, initConfig *types.InitConfig) (string, error) {
	cmd := types.Command{Exec: "dscl", Args: []string{".", "-list", path, attribute}}
	out, err := system.RunCommandOutput(cmd, initConfig.Variables.Flags.Debug)
	if err != nil {
		return "", fmt.Errorf("error listing %s %s: %w", path, attribute, err)
	}

	used := map[int]bool{}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if id, convErr := strconv.Atoi(fields[len(fields)-1]); convErr == nil {
			used[id] = true
		}
	}

	id := macMinID
	for used[id] {
		id++
	}
	return strconv.Itoa(id), nil
}

// ---------------------------------------------------------------------------
// password handling
// ---------------------------------------------------------------------------

// modernCryptHash matches the "$id$" prefix of every crypt(3) scheme glibc and
// libxcrypt support ($1$, $5$, $6$, $2b$, $y$, $gy$, ...).
var modernCryptHash = regexp.MustCompile(`^\$[A-Za-z0-9_-]{1,12}\$`)

// desCryptHash matches the legacy 13-character DES hash.
var desCryptHash = regexp.MustCompile(`^[./A-Za-z0-9]{13}$`)

// setLinuxPassword sets the account password by piping "<name>:<value>" to
// chpasswd on standard input.
//
// The obvious approach — useradd/usermod --password — is wrong twice over. That
// flag writes its argument verbatim into the hash field of /etc/shadow, so a
// cleartext value does not become the account's password; it becomes a hash
// nothing can ever match and the account silently has none. And because the
// value is an argument of a sudo'd process it is readable in `ps` by every local
// user for the lifetime of the call, and lands in sudo's syslog record.
//
// Stdin avoids both: chpasswd hashes cleartext itself, `chpasswd -e` accepts a
// pre-computed hash, and neither form puts the secret in argv or in the debug
// log, which prints cmd.Args.
func setLinuxPassword(user types.User, initConfig *types.InitConfig) error {
	if user.Password == "" {
		return nil
	}

	name := user.Name
	if user.NewName != "" {
		// A rename in the same declaration has already been applied.
		name = user.NewName
	}

	args := []string{}
	if isCryptHash(user.Password) {
		args = append(args, "-e")
	}

	cmd := types.Command{
		Exec:     "chpasswd",
		Args:     args,
		Elevated: true,
		Stdin:    name + ":" + user.Password + "\n",
	}
	if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
		// The error deliberately excludes the value: it is the secret.
		return fmt.Errorf("error setting password for user %s: %v", name, err)
	}

	log.Debugf("Password set for user %s", name)
	return nil
}

// isCryptHash reports whether value is already a crypt(3) hash, which decides
// whether chpasswd is told to hash it or to take it as-is.
func isCryptHash(value string) bool {
	switch value {
	case "!", "!!", "*", "*LK*":
		return true
	}
	return modernCryptHash.MatchString(value) || desCryptHash.MatchString(value)
}

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
