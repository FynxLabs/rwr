package processors

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
)

func ProcessFiles(blueprintData []byte, blueprintDir string, format string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var fileData types.FileData
	var err error

	log.Debugf("Processing files from blueprint")

	// Unmarshal the blueprint data
	err = helpers.UnmarshalBlueprint(blueprintData, format, &fileData)
	if err != nil {
		return fmt.Errorf("error unmarshaling file blueprint data: %w", err)
	}

	// Filter files based on active profiles
	filteredFiles := helpers.FilterByProfiles(fileData.Files, initConfig.Variables.Flags.Profiles)
	log.Debugf("Filtering files: %d total, %d matching active profiles %v",
		len(fileData.Files), len(filteredFiles), initConfig.Variables.Flags.Profiles)

	// Filter directories based on active profiles
	filteredDirectories := helpers.FilterByProfiles(fileData.Directories, initConfig.Variables.Flags.Profiles)
	log.Debugf("Filtering directories: %d total, %d matching active profiles %v",
		len(fileData.Directories), len(filteredDirectories), initConfig.Variables.Flags.Profiles)

	// Filter templates based on active profiles
	filteredTemplates := helpers.FilterByProfiles(fileData.Templates, initConfig.Variables.Flags.Profiles)
	log.Debugf("Filtering templates: %d total, %d matching active profiles %v",
		len(fileData.Templates), len(filteredTemplates), initConfig.Variables.Flags.Profiles)

	// Process filtered files
	err = processFiles(filteredFiles, blueprintDir, osInfo)
	if err != nil {
		return fmt.Errorf("error processing files: %w", err)
	}

	// Process filtered directories
	err = processDirectories(filteredDirectories, blueprintDir, initConfig)
	if err != nil {
		return fmt.Errorf("error processing directories: %w", err)
	}

	// Process filtered templates
	err = processTemplates(filteredTemplates, blueprintDir, osInfo, initConfig)
	if err != nil {
		return fmt.Errorf("error processing templates: %w", err)
	}

	return nil
}

func processFiles(files []types.File, blueprintDir string, osInfo *types.OSInfo) error {
	for _, file := range files {
		if len(file.Names) > 0 {
			for _, name := range file.Names {
				fileWithName := file
				fileWithName.Name = name
				if err := processFile(fileWithName, blueprintDir, osInfo); err != nil {
					return fmt.Errorf("error processing file %s: %w", name, err)
				}
			}
		} else {
			if err := processFile(file, blueprintDir, osInfo); err != nil {
				return fmt.Errorf("error processing file %s: %w", file.Name, err)
			}
		}
	}
	return nil
}

func processFile(file types.File, blueprintDir string, osInfo *types.OSInfo) error {

	log.Debugf("Processing file: %s", file.Name)

	if file.Content == "" && file.Source == "" {
		return fmt.Errorf("either Content or Source must be provided for file %s", file.Name)
	}

	// Handle URL source
	if isURL(file.Source) {
		log.Debug("File Source is URL")
		tempDir, err := os.MkdirTemp("", "rwr-download-")
		if err != nil {
			return fmt.Errorf("error creating temporary directory: %v", err)
		}
		defer func() {
			if removeErr := os.RemoveAll(tempDir); removeErr != nil {
				log.Errorf("Error removing temporary directory: %v", removeErr)
				if err == nil {
					err = removeErr
				}
			}
		}()

		log.Debug("Downloading Source File")
		downloadPath := filepath.Join(tempDir, filepath.Base(file.Source))
		err = system.DownloadFile(file.Source, downloadPath, false)
		if err != nil {
			return fmt.Errorf("error downloading file: %v", err)
		}

		log.Debug("Setting File Source and Name")
		file.Source = filepath.Dir(downloadPath)
		file.Name = filepath.Base(downloadPath)
	}

	// If Content exists, we'll always use it and perform a create action
	if file.Content != "" {
		if file.Action != "create" {
			log.Warnf("File %s has Content but action is not 'create'. Defaulting to 'create' action.", file.Name)
		}
		file.Action = "create"
	}

	// Determine source and target paths
	sourcePath, targetPath, err := determineSourceAndTargetPaths(file, blueprintDir)
	if err != nil {
		return err
	}

	log.Debugf("sourcePath set to: %s; targetPath set to: %s", sourcePath, targetPath)

	switch file.Action {
	case "copy":
		log.Debugf("Copying file: %s to %s (elevated: %v)", sourcePath, targetPath, file.Elevated)
		return system.CopyFile(sourcePath, targetPath, file.Elevated, osInfo)
	case "move":
		log.Debugf("Moving file: %s to %s", sourcePath, targetPath)
		return moveFile(file, blueprintDir)
	case "delete":
		log.Debugf("Deleting file: %s", targetPath)
		return deleteFile(file)
	case "create":
		log.Debugf("Creating file: %s", targetPath)
		return createFile(file)
	case "chmod":
		log.Debugf("Changing file permissions: %s", targetPath)
		return chmodFile(file)
	case "chown":
		log.Debugf("Changing file owner: %s", targetPath)
		return chownFile(file)
	case "chgrp":
		log.Debugf("Changing file group: %s", targetPath)
		return chgrpFile(file)
	case "symlink":
		log.Debugf("Symlinking file: %s to %s", sourcePath, targetPath)
		return symlinkFile(file, blueprintDir)
	default:
		return fmt.Errorf("unsupported action for file: %s", file.Action)
	}
}

