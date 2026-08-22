package processors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/go-git/go-git/v5"
)

// GetBlueprints retrieves blueprints from a Git repository or local location.
// If Git options are configured in the init file, it clones or updates the
// blueprint repository. Otherwise, it returns the configured local location.
// Returns the path to the blueprints directory or an error if retrieval fails.
func GetBlueprints(initConfig *types.InitConfig) (string, error) {
	// Check if GitOptions is provided in the init configuration
	if initConfig.Init.Git != nil {
		gitOpts := initConfig.Init.Git

		// A repository URL used as -i has already been cloned so its manifest can
		// select this init file. If that checkout has the same origin the init
		// declares, it is already the requested blueprint repository. Cloning it a
		// second time wastes the network round trip (and can hang in go-git before
		// the dashboard shows useful progress).
		if sourceRoot, ok := matchingRepositoryRoot(initConfig.Init.Location, gitOpts.URL); ok {
			log.Infof("Using already-resolved blueprint repository: %s", sourceRoot)
			return initConfig.Init.Location, nil
		}

		// Dry-run has to bail out before any of the git handling below: cloning and
		// pulling touch disk and network directly instead of going through
		// system.RunCommand, which is where dry-run is otherwise enforced.
		if system.IsDryRun() {
			log.Infof("[DRY-RUN] Would sync blueprint repository %s into %s", gitOpts.URL, gitOpts.Target)
			return gitOpts.Target, nil
		}

		if _, err := os.Stat(gitOpts.Target); err == nil {
			// Refuse rather than reclaim the path: rwr runs unattended, and a
			// mistyped target (~/dotfiles) used to be silently deleted here.
			if _, err := git.PlainOpen(gitOpts.Target); err != nil {
				entries, readErr := os.ReadDir(gitOpts.Target)
				if readErr != nil || len(entries) != 0 {
					return "", fmt.Errorf("blueprint target %q exists but is not a git repository and is not empty; move or remove it yourself, or point blueprints.git.target at a different path", gitOpts.Target)
				}
				// Older RWR builds created the clone target during init, then
				// rejected their own empty directory here. Removing only a verified
				// empty target repairs that state without risking user content.
				if err := os.Remove(gitOpts.Target); err != nil {
					return "", fmt.Errorf("removing empty blueprint target left by initialization: %w", err)
				}
				log.Infof("Removed empty blueprint target left by an earlier initialization: %s", gitOpts.Target)
			}
		}

		// Now either clone fresh or update existing
		_, err := git.PlainOpen(gitOpts.Target)
		if err != nil {
			// Repository doesn't exist or was removed - clone it
			log.Debugf("Cloning blueprint repository to %s", gitOpts.Target)
			if err := os.MkdirAll(filepath.Dir(gitOpts.Target), 0755); err != nil { // #nosec G301 -- TODO: blueprint-target directory; create with the requested mode
				return "", fmt.Errorf("error creating parent directory: %v", err)
			}
			err = helpers.HandleGitClone(*gitOpts, initConfig)
			if err != nil {
				return "", fmt.Errorf("error cloning blueprint repository: %v", err)
			}
			log.Debugf("Blueprint repository cloned successfully")
		} else {
			// Repository exists and is valid - update it
			log.Debugf("Updating existing blueprint repository at %s", gitOpts.Target)
			err = helpers.CheckAndUpdateRemoteURL(gitOpts.Target, gitOpts.URL)
			if err != nil {
				return "", fmt.Errorf("error checking/updating remote URL: %v", err)
			}

			if gitOpts.Update {
				err = helpers.HandleGitPull(*gitOpts, initConfig)
				if err != nil {
					return "", fmt.Errorf("error updating blueprint repository: %v", err)
				}
			}
			log.Debugf("Blueprint repository updated successfully")
		}

		// Verify the blueprints directory exists and has content
		filesInfo, err := os.ReadDir(gitOpts.Target)
		if err != nil {
			return "", fmt.Errorf("error reading blueprints directory: %v", err)
		}
		if len(filesInfo) == 0 {
			return "", fmt.Errorf("blueprints directory is empty: %s", gitOpts.Target)
		}

		log.Debugf("Using blueprint location: %s", gitOpts.Target)
		return gitOpts.Target, nil
	}

	// If GitOptions is not provided, use the default location from initConfig
	location := initConfig.Init.Location
	log.Debugf("Using default blueprint location: %s", location)
	return location, nil
}

