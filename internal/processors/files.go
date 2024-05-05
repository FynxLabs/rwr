package processors

import (
	"fmt"
	"github.com/thefynx/rwr/internal/types"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/helpers"
)

func ProcessFilesFromFile(blueprintFile string, blueprintDir string, initConfig *types.InitConfig) error {
	var files []types.File
	var directories []types.Directory

	log.Debugf("Processing files from blueprint file: %s", blueprintFile)

	// Read the blueprint file
	blueprintData, err := os.ReadFile(blueprintFile)
	if err != nil {
		log.Fatalf("error reading blueprint file: %v", err)
	}

	// Unmarshal the blueprint data
	var data struct {
		Files       []types.File      `yaml:"files" json:"files" toml:"files"`
		Directories []types.Directory `yaml:"directories" json:"directories" toml:"directories"`
	}

	err = helpers.UnmarshalBlueprint(blueprintData, filepath.Ext(blueprintFile), &data)
	if err != nil {
		log.Fatalf("error unmarshaling file blueprint data: %v", err)
	}
	files = data.Files
	directories = data.Directories

	// Process the files
	err = ProcessFiles(files, blueprintDir)
	if err != nil {
		log.Fatalf("error processing files: %v", err)
	}

	// Process the directories
	err = ProcessDirectories(directories, blueprintDir, initConfig)
	if err != nil {
		log.Fatalf("error processing directories: %v", err)
	}

	return nil
}

func ProcessFilesFromData(blueprintData []byte, blueprintDir string, initConfig *types.InitConfig) error {
	var files []types.File
	var directories []types.Directory

	log.Debugf("Processing files from blueprint data")

	// Unmarshal the resolved blueprint data
	var data struct {
		Files       []types.File      `yaml:"files" json:"files" toml:"files"`
		Directories []types.Directory `yaml:"directories" json:"directories" toml:"directories"`
	}
	err := helpers.UnmarshalBlueprint(blueprintData, initConfig.Init.Format, &data)
	if err != nil {
		log.Fatalf("error unmarshaling file blueprint data: %v", err)
	}
	files = data.Files
	directories = data.Directories

	// Process the files
	err = ProcessFiles(files, blueprintDir)
	if err != nil {
		log.Fatalf("error processing files: %v", err)
	}

	// Process the directories
	err = ProcessDirectories(directories, blueprintDir, initConfig)
	if err != nil {
		log.Fatalf("error processing directories: %v", err)
	}

	return nil
}

func ProcessFiles(files []types.File, blueprintDir string) error {
	for _, file := range files {
		if len(file.Names) > 0 {
			for _, name := range file.Names {
				fileWithName := file
				fileWithName.Name = name
				if err := processFile(fileWithName, blueprintDir); err != nil {
					log.Fatalf("error processing file: %v", err)
				}
			}
		} else {
			if err := processFile(file, blueprintDir); err != nil {
				log.Fatalf("error processing file: %v", err)
			}
		}
	}
	return nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, _ := os.UserHomeDir()
		path = filepath.Join(homeDir, path[2:])
	}
	return path
}

