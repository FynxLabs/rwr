// The directories section of the files processor: per-action directory
// verbs and their attribute application.

package processors

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
)

func processDirectories(directories []types.Directory, blueprintDir string, initConfig *types.InitConfig, track *progress) error {
	track.expect("", len(directories))
	for _, dir := range directories {
		if system.IsDryRun() {
			log.Infof("[DRY-RUN] Would %s directory: %s (target: %s)", dir.Action, dir.Name, dir.Target)
			track.item("", dir.Name, dir.Action, types.StatusPlanned, "dry-run", 0)
			continue
		}
		started := time.Now()
		if err := processDirectory(dir, blueprintDir, initConfig); err != nil {
			recordFailure("directories", dir.Name, err)
			track.item("", dir.Name, dir.Action, types.StatusFailed, err.Error(), time.Since(started))
			continue
		}
		track.itemIdentity("", dir.Name, dir.Action, types.StatusOK, "", time.Since(started), map[string]string{"dest": system.ExpandPath(resolveTargetPath(dir.Target, dir.Name))})
	}
	return nil
}

func processDirectory(dir types.Directory, blueprintDir string, initConfig *types.InitConfig) error {
	switch dir.Action {
	case "copy":
		if err := copyDirectory(dir, blueprintDir, initConfig); err != nil {
			return fmt.Errorf("error copying directory: %w", err)
		}
	case "move":
		if err := moveDirectory(dir, blueprintDir); err != nil {
			return fmt.Errorf("error moving directory: %w", err)
		}
	case "delete":
		if err := deleteDirectory(dir); err != nil {
			return fmt.Errorf("error deleting directory: %w", err)
		}
	case "create":
		if err := createDirectory(dir); err != nil {
			return fmt.Errorf("error creating directory: %w", err)
		}
	case "chmod":
		if err := chmodDirectory(dir); err != nil {
			return fmt.Errorf("error changing directory permissions: %w", err)
		}
	case "chown":
		if err := chownDirectory(dir); err != nil {
			return fmt.Errorf("error changing directory owner: %w", err)
		}
	case "chgrp":
		if err := chgrpDirectory(dir); err != nil {
			return fmt.Errorf("error changing directory group: %w", err)
		}
	case "symlink":
		if err := symlinkDirectory(dir, blueprintDir); err != nil {
			return fmt.Errorf("error creating symlink: %w", err)
		}
	default:
		return fmt.Errorf("unsupported action for directory: %s", dir.Action)
	}
	return nil
}

func copyDirectory(dir types.Directory, blueprintDir string, initConfig *types.InitConfig) error {
	source := filepath.Join(blueprintDir, dir.Source, dir.Name)
	target := filepath.Join(system.ExpandPath(dir.Target), dir.Name)

	if err := os.MkdirAll(target, directoryMode(dir)); err != nil { // #nosec G703 -- target path is operator-supplied blueprint/config input; containment added in PR8
		return fmt.Errorf("error creating target directory: %w", err)
	}

	if err := system.CopyDirectory(source, target, dir.Elevated, helpers.ResolveInteractive(dir.Interactive, initConfig.Variables.Flags.Interactive)); err != nil {
		return fmt.Errorf("error copying directory: %w", err)
	}

	if err := applyDirectoryAttributes(dir); err != nil {
		return fmt.Errorf("error applying directory attributes: %w", err)
	}

	log.Infof("Directory copied: %s -> %s", source, target)
	return nil
}

func moveDirectory(dir types.Directory, blueprintDir string) error {
	source := filepath.Join(blueprintDir, dir.Source, dir.Name)
	target := filepath.Join(system.ExpandPath(dir.Target), dir.Name)

	targetDir := filepath.Dir(target)
	if err := os.MkdirAll(targetDir, defaultDirMode); err != nil {
		return fmt.Errorf("error creating target directory: %w", err)
	}

	if err := os.Rename(source, target); err != nil {
		if os.IsNotExist(err) {
			if _, statErr := os.Lstat(target); statErr == nil {
				log.Infof("Directory already moved: %s", target)
				return nil
			}
		}
		return fmt.Errorf("error moving directory: %w", err)
	}

	log.Infof("Directory moved: %s -> %s", source, target)
	return nil
}

