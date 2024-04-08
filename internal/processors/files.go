package processors

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/helpers"
	"github.com/thefynx/rwr/internal/processors/types"
)

func ProcessFilesFromFile(blueprintFile string) error {
	var files []types.File
	var directories []types.Directory

	// Read the blueprint file based on the file format
	switch filepath.Ext(blueprintFile) {
	case ".yaml", ".yml":
		var data struct {
			Files       []types.File      `yaml:"files"`
			Directories []types.Directory `yaml:"directories"`
		}
		err := helpers.ReadYAMLFile(blueprintFile, &data)
		if err != nil {
			return fmt.Errorf("error reading file blueprint file: %w", err)
		}
		files = data.Files
		directories = data.Directories
	case ".json":
		var data struct {
			Files       []types.File      `json:"files"`
			Directories []types.Directory `json:"directories"`
		}
		err := helpers.ReadJSONFile(blueprintFile, &data)
		if err != nil {
			return fmt.Errorf("error reading file blueprint file: %w", err)
		}
		files = data.Files
		directories = data.Directories
	case ".toml":
		var data struct {
			Files       []types.File      `toml:"files"`
			Directories []types.Directory `toml:"directories"`
		}
		err := helpers.ReadTOMLFile(blueprintFile, &data)
		if err != nil {
			return fmt.Errorf("error reading file blueprint file: %w", err)
		}
		files = data.Files
		directories = data.Directories
	default:
		return fmt.Errorf("unsupported blueprint file format: %s", filepath.Ext(blueprintFile))
	}

	// Process the files
	err := ProcessFiles(files)
	if err != nil {
		return fmt.Errorf("error processing files: %w", err)
	}

	// Process the directories
	err = ProcessDirectories(directories)
	if err != nil {
		return fmt.Errorf("error processing directories: %w", err)
	}

	return nil
}

func ProcessFiles(files []types.File) error {
	for _, file := range files {
		switch file.Action {
		case "copy":
			if err := copyFile(file); err != nil {
				return err
			}
		case "move":
			if err := moveFile(file); err != nil {
				return err
			}
		case "delete":
			if err := deleteFile(file); err != nil {
				return err
			}
		case "create":
			if err := createFile(file); err != nil {
				return err
			}
		case "chmod":
			if err := chmodFile(file); err != nil {
				return err
			}
		case "chown":
			if err := chownFile(file); err != nil {
				return err
			}
		case "chgrp":
			if err := chgrpFile(file); err != nil {
				return err
			}
		case "symlink":
			if err := symlinkFile(file); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported action for file: %s", file.Action)
		}
	}
	return nil
}

func ProcessDirectories(directories []types.Directory) error {
	for _, dir := range directories {
		switch dir.Action {
		case "copy":
			if err := copyDirectory(dir); err != nil {
				return err
			}
		case "move":
			if err := moveDirectory(dir); err != nil {
				return err
			}
		case "delete":
			if err := deleteDirectory(dir); err != nil {
				return err
			}
		case "create":
			if err := createDirectory(dir); err != nil {
				return err
			}
		case "chmod":
			if err := chmodDirectory(dir); err != nil {
				return err
			}
		case "chown":
			if err := chownDirectory(dir); err != nil {
				return err
			}
		case "chgrp":
			if err := chgrpDirectory(dir); err != nil {
				return err
			}
		case "symlink":
			if err := symlinkDirectory(dir); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported action for directory: %s", dir.Action)
		}
	}
	return nil
}

func copyFile(file types.File) error {
	if file.Create {
		if err := os.MkdirAll(filepath.Dir(file.Target), os.ModePerm); err != nil {
			return fmt.Errorf("error creating target directory: %v", err)
		}
	}

	source := filepath.Join(file.Source, file.Name)
	target := filepath.Join(file.Target, file.Name)

	if err := helpers.CopyFile(source, target); err != nil {
		return fmt.Errorf("error copying file: %v", err)
	}

	if err := applyFileAttributes(file); err != nil {
		return err
	}

	log.Infof("File copied: %s -> %s", source, target)
	return nil
}

func moveFile(file types.File) error {
	if file.Create {
		if err := os.MkdirAll(filepath.Dir(file.Target), os.ModePerm); err != nil {
			return fmt.Errorf("error creating target directory: %v", err)
		}
	}

	source := filepath.Join(file.Source, file.Name)
	target := filepath.Join(file.Target, file.Name)

	if err := os.Rename(source, target); err != nil {
		return fmt.Errorf("error moving file: %v", err)
	}

	log.Infof("File moved: %s -> %s", source, target)
	return nil
}

func deleteFile(file types.File) error {
	target := filepath.Join(file.Target, file.Name)

	if err := os.Remove(target); err != nil {
		return fmt.Errorf("error deleting file: %v", err)
	}

	log.Infof("File deleted: %s", target)
	return nil
}

func createFile(file types.File) error {
	if file.Create {
		if err := os.MkdirAll(filepath.Dir(file.Target), os.ModePerm); err != nil {
			return fmt.Errorf("error creating target directory: %v", err)
		}
	}

	target := filepath.Join(file.Target, file.Name)

	if _, err := os.Create(target); err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}

	if err := applyFileAttributes(file); err != nil {
		return err
	}

	log.Infof("File created: %s", target)
	return nil
}

func chmodFile(file types.File) error {
	target := filepath.Join(file.Target, file.Name)

	if err := os.Chmod(target, os.FileMode(file.Mode)); err != nil {
		return fmt.Errorf("error changing file permissions: %v", err)
	}

	log.Infof("File permissions changed: %s (mode: %o)", target, file.Mode)
	return nil
}

