package processors

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
)

const (
	// A plain file with no declared mode is world-readable, as a config file
	// normally is. A rendered template is not: the whole point of exposing
	// credentials to templates is that one can render a .netrc or a gh config,
	// and those must never exist world-readable, not even for an instant.
	defaultFileMode     os.FileMode = 0644
	defaultTemplateMode os.FileMode = 0600

	defaultDirMode os.FileMode = 0755
	// A file no one else may read is not much protected by a directory everyone
	// may list, so a private file gets a private parent.
	privateDirMode os.FileMode = 0700
)

// ProcessFiles handles file, directory, and template operations from blueprint data.
// It supports create, delete, copy, append, symlink, and template rendering actions
// with optional profile filtering and interactive diff-based overwrite prompts.
func ProcessFiles(blueprintData []byte, blueprintDir string, format string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	log.Debugf("Processing files from blueprint")

	files, dirs, templates, err := resolveAndFilterFileData(blueprintData, blueprintDir, format, initConfig)
	if err != nil {
		return err
	}

	// One tracker across all three loops: files, directories, and templates
	// share the processor's single lane, and a second tracker would reset its
	// done/total counts.
	track := newProgress(types.BlueprintTypeFiles)

	if err := processFiles(files, blueprintDir, osInfo, track); err != nil {
		return fmt.Errorf("error processing files: %w", err)
	}

	if err := processDirectories(dirs, blueprintDir, initConfig, track); err != nil {
		return fmt.Errorf("error processing directories: %w", err)
	}

	if err := processTemplates(templates, blueprintDir, osInfo, initConfig, track); err != nil {
		return fmt.Errorf("error processing templates: %w", err)
	}

	return nil
}

// resolveAndFilterFileData unmarshals blueprint data, resolves all imports for files,
// directories, and templates, then filters each by active profiles.
func resolveAndFilterFileData(blueprintData []byte, blueprintDir string, format string, initConfig *types.InitConfig) ([]types.File, []types.Directory, []types.File, error) {
	var fileData types.FileData
	if err := helpers.DecodeBlueprintInto(blueprintData, format, types.BlueprintTypeFiles,
		helpers.TreeSchemaVersion(initConfig), &fileData); err != nil {
		return nil, nil, nil, fmt.Errorf("error unmarshaling file blueprint data: %w", err)
	}

	allFiles, err := processFileImports(fileData.Files, blueprintDir, format, helpers.TreeSchemaVersion(initConfig))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error processing file imports: %w", err)
	}

	allDirs, err := processDirectoryImports(fileData.Directories, blueprintDir, format, helpers.TreeSchemaVersion(initConfig))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error processing directory imports: %w", err)
	}

	allTemplates, err := processTemplateImports(fileData.Templates, blueprintDir, format, helpers.TreeSchemaVersion(initConfig))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error processing template imports: %w", err)
	}

	profiles := initConfig.Variables.Flags.Profiles

	filteredFiles := helpers.FilterByProfiles(allFiles, profiles)
	log.Debugf("Filtering files: %d total, %d matching active profiles %v", len(allFiles), len(filteredFiles), profiles)

	filteredDirs := helpers.FilterByProfiles(allDirs, profiles)
	log.Debugf("Filtering directories: %d total, %d matching active profiles %v", len(allDirs), len(filteredDirs), profiles)

	filteredTemplates := helpers.FilterByProfiles(allTemplates, profiles)
	log.Debugf("Filtering templates: %d total, %d matching active profiles %v", len(allTemplates), len(filteredTemplates), profiles)

	return filteredFiles, filteredDirs, filteredTemplates, nil
}

// One file failing does not stop the rest: the failure goes to the ledger,
// which puts it in the run's exit code, and processing continues.
func processFiles(files []types.File, blueprintDir string, osInfo *types.OSInfo, track *progress) error {
	total := 0
	for _, file := range files {
		if len(file.Names) > 0 {
			total += len(file.Names)
		} else {
			total++
		}
	}
	track.expect("", total)

	run := func(file types.File) {
		started := time.Now()
		switch err := processFile(file, blueprintDir, osInfo); {
		case err != nil:
			recordFailure("files", file.Name, err)
			track.item("", file.Name, file.Action, types.StatusFailed, err.Error(), time.Since(started))
		case system.IsDryRun():
			track.item("", file.Name, file.Action, types.StatusPlanned, "dry-run", 0)
		default:
			track.itemIdentity("", file.Name, file.Action, types.StatusOK, "", time.Since(started), fileJournalIdentity(file, blueprintDir))
		}
	}

	for _, file := range files {
		if len(file.Names) > 0 {
			for _, name := range file.Names {
				fileWithName := file
				fileWithName.Name = name
				run(fileWithName)
			}
		} else {
			run(file)
		}
	}
	return nil
}

