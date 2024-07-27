package processors

import (
	"fmt"
	"github.com/fynxlabs/rwr/internal/types"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
)

func ProcessFiles(blueprintData []byte, blueprintDir string, format string, initConfig *types.InitConfig) error {
	var fileData types.FileData
	var err error

	log.Debugf("Processing files from blueprint")

	// Unmarshal the blueprint data
	err = helpers.UnmarshalBlueprint(blueprintData, format, &fileData)
	if err != nil {
		return fmt.Errorf("error unmarshaling file blueprint data: %w", err)
	}

	// Process regular files
	err = processFiles(fileData.Files, blueprintDir, initConfig)
	if err != nil {
		return fmt.Errorf("error processing files: %w", err)
	}

	// Process directories
	err = processDirectories(fileData.Directories, blueprintDir, initConfig)
	if err != nil {
		return fmt.Errorf("error processing directories: %w", err)
	}

	// Process templates
	err = processTemplates(fileData.Templates, blueprintDir, initConfig)
	if err != nil {
		return fmt.Errorf("error processing templates: %w", err)
	}

	return nil
}

func processFiles(files []types.File, blueprintDir string, initConfig *types.InitConfig) error {
	for _, file := range files {
		if len(file.Names) > 0 {
			for _, name := range file.Names {
				fileWithName := file
				fileWithName.Name = name
				if err := processFile(fileWithName, blueprintDir, initConfig); err != nil {
					return fmt.Errorf("error processing file %s: %w", name, err)
				}
			}
		} else {
			if err := processFile(file, blueprintDir, initConfig); err != nil {
				return fmt.Errorf("error processing file %s: %w", file.Name, err)
			}
		}
	}
	return nil
}

func processFile(file types.File, blueprintDir string, initConfig *types.InitConfig) error {
	if file.Content != "" {
		renderedContent, err := helpers.ResolveTemplate([]byte(file.Content), initConfig.Variables)
		if err != nil {
			log.Errorf("Error rendering template for file %s: %v", file.Target, err)
			return err
		}

		file.Content = string(renderedContent)
	}

	switch file.Action {
	case "copy":
		return copyFile(file, blueprintDir)
	case "move":
		return moveFile(file, blueprintDir)
	case "delete":
		return deleteFile(file)
	case "create":
		return createFile(file)
	case "chmod":
		return chmodFile(file)
	case "chown":
		return chownFile(file)
	case "chgrp":
		return chgrpFile(file)
	case "symlink":
		return symlinkFile(file, blueprintDir)
	default:
		return fmt.Errorf("unsupported action for file: %s", file.Action)
	}
}

func processDirectories(directories []types.Directory, blueprintDir string, initConfig *types.InitConfig) error {
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
	target := filepath.Join(helpers.ExpandPath(file.Target), file.Name)

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
	target := filepath.Join(helpers.ExpandPath(file.Target), file.Name)

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
	target := filepath.Join(helpers.ExpandPath(file.Target), file.Name)

	if err := os.Remove(target); err != nil {
		log.Fatalf("error deleting file: %v", err)
	}

	log.Infof("File deleted: %s", target)
	return nil
}

func createFile(file types.File) error {
	target := filepath.Join(helpers.ExpandPath(file.Target), file.Name)

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
	target := filepath.Join(helpers.ExpandPath(file.Target), file.Name)

	if err := os.Chmod(target, os.FileMode(file.Mode)); err != nil {
		log.Fatalf("error changing file permissions: %v", err)
	}

	log.Infof("File permissions changed: %s (mode: %o)", target, file.Mode)
	return nil
}

func chownFile(file types.File) error {
	target := filepath.Join(helpers.ExpandPath(file.Target), file.Name)

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
	target := filepath.Join(helpers.ExpandPath(file.Target), file.Name)

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
	target := helpers.ExpandPath(file.Target)

	if err := os.Symlink(source, target); err != nil {
		log.Fatalf("error creating symlink: %v", err)
	}

	log.Infof("Symlink created: %s -> %s", source, target)
	return nil
}

func copyDirectory(dir types.Directory, blueprintDir string, initConfig *types.InitConfig) error {
	source := filepath.Join(blueprintDir, dir.Source, dir.Name)
	target := filepath.Join(helpers.ExpandPath(dir.Target), dir.Name)

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
	target := filepath.Join(helpers.ExpandPath(dir.Target), dir.Name)

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
	target := filepath.Join(helpers.ExpandPath(dir.Target), dir.Name)

	if err := os.RemoveAll(target); err != nil {
		log.Fatalf("error deleting directory: %v", err)
	}

	log.Infof("Directory deleted: %s", target)
	return nil
}

func createDirectory(dir types.Directory) error {
	target := filepath.Join(helpers.ExpandPath(dir.Target), dir.Name)

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
	target := filepath.Join(helpers.ExpandPath(dir.Target), dir.Name)

	if err := os.Chmod(target, os.FileMode(dir.Mode)); err != nil {
		log.Fatalf("error changing directory permissions: %v", err)
	}

	log.Infof("Directory permissions changed: %s (mode: %o)", target, dir.Mode)
	return nil
}

func chownDirectory(dir types.Directory) error {
	target := filepath.Join(helpers.ExpandPath(dir.Target), dir.Name)

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
	target := filepath.Join(helpers.ExpandPath(dir.Target), dir.Name)

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
	target := helpers.ExpandPath(dir.Target)

	if err := os.Symlink(source, target); err != nil {
		log.Fatalf("error creating symlink: %v", err)
	}

	log.Infof("Symlink created: %s -> %s", source, target)
	return nil
}

func applyFileAttributes(file types.File) error {
	target := filepath.Join(helpers.ExpandPath(file.Target), file.Name)

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
	target := filepath.Join(helpers.ExpandPath(dir.Target), dir.Name)

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

func processTemplates(templates []types.Template, blueprintDir string, initConfig *types.InitConfig) error {
	for _, tmpl := range templates {
		log.Debugf("Processing template: %s", tmpl.Source)

		content, err := os.ReadFile(filepath.Join(blueprintDir, tmpl.Source))
		if err != nil {
			return fmt.Errorf("error reading template file %s: %w", tmpl.Source, err)
		}

		resolvedContent, err := helpers.ResolveTemplate(content, initConfig.Variables)
		if err != nil {
			return fmt.Errorf("error resolving template %s: %w", tmpl.Source, err)
		}

		if tmpl.Target != "" {
			targetPath := helpers.ExpandPath(tmpl.Target)
			err = helpers.WriteToFile(targetPath, string(resolvedContent), false)
			if err != nil {
				return fmt.Errorf("error writing rendered template to file %s: %w", targetPath, err)
			}
			log.Infof("Template processed and written to: %s", targetPath)
		}
	}
	return nil
}