func processFile(file types.File, blueprintDir string) error {
	switch file.Action {
	case "copy":
		log.Debugf("Copying file: %s", file.Name)
		if err := copyFile(file, blueprintDir); err != nil {
			log.Fatalf("error copying file: %v", err)
		}
	case "move":
		log.Debugf("Moving file: %s", file.Name)
		if err := moveFile(file, blueprintDir); err != nil {
			log.Fatalf("error moving file: %v", err)
		}
	case "delete":
		log.Debugf("Deleting file: %s", file.Name)
		if err := deleteFile(file); err != nil {
			log.Fatalf("error deleting file: %v", err)
		}
	case "create":
		log.Debugf("Creating file: %s", file.Name)
		if err := createFile(file); err != nil {
			log.Fatalf("error creating file: %v", err)
		}
	case "chmod":
		log.Debugf("Changing file permissions: %s", file.Name)
		if err := chmodFile(file); err != nil {
			log.Fatalf("error changing file permissions: %v", err)
		}
	case "chown":
		log.Debugf("Changing file owner: %s", file.Name)
		if err := chownFile(file); err != nil {
			log.Fatalf("error changing file owner: %v", err)
		}
	case "chgrp":
		log.Debugf("Changing file group: %s", file.Name)
		if err := chgrpFile(file); err != nil {
			log.Fatalf("error changing file group: %v", err)
		}
	case "symlink":
		log.Debugf("Creating symlink: %s", file.Name)
		if err := symlinkFile(file, blueprintDir); err != nil {
			log.Fatalf("error creating symlink: %v", err)
		}
	default:
		return fmt.Errorf("unsupported action for file: %s", file.Action)
	}
	return nil
}

func ProcessDirectories(directories []types.Directory, blueprintDir string, initConfig *types.InitConfig) error {
	for _, dir := range directories {
		switch dir.Action {
		case "copy":
			if err := copyDirectory(dir, blueprintDir, initConfig); err != nil {
				log.Fatalf("error copying directory: %v", err)
			}
		case "move":
			if err := moveDirectory(dir, blueprintDir); err != nil {
				log.Fatalf("error moving directory: %v", err)
			}
		case "delete":
			if err := deleteDirectory(dir); err != nil {
				log.Fatalf("error deleting directory: %v", err)
			}
		case "create":
			if err := createDirectory(dir); err != nil {
				log.Fatalf("error creating directory: %v", err)
			}
		case "chmod":
			if err := chmodDirectory(dir); err != nil {
				log.Fatalf("error changing directory permissions: %v", err)
			}
		case "chown":
			if err := chownDirectory(dir); err != nil {
				log.Fatalf("error changing directory owner: %v", err)
			}
		case "chgrp":
			if err := chgrpDirectory(dir); err != nil {
				log.Fatalf("error changing directory group: %v", err)
			}
		case "symlink":
			if err := symlinkDirectory(dir, blueprintDir); err != nil {
				log.Fatalf("error creating symlink: %v", err)
			}
		default:
			return fmt.Errorf("unsupported action for directory: %s", dir.Action)
		}
	}
	return nil
}

func copyFile(file types.File, blueprintDir string) error {
	source := filepath.Join(blueprintDir, file.Source, file.Name)
	target := filepath.Join(expandPath(file.Target), file.Name)

	// Create the target directory if it doesn't exist
	targetDir := filepath.Dir(target)
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		log.Fatalf("error creating target directory: %v", err)
	}

	if err := helpers.CopyFile(source, target, file.Elevated); err != nil {
		log.Fatalf("error copying file: %v", err)
	}

	if err := applyFileAttributes(file); err != nil {
		log.Fatalf("error applying file attributes: %v", err)
	}

	log.Infof("File copied: %s -> %s", source, target)
	return nil
}

func moveFile(file types.File, blueprintDir string) error {
	source := filepath.Join(blueprintDir, file.Source, file.Name)
	target := filepath.Join(expandPath(file.Target), file.Name)

	// Create the target directory if it doesn't exist
	targetDir := filepath.Dir(target)
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		log.Fatalf("error creating target directory: %v", err)
	}

	if err := os.Rename(source, target); err != nil {
		log.Fatalf("error moving file: %v", err)
	}

	log.Infof("File moved: %s -> %s", source, target)
	return nil
}

func deleteFile(file types.File) error {
	target := filepath.Join(expandPath(file.Target), file.Name)

	if err := os.Remove(target); err != nil {
		log.Fatalf("error deleting file: %v", err)
	}

	log.Infof("File deleted: %s", target)
	return nil
}