func processTemplates(templates []types.File, blueprintDir string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	log.Info("Starting to process templates")
	for i, tmpl := range templates {
		log.Debugf("Processing template %d: %+v", i, tmpl)
		if tmpl.Name == "" && len(tmpl.Names) == 0 {
			log.Warn("Skipping empty template")
			continue
		}
		if len(tmpl.Names) > 0 {
			log.Debugf("Template has multiple names: %v", tmpl.Names)
			for _, name := range tmpl.Names {
				log.Infof("Processing template with name: %s", name)
				fileWithName := tmpl
				fileWithName.Name = name
				err := processTemplate(fileWithName, blueprintDir, osInfo, initConfig)
				if err != nil {
					log.Errorf("Error processing template to file %s: %v", fileWithName.Name, err)
					return fmt.Errorf("error processing template to file %s: %w", fileWithName.Name, err)
				}
			}
		} else {
			log.Infof("Processing single template: %s", tmpl.Name)
			err := processTemplate(tmpl, blueprintDir, osInfo, initConfig)
			if err != nil {
				log.Errorf("Error processing template to file %s: %v", tmpl.Name, err)
				return fmt.Errorf("error processing template to file %s: %w", tmpl.Name, err)
			}
		}
	}
	log.Info("Finished processing all templates")
	return nil
}

func processTemplate(template types.File, blueprintDir string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	log.Infof("Processing template: %s", template.Name)

	if template.Name == "" || template.Source == "" || template.Target == "" {
		log.Warnf("Skipping template with missing required fields: %+v", template)
		return nil
	}

	sourcePath := filepath.Join(blueprintDir, template.Source, template.Name)
	log.Debugf("Full source path: %s", sourcePath)

	content, err := os.ReadFile(sourcePath)
	if err != nil {
		log.Errorf("Error reading template file %s: %v", sourcePath, err)
		return fmt.Errorf("error reading template file %s: %w", sourcePath, err)
	}
	log.Debugf("Successfully read template file, content length: %d bytes", len(content))

	log.Debug("Resolving template variables")
	mergedVariables := initConfig.Variables
	for k, v := range template.Variables {
		mergedVariables.UserDefined[k] = v
	}
	resolvedContent, err := helpers.ResolveTemplate(content, mergedVariables)
	if err != nil {
		log.Errorf("Error resolving template %s: %v", sourcePath, err)
		return fmt.Errorf("error resolving template %s: %w", sourcePath, err)
	}
	log.Debugf("Successfully resolved template, new content length: %d bytes", len(resolvedContent))

	// Create a File struct from the Template
	file := types.File{
		Name:     template.Name,
		Action:   template.Action,
		Content:  string(resolvedContent),
		Target:   template.Target,
		Owner:    template.Owner,
		Group:    template.Group,
		Mode:     template.Mode,
		Elevated: template.Elevated,
	}

	// Process the template as a file
	err = processFile(file, blueprintDir, osInfo)
	if err != nil {
		log.Errorf("Error processing template as file %s: %v", template.Name, err)
		return fmt.Errorf("error processing template as file %s: %w", template.Name, err)
	}

	log.Infof("Template processed successfully: %s", template.Name)
	return nil
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

func moveFile(file types.File, blueprintDir string) error {
	source := filepath.Join(blueprintDir, file.Source, file.Name)
	target := filepath.Join(system.ExpandPath(file.Target), file.Name)

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
	target := filepath.Join(system.ExpandPath(file.Target), file.Name)

	if err := os.Remove(target); err != nil {
		log.Fatalf("error deleting file: %v", err)
	}

	log.Infof("File deleted: %s", target)
	return nil
}

func createFile(file types.File) error {

	log.Debugf("Creating file type: %v", file)

	targetPath := filepath.Join(system.ExpandPath(file.Target), file.Name)

	log.Debugf("Creating file: %s", targetPath)

	targetDir := filepath.Dir(targetPath)

	log.Debugf("Creating file dir: %s", targetDir)

	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating target directory: %v", err)
	}

	// Create the file
	f, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Errorf("error closing file: %v", err)
		}
	}(f)

	// Write the content to the file
	_, err = f.WriteString(file.Content)
	if err != nil {
		return fmt.Errorf("error writing content to file: %v", err)
	}

	log.Infof("File created and content written: %s", file.Target)

	if err := applyFileAttributes(targetPath, file); err != nil {
		return fmt.Errorf("error applying file attributes: %v", err)
	}

	return nil
}

