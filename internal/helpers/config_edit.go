package helpers

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"charm.land/log/v2"
	"github.com/spf13/viper"
)

// EditConfig opens the config file in the operator's editor, creating a
// default file first when none exists. After the editor exits the file is
// re-parsed; a file that no longer parses is warned about, not reverted -
// it is the operator's file.
func EditConfig() error {
	path := ConfigFilePath()
	if path == "" {
		return fmt.Errorf("cannot resolve the config file path")
	}
	if _, err := os.Stat(path); err != nil {
		if err := EnsureConfigDir(viper.GetString("rwr.configdir")); err != nil {
			return err
		}
		if err := PrecreateSecureConfigFile(path); err != nil {
			return err
		}
	}

	editor := editorCommand()
	cmd := exec.Command(editor, path) // #nosec G204 -- $VISUAL/$EDITOR is the operator's own choice, the standard editor contract
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s exited: %w", editor, err)
	}

	check := viper.New()
	check.SetConfigFile(path)
	if err := check.ReadInConfig(); err != nil {
		log.Warnf("%s no longer parses: %v", path, err)
	}
	return nil
}

// editorCommand picks the editor: $VISUAL, $EDITOR, then a platform default.
func editorCommand() string {
	for _, env := range []string{"VISUAL", "EDITOR"} {
		if v := os.Getenv(env); v != "" {
			return v
		}
	}
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	return "vi"
}