func chownFile(file types.File) error {
	target := filepath.Join(file.Target, file.Name)

	if err := os.Chown(target, file.Owner, -1); err != nil {
		return fmt.Errorf("error changing file owner: %v", err)
	}

	log.Infof("File owner changed: %s (owner: %d)", target, file.Owner)
	return nil
}

func chgrpFile(file types.File) error {
	target := filepath.Join(file.Target, file.Name)

	if err := os.Chown(target, -1, file.Group); err != nil {
		return fmt.Errorf("error changing file group: %v", err)
	}

	log.Infof("File group changed: %s (group: %d)", target, file.Group)
	return nil
}

func symlinkFile(file types.File) error {
	source := filepath.Join(file.Source, file.Name)
	target := file.Target

	if err := os.Symlink(source, target); err != nil {
		return fmt.Errorf("error creating symlink: %v", err)
	}

	log.Infof("Symlink created: %s -> %s", source, target)
	return nil
}

func copyDirectory(dir types.Directory) error {
	if dir.Create {
		if err := os.MkdirAll(filepath.Dir(dir.Target), os.ModePerm); err != nil {
			return fmt.Errorf("error creating target directory: %v", err)
		}
	}

	source := filepath.Join(dir.Source, dir.Name)
	target := filepath.Join(dir.Target, dir.Name)

	if err := helpers.CopyDirectory(source, target); err != nil {
		return fmt.Errorf("error copying directory: %v", err)
	}

	if err := applyDirectoryAttributes(dir); err != nil {
		return err
	}

	log.Infof("Directory copied: %s -> %s", source, target)
	return nil
}

func moveDirectory(dir types.Directory) error {
	if dir.Create {
		if err := os.MkdirAll(filepath.Dir(dir.Target), os.ModePerm); err != nil {
			return fmt.Errorf("error creating target directory: %v", err)
		}
	}

	source := filepath.Join(dir.Source, dir.Name)
	target := filepath.Join(dir.Target, dir.Name)

	if err := os.Rename(source, target); err != nil {
		return fmt.Errorf("error moving directory: %v", err)
	}

	log.Infof("Directory moved: %s -> %s", source, target)
	return nil
}

func deleteDirectory(dir types.Directory) error {
	target := filepath.Join(dir.Target, dir.Name)

	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("error deleting directory: %v", err)
	}

	log.Infof("Directory deleted: %s", target)
	return nil
}

func createDirectory(dir types.Directory) error {
	target := filepath.Join(dir.Target, dir.Name)

	if err := os.MkdirAll(target, os.ModePerm); err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}

	if err := applyDirectoryAttributes(dir); err != nil {
		return err
	}

	log.Infof("Directory created: %s", target)
	return nil
}

func chmodDirectory(dir types.Directory) error {
	target := filepath.Join(dir.Target, dir.Name)

	if err := os.Chmod(target, os.FileMode(dir.Mode)); err != nil {
		return fmt.Errorf("error changing directory permissions: %v", err)
	}

	log.Infof("Directory permissions changed: %s (mode: %o)", target, dir.Mode)
	return nil
}

func chownDirectory(dir types.Directory) error {
	target := filepath.Join(dir.Target, dir.Name)

	if err := os.Chown(target, dir.Owner, -1); err != nil {
		return fmt.Errorf("error changing directory owner: %v", err)
	}

	log.Infof("Directory owner changed: %s (owner: %d)", target, dir.Owner)
	return nil
}

func chgrpDirectory(dir types.Directory) error {
	target := filepath.Join(dir.Target, dir.Name)

	if err := os.Chown(target, -1, dir.Group); err != nil {
		return fmt.Errorf("error changing directory group: %v", err)
	}

	log.Infof("Directory group changed: %s (group: %d)", target, dir.Group)
	return nil
}

func symlinkDirectory(dir types.Directory) error {
	source := filepath.Join(dir.Source, dir.Name)
	target := dir.Target

	if err := os.Symlink(source, target); err != nil {
		return fmt.Errorf("error creating symlink: %v", err)
	}

	log.Infof("Symlink created: %s -> %s", source, target)
	return nil
}

func applyFileAttributes(file types.File) error {
	target := filepath.Join(file.Target, file.Name)

	if file.Mode != 0 {
		if err := os.Chmod(target, os.FileMode(file.Mode)); err != nil {
			return fmt.Errorf("error changing file permissions: %v", err)
		}
	}

	if file.Owner != 0 {
		if err := os.Chown(target, file.Owner, -1); err != nil {
			return fmt.Errorf("error changing file owner: %v", err)
		}
	}

	if file.Group != 0 {
		if err := os.Chown(target, -1, file.Group); err != nil {
			return fmt.Errorf("error changing file group: %v", err)
		}
	}

	return nil
}

func applyDirectoryAttributes(dir types.Directory) error {
	target := filepath.Join(dir.Target, dir.Name)

	if dir.Mode != 0 {
		if err := os.Chmod(target, os.FileMode(dir.Mode)); err != nil {
			return fmt.Errorf("error changing directory permissions: %v", err)
		}
	}

	if dir.Owner != 0 {
		if err := os.Chown(target, dir.Owner, -1); err != nil {
			return fmt.Errorf("error changing directory owner: %v", err)
		}
	}

	if dir.Group != 0 {
		if err := os.Chown(target, -1, dir.Group); err != nil {
			return fmt.Errorf("error changing directory group: %v", err)
		}
	}

	return nil
}