func chmodFile(file types.File) error {
	target := filepath.Join(system.ExpandPath(file.Target), file.Name)

	if err := os.Chmod(target, os.FileMode(file.Mode)); err != nil {
		log.Fatalf("error changing file permissions: %v", err)
	}

	log.Infof("File permissions changed: %s (mode: %o)", target, file.Mode)
	return nil
}

func chownFile(file types.File) error {
	target := filepath.Join(system.ExpandPath(file.Target), file.Name)

	if file.Owner != "" {
		uid, err := system.LookupUID(file.Owner)
		if err != nil {
			log.Fatalf("error looking up owner UID: %v", err)
		}
		if err := os.Chown(target, uid, -1); err != nil {
			log.Fatalf("error changing file owner: %v", err)
		}
	}

	if file.Group != "" {
		gid, err := system.LookupGID(file.Group)
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
	target := filepath.Join(system.ExpandPath(file.Target), file.Name)

	if file.Group != "" {
		gid, err := system.LookupGID(file.Group)
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
	target := system.ExpandPath(file.Target)

	if err := os.Symlink(source, target); err != nil {
		log.Fatalf("error creating symlink: %v", err)
	}

	log.Infof("Symlink created: %s -> %s", source, target)
	return nil
}

func copyDirectory(dir types.Directory, blueprintDir string, initConfig *types.InitConfig) error {
	source := filepath.Join(blueprintDir, dir.Source, dir.Name)
	target := filepath.Join(system.ExpandPath(dir.Target), dir.Name)

	// Create the target directory if it doesn't exist
	if err := os.MkdirAll(target, os.ModePerm); err != nil {
		log.Fatalf("error creating target directory: %v", err)
	}

	if err := system.CopyDirectory(source, target, dir.Elevated, initConfig.Variables.Flags.Interactive); err != nil {
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
	target := filepath.Join(system.ExpandPath(dir.Target), dir.Name)

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
	target := filepath.Join(system.ExpandPath(dir.Target), dir.Name)

	if err := os.RemoveAll(target); err != nil {
		log.Fatalf("error deleting directory: %v", err)
	}

	log.Infof("Directory deleted: %s", target)
	return nil
}

func createDirectory(dir types.Directory) error {
	target := filepath.Join(system.ExpandPath(dir.Target), dir.Name)

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
	target := filepath.Join(system.ExpandPath(dir.Target), dir.Name)

	if err := os.Chmod(target, os.FileMode(dir.Mode)); err != nil {
		log.Fatalf("error changing directory permissions: %v", err)
	}

	log.Infof("Directory permissions changed: %s (mode: %o)", target, dir.Mode)
	return nil
}

func chownDirectory(dir types.Directory) error {
	target := filepath.Join(system.ExpandPath(dir.Target), dir.Name)

	if dir.Owner != "" {
		uid, err := system.LookupUID(dir.Owner)
		if err != nil {
			log.Fatalf("error looking up owner UID: %v", err)
		}
		if err := os.Chown(target, uid, -1); err != nil {
			log.Fatalf("error changing directory owner: %v", err)
		}
	}

	if dir.Group != "" {
		gid, err := system.LookupGID(dir.Group)
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
	target := filepath.Join(system.ExpandPath(dir.Target), dir.Name)

	if dir.Group != "" {
		gid, err := system.LookupGID(dir.Group)
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
	target := system.ExpandPath(dir.Target)

	if err := os.Symlink(source, target); err != nil {
		log.Fatalf("error creating symlink: %v", err)
	}

	log.Infof("Symlink created: %s -> %s", source, target)
	return nil
}

func applyFileAttributes(targetPath string, file types.File) error {
	if file.Mode != 0 {
		if err := os.Chmod(targetPath, os.FileMode(file.Mode)); err != nil {
			return fmt.Errorf("error changing file permissions: %v", err)
		}
	}

	if file.Owner != "" || file.Group != "" {
		if err := chownFile(file); err != nil {
			return fmt.Errorf("error changing file owner/group: %v", err)
		}
	}

	return nil
}

func applyDirectoryAttributes(dir types.Directory) error {
	target := filepath.Join(system.ExpandPath(dir.Target), dir.Name)

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

func isURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func determineSourceAndTargetPaths(file types.File, blueprintDir string) (string, string, error) {
	var sourcePath, targetPath string

	// Determine source path
	if isURL(file.Source) {
		log.Fatalf("Source is URL, should not be URL at this point - URL Check/Download has failed")
	} else if file.Content != "" {
		log.Debug("File Content present, sourcePath will be empty")
		sourcePath = ""
	} else {
		sourcePath = filepath.Join(blueprintDir, file.Source, file.Name)
	}

	// Determine target path
	targetPath = system.ExpandPath(file.Target)
	if !strings.HasSuffix(targetPath, string(os.PathSeparator)) {
		targetPath = filepath.Join(filepath.Dir(targetPath), filepath.Base(targetPath))
	} else {
		targetPath = filepath.Join(targetPath, file.Name)
	}

	return sourcePath, targetPath, nil
}
