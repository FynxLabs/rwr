// The shadow-utils backend: Linux user and group management via
// groupadd/usermod and friends. Selected at runtime on the detected OS —
// deliberately NOT a _linux.go file, which would carry an implicit GOOS
// build constraint and break cross-platform validation.

package processors

import (
	"fmt"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

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