// fileActionConsumesContent reports whether an action reads file content at
// all. The metadata actions act on a target that already exists; the empty
// action defaults to create-from-content later in processFile.
func fileActionConsumesContent(action string) bool {
	switch action {
	case types.FileActionChmod, types.FileActionChown, types.FileActionChgrp, types.FileActionDelete:
		return false
	}
	return true
}

func processFile(file types.File, blueprintDir string, osInfo *types.OSInfo) error {

	log.Debugf("Processing file: %s", file.Name)

	// Only the actions that consume content need it. chmod/chown/chgrp/delete
	// operate on an existing target — demanding content or source for them
	// broke the documented metadata examples (docs/blueprints/files.md), which
	// pair a copy entry with a follow-up chmod/chown entry. The validator
	// (internal/validate/components.go) has always gated this by action.
	if file.Content == "" && file.Source == "" && fileActionConsumesContent(file.Action) {
		return fmt.Errorf("either Content or Source must be provided for file %s (action %q)", file.Name, file.Action)
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
				log.Errorf("Error removing temporary directory %s: %v", tempDir, removeErr)
			}
		}()

		log.Debug("Downloading Source File")
		name := file.Name
		if name == "" {
			name = filepath.Base(file.Source)
		}
		downloadPath := filepath.Join(tempDir, name)
		// A URL source without a declared digest is trusted on nothing but the
		// TLS connection that served it. Warn now; a later major refuses.
		if file.Sha256 == "" {
			log.Warnf("File %s downloads %s with no sha256 declared — the content is unpinned. "+
				"Add sha256 to the entry; a future major version will refuse this.", file.Name, file.Source)
		}
		err = system.DownloadFileWithChecksum(file.Source, downloadPath, false, file.Sha256)
		if err != nil {
			return fmt.Errorf("error downloading file: %v", err)
		}

		log.Debug("Setting File Source and Name")
		file.Source = filepath.Dir(downloadPath)
		file.Name = name
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

	if system.IsDryRun() {
		log.Infof("[DRY-RUN] Would %s file: %s (source: %s, target: %s)", file.Action, file.Name, sourcePath, targetPath)
		return nil
	}

	switch file.Action {
	case "copy":
		log.Debugf("Copying file: %s to %s (elevated: %v)", sourcePath, targetPath, file.Elevated)
		if err := system.CopyFile(sourcePath, targetPath, file.Elevated, osInfo); err != nil {
			return err
		}
		// A copy used to keep the source's mode and drop the declared one, so a
		// blueprint copying a secret with mode: 0600 landed it world-readable and
		// said nothing. The declared mode is the intent regardless of the action
		// that put the file there.
		return applyFileAttributes(targetPath, file)
	case "move":
		log.Debugf("Moving file: %s to %s", sourcePath, targetPath)
		return moveFile(sourcePath, targetPath)
	case "delete":
		log.Debugf("Deleting file: %s", targetPath)
		return deleteFile(targetPath)
	case "create":
		log.Debugf("Creating file: %s", targetPath)
		return createFile(file, targetPath)
	case "chmod":
		log.Debugf("Changing file permissions: %s", targetPath)
		return chmodFile(file, targetPath)
	case "chown":
		log.Debugf("Changing file owner: %s", targetPath)
		return chownFile(file, targetPath)
	case "chgrp":
		log.Debugf("Changing file group: %s", targetPath)
		return chgrpFile(file, targetPath)
	case "symlink":
		log.Debugf("Symlinking file: %s to %s", sourcePath, targetPath)
		return symlinkFile(sourcePath, targetPath)
	default:
		return fmt.Errorf("unsupported action for file: %s", file.Action)
	}
}

func moveFile(source, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), defaultDirMode); err != nil {
		return fmt.Errorf("error creating target directory: %w", err)
	}

	if err := os.Rename(source, target); err != nil { // #nosec G703 -- target path is operator-supplied blueprint/config input; containment added in PR8
		// A move already carried out on an earlier run has no source left. That is
		// the state the blueprint asked for, so it is not a failure.
		if os.IsNotExist(err) {
			if _, statErr := os.Lstat(target); statErr == nil { // #nosec G703 -- target path is operator-supplied blueprint/config input; containment added in PR8
				log.Infof("File already moved: %s", target)
				return nil
			}
		}
		return fmt.Errorf("error moving file: %w", err)
	}

	log.Infof("File moved: %s -> %s", source, target)
	return nil
}

