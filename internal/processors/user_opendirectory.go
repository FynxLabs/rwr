// The Open Directory backend: macOS user and group management via
// dscl. Selected at runtime on the detected OS — deliberately NOT a
// _darwin.go file, which would carry an implicit GOOS build constraint
// and break cross-platform validation.

package processors

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

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