func deleteDirectory(dir types.Directory) error {
	target := filepath.Join(system.ExpandPath(dir.Target), dir.Name)

	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("error deleting directory: %w", err)
	}

	log.Infof("Directory deleted: %s", target)
	return nil
}

func createDirectory(dir types.Directory) error {
	target := filepath.Join(system.ExpandPath(dir.Target), dir.Name)

	if err := os.MkdirAll(target, directoryMode(dir)); err != nil { // #nosec G703 -- target path is operator-supplied blueprint/config input; containment added in PR8
		return fmt.Errorf("error creating directory: %w", err)
	}

	if err := applyDirectoryAttributes(dir); err != nil {
		return fmt.Errorf("error applying directory attributes: %w", err)
	}

	log.Infof("Directory created: %s", target)
	return nil
}

func chmodDirectory(dir types.Directory) error {
	target := filepath.Join(system.ExpandPath(dir.Target), dir.Name)

	if !dir.Mode.IsSet() {
		return fmt.Errorf("no mode declared for chmod of %s: add mode: \"0755\" to the directory entry", target)
	}

	if err := os.Chmod(target, dir.Mode.OSMode()); err != nil {
		return fmt.Errorf("error changing directory permissions: %w", err)
	}

	log.Infof("Directory permissions changed: %s (mode: %s)", target, dir.Mode)
	return nil
}

func chownDirectory(dir types.Directory) error {
	target := filepath.Join(system.ExpandPath(dir.Target), dir.Name)

	if dir.Owner != "" {
		uid, err := system.LookupUID(dir.Owner)
		if err != nil {
			return fmt.Errorf("error looking up owner UID: %w", err)
		}
		if err := os.Chown(target, uid, -1); err != nil {
			return fmt.Errorf("error changing directory owner: %w", err)
		}
	}

	if dir.Group != "" {
		gid, err := system.LookupGID(dir.Group)
		if err != nil {
			return fmt.Errorf("error looking up group GID: %w", err)
		}
		if err := os.Chown(target, -1, gid); err != nil {
			return fmt.Errorf("error changing directory group: %w", err)
		}
	}

	log.Infof("Directory owner/group changed: %s (owner: %s, group: %s)", target, dir.Owner, dir.Group)
	return nil
}

// chgrpDirectory is the group half of chownDirectory: same lookup, same chown
// call, owner left untouched.
func chgrpDirectory(dir types.Directory) error {
	groupOnly := dir
	groupOnly.Owner = ""
	return chownDirectory(groupOnly)
}

func symlinkDirectory(dir types.Directory, blueprintDir string) error {
	source := filepath.Join(blueprintDir, dir.Source, dir.Name)
	target := system.ExpandPath(dir.Target)

	return ensureSymlink(source, target)
}

func directoryMode(dir types.Directory) os.FileMode {
	if dir.Mode.IsSet() {
		return dir.Mode.OSMode()
	}
	return defaultDirMode
}

func applyDirectoryAttributes(dir types.Directory) error {
	target := filepath.Join(system.ExpandPath(dir.Target), dir.Name)

	if dir.Mode.IsSet() {
		if err := os.Chmod(target, dir.Mode.OSMode()); err != nil {
			return fmt.Errorf("error changing directory permissions: %w", err)
		}
	}

	if dir.Owner != "" || dir.Group != "" {
		if err := chownDirectory(dir); err != nil {
			return fmt.Errorf("error changing directory owner/group: %w", err)
		}
	}

	return nil
}

func processDirectoryImports(items []types.Directory, blueprintDir string, format string, treeVersion int) ([]types.Directory, error) {
	return helpers.ResolveImports(items, blueprintDir,
		func(item types.Directory) string { return item.Import },
		func(data []byte, fileFormat string) ([]types.Directory, error) {
			var d types.FileData
			if err := helpers.DecodeBlueprintInto(data, fileFormat, types.BlueprintTypeFiles, treeVersion, &d); err != nil {
				return nil, err
			}
			return d.Directories, nil
		}, format)
}
