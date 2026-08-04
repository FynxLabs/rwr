package processors

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
)

// RunBootstrap is the standalone `rwr bootstrap` entry. Asking for bootstrap
// by name implies wanting it to run, so an explicit invocation bypasses the
// run-once marker - the marker exists to keep `rwr all` idempotent, not to
// refuse an explicit request. The marker is still refreshed on success and
// dry-run is still honored (no marker write, no mutations).
func RunBootstrap(initConfig *types.InitConfig, osInfo *types.OSInfo) error {
	location := initConfig.Init.Location
	bootstrapFile := findBootstrapFile(location)
	if bootstrapFile == "" {
		return fmt.Errorf("no bootstrap file in %s (looked for %s)",
			location, strings.Join(helpers.CandidateFilenames("bootstrap"), ", "))
	}

	previous := initConfig.Variables.Flags.ForceBootstrap
	initConfig.Variables.Flags.ForceBootstrap = true
	defer func() { initConfig.Variables.Flags.ForceBootstrap = previous }()

	return ProcessBootstrap(bootstrapFile, initConfig, osInfo)
}

// ProcessBootstrap runs one-time system bootstrap operations including packages,
// directories, files, SSH keys, Git repos, services, groups, and users.
// It skips execution if the system is already bootstrapped unless force is set.
func ProcessBootstrap(blueprintFile string, initConfig *types.InitConfig, osInfo *types.OSInfo) error {
	if !initConfig.Variables.Flags.ForceBootstrap && helpers.IsBootstrapped() {
		log.Info("System is already bootstrapped. Skipping bootstrap process.")
		return nil
	}

	log.Info("Starting bootstrap processor...")

	var bootstrapData types.BootstrapData
	var blueprintData []byte
	var err error

	// Resolve variables in the blueprint file if templates are enabled
	blueprintData, err = os.ReadFile(blueprintFile) // #nosec G304 -- path is operator-supplied blueprint/config input; containment added in PR8
	if err != nil {
		log.Errorf("Error reading blueprint file: %v", err)
		return err
	}

	blueprintData, err = helpers.ResolveTemplate(blueprintData, initConfig.Variables)
	if err != nil {
		log.Errorf("Error resolving variables in bootstrap file: %v", err)
		return err
	}

	blueprintDir := filepath.Dir(blueprintFile)

	// Per-file via the registry; Init.Format is only the fallback for the
	// no-file case (inline bootstrap data).
	format := initConfig.Init.Format
	if blueprintFile != "" {
		derived, err := helpers.FormatForPath(blueprintFile)
		if err != nil {
			return err
		}
		format = derived
	}

	// Unmarshal the blueprint data
	log.Debugf("Unmarshaling bootstrap data from %s", blueprintFile)
	err = helpers.DecodeBlueprintInto(blueprintData, format,
		types.BlueprintTypeBootstrap, helpers.TreeSchemaVersion(initConfig), &bootstrapData)
	if err != nil {
		log.Errorf("Error unmarshaling bootstrap blueprint: %v", err)
		return err
	}

	// Process packages
	log.Debugf("Processing packages from %s", blueprintFile)
	packagesData := &types.PackagesData{
		Packages: bootstrapData.Packages,
	}
	err = ProcessPackages(nil, packagesData, blueprintDir, format, osInfo, initConfig)
	if err != nil {
		log.Errorf("Error processing packages: %v", err)
		return err
	}

	// Bootstrap runs the shared per-type loops, so it carries its own progress
	// trackers keyed to the lanes those loops fill.
	filesTrack := newProgress(types.BlueprintTypeFiles)
	usersTrack := newProgress(types.BlueprintTypeUsers)

	// Process directories
	log.Debugf("Processing directories from %s", blueprintFile)
	err = processDirectories(bootstrapData.Directories, blueprintDir, initConfig, filesTrack)
	if err != nil {
		log.Errorf("Error processing directories: %v", err)
		return err
	}

	// Process Files
	log.Debugf("Processing files from %s", blueprintFile)
	err = processFiles(bootstrapData.Files, blueprintDir, osInfo, filesTrack)
	if err != nil {
		log.Errorf("Error processing directories: %v", err)
		return err
	}

	// Process SSH
	log.Debugf("Processing files from %s", blueprintFile)
	err = processSSHKeys(bootstrapData.SSHKeys, osInfo, initConfig)
	if err != nil {
		log.Errorf("Error processing directories: %v", err)
		return err
	}

	// Process Git repositories
	log.Debugf("Processing Git repositories from %s", blueprintFile)
	err = processGitRepositories(bootstrapData.Git, initConfig)
	if err != nil {
		log.Errorf("Error processing Git repositories: %v", err)
		return err
	}

	// Process services
	log.Debugf("Processing services from %s", blueprintFile)
	err = processServices(bootstrapData.Services, osInfo, initConfig)
	if err != nil {
		log.Errorf("Error processing services: %v", err)
		return err
	}

	// Process users/groups
	log.Debugf("Processing users/groups from %s", blueprintFile)
	err = processUsers(bootstrapData.Users, initConfig, usersTrack)
	if err != nil {
		log.Errorf("Error processing groups: %v", err)
		return err
	}

	// Process groups
	log.Debugf("Processing groups from %s", blueprintFile)
	err = processGroups(bootstrapData.Groups, initConfig, usersTrack)
	if err != nil {
		log.Errorf("Error processing groups: %v", err)
		return err
	}

	// Set the bootstrap file
	log.Debugf("Setting bootstrap fileProcessDirectories")
	if err := writeBootstrapMarker(); err != nil {
		log.Errorf("Error setting bootstrap file: %v", err)
		return err
	}

	log.Info("Bootstrap processor completed successfully.")
	return nil
}

// writeBootstrapMarker records that bootstrap has run, unless this was a dry-run.
// Writing the marker during a dry-run would make every later real run believe the
// system is already bootstrapped and skip bootstrap entirely.
func writeBootstrapMarker() error {
	if system.IsDryRun() {
		log.Infof("[DRY-RUN] Would write the bootstrap marker file")
		return nil
	}
	return helpers.Bootstrap()
}