func createFile(file types.File) error {
	target := filepath.Join(expandPath(file.Target), file.Name)

	// Create the target directory if it doesn't exist
	targetDir := filepath.Dir(target)
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		log.Fatalf("error creating target directory: %v", err)
	}

	if _, err := os.Create(target); err != nil {
		log.Fatalf("error creating file: %v", err)
	}

	if err := applyFileAttributes(file); err != nil {
		log.Fatalf("error applying file attributes: %v", err)
	}

	log.Infof("File created: %s", target)
	return nil
}

func chmodFile(file types.File) error {
	target := filepath.Join(expandPath(file.Target), file.Name)

	if err := os.Chmod(target, os.FileMode(file.Mode)); err != nil {
		log.Fatalf("error changing file permissions: %v", err)
	}

	log.Infof("File permissions changed: %s (mode: %o)", target, file.Mode)
	return nil
}

func chownFile(file types.File) error {
	target := filepath.Join(expandPath(file.Target), file.Name)

	if file.Owner != "" {
		uid, err := helpers.LookupUID(file.Owner)
		if err != nil {
			log.Fatalf("error looking up owner UID: %v", err)
		}
		if err := os.Chown(target, uid, -1); err != nil {
			log.Fatalf("error changing file owner: %v", err)
		}
	}

	if file.Group != "" {
		gid, err := helpers.LookupGID(file.Group)
		if err != nil {
			log.Fatalf("error looking up group GID: %v", err)
		}
		if err := os.Chown(target, -1, gid); err != nil {
			log.Fatalf("error changing file group: %v", err)
		}
	}

	log.Infof("File owner/group changed: %s (owner: %s, group: %s)", target, file.Owner, file.Group)
	return nil
}

func chgrpFile(file types.File) error {
	target := filepath.Join(expandPath(file.Target), file.Name)

	if file.Group != "" {
		gid, err := helpers.LookupGID(file.Group)
		if err != nil {
			log.Fatalf("error looking up group GID: %v", err)
		}
		if err := os.Chown(target, -1, gid); err != nil {
			log.Fatalf("error changing file group: %v", err)
		}
	}

	log.Infof("File group changed: %s (group: %s)", target, file.Group)
	return nil
}

func symlinkFile(file types.File, blueprintDir string) error {
	source := filepath.Join(blueprintDir, file.Source, file.Name)
	target := expandPath(file.Target)

	if err := os.Symlink(source, target); err != nil {
		log.Fatalf("error creating symlink: %v", err)
	}

	log.Infof("Symlink created: %s -> %s", source, target)
	return nil
}

func copyDirectory(dir types.Directory, blueprintDir string, initConfig *types.InitConfig) error {
	source := filepath.Join(blueprintDir, dir.Source, dir.Name)
	target := filepath.Join(expandPath(dir.Target), dir.Name)

	// Create the target directory if it doesn't exist
	if err := os.MkdirAll(target, os.ModePerm); err != nil {
		log.Fatalf("error creating target directory: %v", err)
	}

	if err := helpers.CopyDirectory(source, target, dir.Elevated, initConfig.Variables.Flags.Interactive); err != nil {
		log.Fatalf("error copying directory: %v", err)
	}

	if err := applyDirectoryAttributes(dir); err != nil {
		log.Fatalf("error applying directory attributes: %v", err)
	}

	log.Infof("Directory copied: %s -> %s", source, target)
	return nil
}

func moveDirectory(dir types.Directory, blueprintDir string) error {
	source := filepath.Join(blueprintDir, dir.Source, dir.Name)
	target := filepath.Join(expandPath(dir.Target), dir.Name)

	// Create the target directory if it doesn't exist
	targetDir := filepath.Dir(target)
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		log.Fatalf("error creating target directory: %v", err)
	}

	if err := os.Rename(source, target); err != nil {
		log.Fatalf("error moving directory: %v", err)
	}

	log.Infof("Directory moved: %s -> %s", source, target)
	return nil
}