func matchingRepositoryRoot(location, desiredURL string) (string, bool) {
	dir := filepath.Clean(location)
	for {
		repo, err := git.PlainOpen(dir)
		if err == nil {
			remote, remoteErr := repo.Remote("origin")
			if remoteErr != nil || len(remote.Config().URLs) == 0 {
				return "", false
			}
			return dir, sameRepositoryURL(remote.Config().URLs[0], desiredURL)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func sameRepositoryURL(a, b string) bool {
	normalize := func(value string) string {
		value = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSpace(value), "/"), ".git")
		value = strings.TrimPrefix(value, "git@github.com:")
		value = strings.TrimPrefix(value, "https://github.com/")
		return strings.ToLower(value)
	}
	return normalize(a) == normalize(b)
}

// defaultRunOrder is the order blueprint processors run in when the init file
// does not specify one.
//
// packageManagers is deliberately absent: it is not dispatched from the blueprint
// loop at all, it runs ahead of it from initConfig.PackageManagers. Listing it
// here only produced an "Unknown processor" warning.
//
// users is present because it was previously missing, which meant `rwr all` never
// created users unless the init file hand-wrote its own order - while
// `rwr run users` worked, making the omission easy to miss.
var defaultRunOrder = []string{
	types.BlueprintTypeRepositories,
	types.BlueprintTypePackages,
	types.BlueprintTypeSSHKeys,
	types.BlueprintTypeUsers,
	types.BlueprintTypeFiles,
	types.BlueprintTypeFonts,
	types.BlueprintTypeServices,
	types.BlueprintTypeGit,
	types.BlueprintTypeScripts,
	types.BlueprintTypeConfiguration,
}

// GetBlueprintRunOrder determines the order in which blueprint processors should run.
// If a custom order is specified in the init configuration, it uses that order.
// Otherwise, it returns defaultRunOrder.
// Returns a slice of processor names in execution order.
func GetBlueprintRunOrder(initConfig *types.InitConfig) ([]string, error) {
	var runOrder []string

	if initConfig.Init.Order != nil {
		for _, item := range initConfig.Init.Order {
			if str, ok := item.(string); ok {
				runOrder = append(runOrder, str)
			} else if subOrder, ok := item.(map[string]interface{}); ok {
				// Go randomizes map iteration, so a nested order entry produced a
				// different sequence on each run - the one thing an explicit order
				// is written to control. Sort so the result is at least stable;
				// a map cannot preserve authored order, which is why the sorted
				// keys are logged loudly enough to notice.
				processors := make([]string, 0, len(subOrder))
				for processor := range subOrder {
					processors = append(processors, processor)
				}
				sort.Strings(processors)
				if len(processors) > 1 {
					log.Warnf("Nested order entry lists %d processors as a mapping; running them in sorted order %v. Use a flat list to control the sequence.", len(processors), processors)
				}
				runOrder = append(runOrder, processors...)
			}
		}
	} else {
		runOrder = append(runOrder, defaultRunOrder...)
	}

	log.Debugf("Blueprint run order: %v", runOrder)
	return runOrder, nil
}

// GetBlueprintFileOrder builds a mapping of processors to their blueprint files.
// It processes files in the specified order and optionally scans for additional
// files if runOnlyListed is false. The function walks directory trees and identifies
// which processor each blueprint file belongs to based on the file path structure.
// Returns a map of processor names to slices of blueprint file paths, or an error
// if file scanning fails.
func GetBlueprintFileOrder(blueprintDir string, order []interface{}, runOnlyListed bool, initConfig *types.InitConfig) (map[string][]string, error) {
	fileOrder := make(map[string][]string)

	log.Debugf("Getting blueprint file order from directory: %s", blueprintDir)

	// Helper function to extract processor type from path
	getProcessorType := func(path string) string {
		parts := strings.Split(path, string(os.PathSeparator))
		// Look for known processor types in the path
		for _, part := range parts {
			switch part {
			case types.BlueprintTypePackages, types.BlueprintTypeRepositories, types.BlueprintTypeFiles, types.BlueprintTypeServices, types.BlueprintTypeUsers,
				types.BlueprintTypeGit, types.BlueprintTypeScripts, types.BlueprintTypeSSHKeys, types.BlueprintTypeFonts, types.BlueprintTypeConfiguration:
				return part
			}
		}
		return filepath.Dir(path)
	}

	isKnownProcessor := func(processor string) bool {
		switch processor {
		case types.BlueprintTypePackages, types.BlueprintTypeRepositories, types.BlueprintTypeFiles, types.BlueprintTypeServices, types.BlueprintTypeUsers,
			types.BlueprintTypeGit, types.BlueprintTypeScripts, types.BlueprintTypeSSHKeys, types.BlueprintTypeFonts, types.BlueprintTypeConfiguration:
			return true
		}
		return false
	}

	// The run loop only executes buckets named after a processor. A file whose
	// path names none is typed by its content instead - the flattened and
	// minimal_files layouts the examples ship have no processor directories at
	// all, and used to land in a dead bucket and exit 0 having executed
	// nothing. A multi-type file (minimal_files' all_in_one) routes to every
	// type it declares; the dispatch subsets it per processor. A file whose
	// content matches nothing gets a loud statement that it will not run.
	routeByPath := func(absPath, relPath string) []string {
		processor := getProcessorType(relPath)
		if isKnownProcessor(processor) {
			return []string{processor}
		}
		if detected := detectBlueprintTypesFromContent(absPath, initConfig); len(detected) > 0 {
			log.Debugf("Blueprint file %s routed to %v by its content", relPath, detected)
			return detected
		}
		log.Warnf("Blueprint file %s is not under a recognized processor directory and its content matches no blueprint type; it will NOT be executed. "+
			"Move it under one of: packages/, repositories/, files/, services/, users/, git/, scripts/, ssh_keys/, fonts/, configuration/ - or give it top-level blueprint keys.", relPath)
		return []string{processor}
	}

	// The init file configures the run, bootstrap is dispatched separately
	// (findBootstrapFile), and a root manifest names configurations - none is
	// a processor blueprint, so none belongs in a processor bucket, nor in
	// the unrouted warning above. The manifest matters doubly: its top-level
	// configurations key would otherwise content-route it to the
	// configuration processor.
	isReservedFile := func(path string) bool {
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		return base == "init" || base == "bootstrap" || base == "manifest"
	}

	// Process ordered items first
	for _, item := range order {
		if str, ok := item.(string); ok {
			fullPath := filepath.Join(blueprintDir, str)

			if info, err := os.Stat(fullPath); err == nil {
				if info.IsDir() {
					// Process directory
					err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
						if err != nil {
							return err
						}
						// Version-control and other dot directories hold no
						// blueprints, and .git holds a great many files.
						if info.IsDir() && path != fullPath && strings.HasPrefix(info.Name(), ".") {
							return filepath.SkipDir
						}
						if !info.IsDir() && helpers.IsBlueprintFile(path) && !isReservedFile(path) {
							relPath, err := filepath.Rel(blueprintDir, path)
							if err != nil {
								return err
							}
							for _, processor := range routeByPath(path, relPath) {
								fileOrder[processor] = append(fileOrder[processor], relPath)
								log.Debugf("Added file to processor %s: %s", processor, relPath)
							}
						}
						return nil
					})
					if err != nil {
						return nil, err
					}
				} else {
					// Single file
					relPath, err := filepath.Rel(blueprintDir, fullPath)
					if err != nil {
						return nil, err
					}
					processor := getProcessorType(relPath)
					fileOrder[processor] = append(fileOrder[processor], relPath)
					log.Debugf("Added single file to processor %s: %s", processor, relPath)
				}
			}
		}
	}

	// If not runOnlyListed, scan for additional files
	if !runOnlyListed {
		err := filepath.Walk(blueprintDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// Same dot-directory rule as the ordered walk above.
			if info.IsDir() && path != blueprintDir && strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			if !info.IsDir() && helpers.IsBlueprintFile(path) && !isReservedFile(path) {
				relPath, err := filepath.Rel(blueprintDir, path)
				if err != nil {
					return err
				}
				for _, processor := range routeByPath(path, relPath) {
					if _, exists := fileOrder[processor]; !exists {
						fileOrder[processor] = []string{relPath}
					} else if !helpers.Contains(fileOrder[processor], relPath) {
						fileOrder[processor] = append(fileOrder[processor], relPath)
					}
					log.Debugf("Added additional file to processor %s: %s", processor, relPath)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// Log final order
	for processor, files := range fileOrder {
		log.Debugf("Processor %s files:", processor)
		for _, file := range files {
			log.Debugf("  - %s", file)
		}
	}

	return fileOrder, nil
}

// blueprintKeyToType maps a file's top-level keys to the processor that reads
// them, for content-based routing when the path names no processor directory.
var blueprintKeyToType = map[string]string{
	"packages":       types.BlueprintTypePackages,
	"repositories":   types.BlueprintTypeRepositories,
	"files":          types.BlueprintTypeFiles,
	"templates":      types.BlueprintTypeFiles,
	"directories":    types.BlueprintTypeFiles,
	"services":       types.BlueprintTypeServices,
	"git":            types.BlueprintTypeGit,
	"scripts":        types.BlueprintTypeScripts,
	"ssh_keys":       types.BlueprintTypeSSHKeys,
	"fonts":          types.BlueprintTypeFonts,
	"users":          types.BlueprintTypeUsers,
	"groups":         types.BlueprintTypeUsers,
	"configurations": types.BlueprintTypeConfiguration,
}

// detectBlueprintTypesFromContent types a blueprint by its top-level keys,
// returning every matched type in run-order-stable form. Templates are
// resolved leniently first - content routing happens before the run renders
// anything for real.
func detectBlueprintTypesFromContent(path string, initConfig *types.InitConfig) []string {
	top, _, err := decodeTopLevel(path, initConfig)
	if err != nil {
		return nil
	}

	seen := map[string]bool{}
	var detected []string
	for key := range top {
		blueprintType, known := blueprintKeyToType[key]
		if known && !seen[blueprintType] {
			seen[blueprintType] = true
			detected = append(detected, blueprintType)
		}
	}
	sort.Strings(detected)
	return detected
}

// decodeTopLevel reads a blueprint's top-level mapping leniently.
func decodeTopLevel(path string, initConfig *types.InitConfig) (map[string]interface{}, string, error) {
	format, err := helpers.FormatForPath(path)
	if err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(path) // #nosec G304 -- read-only inspection of the operator's own blueprint tree
	if err != nil {
		return nil, "", err
	}
	resolved, err := helpers.ResolveTemplateForValidation(data, initConfig.Variables)
	if err != nil {
		return nil, "", err
	}

	var top map[string]interface{}
	if err := helpers.UnmarshalBlueprint(resolved, format, &top); err != nil {
		return nil, "", err
	}
	return top, format, nil
}

// subsetForProcessor prepares one processor's view of a blueprint. A
// single-type file passes through untouched, so strict decode keeps its
// unknown-key typo protection. A multi-type file (minimal_files'
// all_in_one.yaml) is cut down to the keys this processor reads - plus
// schema_version - re-encoded as JSON; a top-level key belonging to no type at
// all is an error rather than something to silently drop.
func subsetForProcessor(resolved []byte, format, processor string) ([]byte, string, error) {
	var top map[string]interface{}
	if err := helpers.UnmarshalBlueprint(resolved, format, &top); err != nil {
		return nil, "", err
	}

	typesPresent := map[string]bool{}
	for key := range top {
		if blueprintType, ok := blueprintKeyToType[key]; ok {
			typesPresent[blueprintType] = true
		}
	}
	if len(typesPresent) <= 1 {
		return resolved, format, nil
	}

	subset := map[string]interface{}{}
	for key, value := range top {
		blueprintType, known := blueprintKeyToType[key]
		switch {
		case known && blueprintType == processor:
			subset[key] = value
		case known:
			// Another processor's section; it gets its own dispatch.
		case key == "schema_version" || key == "variables":
			subset[key] = value
		default:
			return nil, "", fmt.Errorf("multi-type blueprint declares unknown top-level key %q", key)
		}
	}

	out, err := json.Marshal(subset)
	if err != nil {
		return nil, "", fmt.Errorf("error re-encoding multi-type blueprint for %s: %w", processor, err)
	}
	return out, types.FormatJSON, nil
}