func deleteFile(target string) error {
	if err := os.Remove(target); err != nil { // #nosec G703 -- target path is operator-supplied blueprint/config input; containment added in PR8
		if os.IsNotExist(err) {
			log.Debugf("File already absent: %s", target)
			return nil
		}
		return fmt.Errorf("error deleting file: %w", err)
	}

	log.Infof("File deleted: %s", target)
	return nil
}

func createFile(file types.File, targetPath string) error {
	log.Debugf("Creating file: %s", targetPath)

	mode := defaultFileMode
	if file.Mode.IsSet() {
		mode = file.Mode.OSMode()
	}

	dirMode := defaultDirMode
	if mode.Perm()&0o077 == 0 {
		dirMode = privateDirMode
	}

	targetDir := filepath.Dir(targetPath)
	log.Debugf("Creating file dir: %s", targetDir)
	if err := os.MkdirAll(targetDir, dirMode); err != nil { // #nosec G703 -- target path is operator-supplied blueprint/config input; containment added in PR8
		return fmt.Errorf("error creating target directory: %v", err)
	}

	// The mode is carried into the open so the content is never readable by anyone
	// the blueprint did not name — a later chmod would leave a window in which a
	// rendered credential sat on disk world-readable.
	f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY, mode) // #nosec G304 G703 -- path is operator-supplied blueprint/config input; containment added in PR8
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Errorf("error closing file: %v", err)
		}
	}(f)

	// O_CREATE's mode applies only to a file that did not exist. A rerun over a
	// file left too permissive by an earlier one still has to narrow it, and has
	// to do so before the content goes back in.
	if info, err := f.Stat(); err == nil && info.Mode().Perm() != mode.Perm() {
		if err := f.Chmod(mode); err != nil {
			return fmt.Errorf("error setting file permissions: %v", err)
		}
	}

	if err := f.Truncate(0); err != nil {
		return fmt.Errorf("error truncating file: %v", err)
	}

	_, err = f.WriteString(file.Content)
	if err != nil {
		return fmt.Errorf("error writing content to file: %v", err)
	}

	log.Infof("File created and content written: %s", targetPath)

	if err := applyFileAttributes(targetPath, file); err != nil {
		return fmt.Errorf("error applying file attributes: %v", err)
	}

	return nil
}

func chmodFile(file types.File, target string) error {
	// Without this the zero value would be applied literally, and a chmod entry
	// that forgot its mode would take every permission off the file.
	if !file.Mode.IsSet() {
		return fmt.Errorf("no mode declared for chmod of %s: add mode: \"0644\" to the file entry", target)
	}

	if err := os.Chmod(target, file.Mode.OSMode()); err != nil { // #nosec G703 -- target path is operator-supplied blueprint/config input; containment added in PR8
		return fmt.Errorf("error changing file permissions: %w", err)
	}

	log.Infof("File permissions changed: %s (mode: %s)", target, file.Mode)
	return nil
}

func chownFile(file types.File, target string) error {
	if file.Owner != "" {
		uid, err := system.LookupUID(file.Owner)
		if err != nil {
			return fmt.Errorf("error looking up owner UID: %w", err)
		}
		if err := os.Chown(target, uid, -1); err != nil { // #nosec G703 -- target path is operator-supplied blueprint/config input; containment added in PR8
			return fmt.Errorf("error changing file owner: %w", err)
		}
	}

	if file.Group != "" {
		gid, err := system.LookupGID(file.Group)
		if err != nil {
			return fmt.Errorf("error looking up group GID: %w", err)
		}
		if err := os.Chown(target, -1, gid); err != nil { // #nosec G703 -- target path is operator-supplied blueprint/config input; containment added in PR8
			return fmt.Errorf("error changing file group: %w", err)
		}
	}

	log.Infof("File owner/group changed: %s (owner: %s, group: %s)", target, file.Owner, file.Group)
	return nil
}

// chgrpFile is the group half of chownFile: same lookup, same chown call,
// owner left untouched.
func chgrpFile(file types.File, target string) error {
	groupOnly := file
	groupOnly.Owner = ""
	return chownFile(groupOnly, target)
}

func symlinkFile(source, target string) error {
	return ensureSymlink(source, target)
}