func deleteDirectory(dir types.Directory) error {
	target := filepath.Join(expandPath(dir.Target), dir.Name)

	if err := os.RemoveAll(target); err != nil {
		log.Fatalf("error deleting directory: %v", err)
	}

	log.Infof("Directory deleted: %s", target)
	return nil
}

func createDirectory(dir types.Directory) error {
	target := filepath.Join(expandPath(dir.Target), dir.Name)

	if err := os.MkdirAll(target, os.ModePerm); err != nil {
		log.Fatalf("error creating directory: %v", err)
	}

	if err := applyDirectoryAttributes(dir); err != nil {
		log.Fatalf("error applying directory attributes: %v", err)
	}

	log.Infof("Directory created: %s", target)
	return nil
}

func chmodDirectory(dir types.Directory) error {
	target := filepath.Join(expandPath(dir.Target), dir.Name)

	if err := os.Chmod(target, os.FileMode(dir.Mode)); err != nil {
		log.Fatalf("error changing directory permissions: %v", err)
	}

	log.Infof("Directory permissions changed: %s (mode: %o)", target, dir.Mode)
	return nil
}

func chownDirectory(dir types.Directory) error {
	target := filepath.Join(expandPath(dir.Target), dir.Name)

	if dir.Owner != "" {
		uid, err := helpers.LookupUID(dir.Owner)
		if err != nil {
			log.Fatalf("error looking up owner UID: %v", err)
		}
		if err := os.Chown(target, uid, -1); err != nil {
			log.Fatalf("error changing directory owner: %v", err)
		}
	}

	if dir.Group != "" {
		gid, err := helpers.LookupGID(dir.Group)
		if err != nil {
			log.Fatalf("error looking up group GID: %v", err)
		}
		if err := os.Chown(target, -1, gid); err != nil {
			log.Fatalf("error changing directory group: %v", err)
		}
	}

	log.Infof("Directory owner/group changed: %s (owner: %s, group: %s)", target, dir.Owner, dir.Group)
	return nil
}

func chgrpDirectory(dir types.Directory) error {
	target := filepath.Join(expandPath(dir.Target), dir.Name)

	if dir.Group != "" {
		gid, err := helpers.LookupGID(dir.Group)
		if err != nil {
			log.Fatalf("error looking up group GID: %v", err)
		}
		if err := os.Chown(target, -1, gid); err != nil {
			log.Fatalf("error changing directory group: %v", err)
		}
	}

	log.Infof("Directory group changed: %s (group: %s)", target, dir.Group)
	return nil
}

func symlinkDirectory(dir types.Directory, blueprintDir string) error {
	source := filepath.Join(blueprintDir, dir.Source, dir.Name)
	target := expandPath(dir.Target)

	if err := os.Symlink(source, target); err != nil {
		log.Fatalf("error creating symlink: %v", err)
	}

	log.Infof("Symlink created: %s -> %s", source, target)
	return nil
}

func applyFileAttributes(file types.File) error {
	target := filepath.Join(expandPath(file.Target), file.Name)

	if file.Mode != 0 {
		if err := os.Chmod(target, os.FileMode(file.Mode)); err != nil {
			log.Fatalf("error changing file permissions: %v", err)
		}
	}

	if file.Owner != "" || file.Group != "" {
		if err := chownFile(file); err != nil {
			log.Fatalf("error changing file owner/group: %v", err)
		}
	}

	return nil
}

func applyDirectoryAttributes(dir types.Directory) error {
	target := filepath.Join(expandPath(dir.Target), dir.Name)

	if dir.Mode != 0 {
		if err := os.Chmod(target, os.FileMode(dir.Mode)); err != nil {
			log.Fatalf("error changing directory permissions: %v", err)
		}
	}

	if dir.Owner != "" || dir.Group != "" {
		if err := chownDirectory(dir); err != nil {
			log.Fatalf("error changing directory owner/group: %v", err)
		}
	}

	return nil
}