// ensureSymlink brings target to the state "a symlink pointing at source",
// whatever it happens to be in now. Creating one unconditionally fails with
// EEXIST on the second run of a blueprint, and one failing file aborts the
// whole run.
func ensureSymlink(source, target string) error {
	info, err := os.Lstat(target) // #nosec G703 -- target path is operator-supplied blueprint/config input; containment added in PR8
	switch {
	case err == nil && info.Mode()&os.ModeSymlink == 0:
		return fmt.Errorf("cannot create symlink at %s: a regular file or directory already exists there", target)
	case err == nil:
		existing, readErr := os.Readlink(target)
		if readErr != nil {
			return fmt.Errorf("error reading existing symlink %s: %w", target, readErr)
		}
		if existing == source {
			log.Debugf("Symlink already present: %s -> %s", target, source)
			return nil
		}
		log.Infof("Replacing symlink %s: pointed at %s, should point at %s", target, existing, source)
		if err := os.Remove(target); err != nil { // #nosec G703 -- target path is operator-supplied blueprint/config input; containment added in PR8
			return fmt.Errorf("error removing existing symlink %s: %w", target, err)
		}
	case !os.IsNotExist(err):
		return fmt.Errorf("error inspecting symlink target %s: %w", target, err)
	}

	if err := os.MkdirAll(filepath.Dir(target), defaultDirMode); err != nil { // #nosec G703 -- target path is operator-supplied blueprint/config input; containment added in PR8
		return fmt.Errorf("error creating symlink parent directory: %w", err)
	}

	if err := os.Symlink(source, target); err != nil {
		return fmt.Errorf("error creating symlink: %w", err)
	}

	log.Infof("Symlink created: %s -> %s", target, source)
	return nil
}

func applyFileAttributes(targetPath string, file types.File) error {
	if file.Mode.IsSet() {
		if err := os.Chmod(targetPath, file.Mode.OSMode()); err != nil { // #nosec G703 -- target path is operator-supplied blueprint/config input; containment added in PR8
			return fmt.Errorf("error changing file permissions: %v", err)
		}
	}

	if file.Owner != "" || file.Group != "" {
		if err := chownFile(file, targetPath); err != nil {
			return fmt.Errorf("error changing file owner/group: %v", err)
		}
	}

	return nil
}

func isURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// resolveTargetPath resolves a file entry's target to the one path every action
// operates on. A target ending in a separator names the directory the file goes
// into; anything else is the file's own path, the rename form documented in
// docs/blueprints/files.md.
//
// The trailing separator is read off the blueprint's own value: ExpandPath sends
// the "~/" form through filepath.Join, which drops it.
//
// A target that is already a directory takes the file inside it even without the
// separator, because the alternative is truncating a directory, which no
// blueprint can have meant.
func resolveTargetPath(target, name string) string {
	expanded := system.ExpandPath(target)

	if strings.HasSuffix(target, "/") || strings.HasSuffix(target, string(os.PathSeparator)) {
		return filepath.Join(expanded, name)
	}

	if info, err := os.Stat(expanded); err == nil && info.IsDir() {
		return filepath.Join(expanded, name)
	}

	return filepath.Clean(expanded)
}

func determineSourceAndTargetPaths(file types.File, blueprintDir string) (string, string, error) {
	var sourcePath string

	// Determine source path
	if isURL(file.Source) {
		return "", "", fmt.Errorf("source is URL, should not be URL at this point - URL check/download has failed")
	} else if file.Content != "" {
		log.Debug("File Content present, sourcePath will be empty")
		sourcePath = ""
	} else if filepath.IsAbs(file.Source) {
		// An absolute source stands on its own: joining it under blueprintDir
		// would silently produce <blueprintDir>/<abs path>, a path that never
		// exists. URL downloads land here too, via their absolute temp dir.
		sourcePath = filepath.Join(file.Source, file.Name)
	} else {
		sourcePath = filepath.Join(blueprintDir, file.Source, file.Name)
	}

	return sourcePath, resolveTargetPath(file.Target, file.Name), nil
}

func processFileImports(items []types.File, blueprintDir string, format string, treeVersion int) ([]types.File, error) {
	return helpers.ResolveImports(items, blueprintDir,
		func(item types.File) string { return item.Import },
		func(data []byte, fileFormat string) ([]types.File, error) {
			var d types.FileData
			if err := helpers.DecodeBlueprintInto(data, fileFormat, types.BlueprintTypeFiles, treeVersion, &d); err != nil {
				return nil, err
			}
			return d.Files, nil
		}, format)
}
