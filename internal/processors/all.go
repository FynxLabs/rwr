// Package processors handles the execution of blueprint-defined operations.
// It processes various components including packages, repositories, files,
// services, Git configurations, scripts, SSH keys, fonts, and user management.
// The package orchestrates the complete blueprint workflow, from initialization
// and bootstrap to final configuration application on the target system.
package processors

import (
	"fmt"
	"os"
	"path/filepath"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// findBootstrapFile returns the bootstrap blueprint in dir, in whichever format it
// is written, or "" when the tree has none.
func findBootstrapFile(dir string) string {
	for _, ext := range []string{types.FormatExtYAML, types.FormatExtYAMLAlt, types.FormatExtJSON, types.FormatExtTOML} {
		candidate := filepath.Join(dir, "bootstrap"+ext)
		if system.FileExists(candidate) {
			return candidate
		}
	}
	return ""
}

// All orchestrates the execution of all blueprint processors in the defined run order.
// It handles bootstrap, package installation, file management, services, and other
// operations sequentially, cleaning up package manager caches on completion.
func All(initConfig *types.InitConfig, osInfo *types.OSInfo, runOrder []string) error {
	var err error
	var blueprintRunOrder []string

	resetFailures()

	log.Debugf("ForceBootstrap: %v", initConfig.Variables.Flags.ForceBootstrap)

	if system.IsDryRun() {
		log.Warnf("=== DRY-RUN MODE ===")
		log.Warnf("No changes will be made to the system")
		log.Warnf("====================")
	}

	// First, ensure the blueprint repository is set up
	_, err = GetBlueprints(initConfig)
	if err != nil {
		return fmt.Errorf("error initializing blueprints: %w", err)
	}

	// Check if macOS and no package manager is installed
	if osInfo.System.OS == types.OSDarwin {
		// Check if any package manager is installed
		hasPackageManager := false
		for _, pm := range osInfo.PackageManager.Managers {
			if pm.Bin != "" {
				hasPackageManager = true
				break
			}
		}

		if !hasPackageManager {
			log.Info("No package manager detected on macOS. Installing one is required to proceed.")

			var chosenPM string
			if initConfig.Variables.Flags.Interactive {
				chosenPM = system.PromptUserChoice("Choose a package manager to install", []string{"brew", "nix"}, "brew")
			} else {
				chosenPM = "brew"
				log.Info("Non-interactive mode: defaulting to Homebrew (brew)")
			}

			pmInfo := types.PackageManagerInfo{
				Name:   chosenPM,
				Action: types.ActionInstall,
			}

			err = ProcessPackageManagers([]types.PackageManagerInfo{pmInfo}, osInfo, initConfig)
			if err != nil {
				return fmt.Errorf("error installing package manager: %w", err)
			}
		}
	}

	// Make sure the blueprint location exists
	if _, err := os.Stat(initConfig.Init.Location); err != nil {
		return fmt.Errorf("blueprint location does not exist: %s", initConfig.Init.Location)
	}

	if runOrder != nil {
		blueprintRunOrder = append([]string(nil), runOrder...)
	} else {
		blueprintRunOrder, err = GetBlueprintRunOrder(initConfig)
		if err != nil {
			return fmt.Errorf("error getting blueprint run order: %w", err)
		}
	}

	// Process package managers first if specified
	if initConfig.PackageManagers != nil {
		log.Debugf("Processing package managers")
		err = ProcessPackageManagers(initConfig.PackageManagers, osInfo, initConfig)
		if err != nil {
			return fmt.Errorf("error processing package managers: %w", err)
		}
	}

	if err := checkRequestedProfiles(initConfig); err != nil {
		return err
	}

	// Get the blueprint file order
	fileOrder, err := GetBlueprintFileOrder(initConfig.Init.Location, initConfig.Init.Order, initConfig.Init.RunOnlyListed, initConfig)
	if err != nil {
		return fmt.Errorf("error getting blueprint file order: %w", err)
	}

	// Run the bootstrap processor first if it exists.
	//
	// Every extension is checked, not just .yaml: a tree written in TOML or JSON
	// declares its format in the init file, and `rwr validate` already accepts all
	// four. Looking only for bootstrap.yaml meant those trees silently never
	// bootstrapped, with no message saying so.
	if bootstrapFile := findBootstrapFile(initConfig.Init.Location); bootstrapFile != "" {
		err = ProcessBootstrap(bootstrapFile, initConfig, osInfo)
		if err != nil {
			return fmt.Errorf("error processing bootstrap: %w", err)
		}
	}

	// Process each blueprint in order
	for _, processor := range blueprintRunOrder {
		if files, ok := fileOrder[processor]; ok {
			for _, file := range files {
				blueprintFile := filepath.Join(initConfig.Init.Location, file)
				log.Debugf("Processing blueprint file: %s", blueprintFile)

				// Verify file exists
				if _, err := os.Stat(blueprintFile); err != nil {
					log.Warnf("Blueprint file does not exist: %s", blueprintFile)
					continue
				}

				blueprintDir := filepath.Dir(blueprintFile)
				format := filepath.Ext(blueprintFile)[1:] // Remove the leading dot

				blueprintData, err := os.ReadFile(blueprintFile) // #nosec G304 -- path is operator-supplied blueprint/config input; containment added in PR8
				if err != nil {
					return fmt.Errorf("error reading blueprint file %s: %w", blueprintFile, err)
				}

				resolvedBlueprint, err := helpers.ResolveTemplate(blueprintData, initConfig.Variables)
				if err != nil {
					return fmt.Errorf("error resolving variables in %s: %w", processor, err)
				}

				switch processor {
				case types.BlueprintTypeRepositories:
					log.Infof("Processing repositories")
					err = ProcessRepositories(resolvedBlueprint, format, osInfo, initConfig)
				case types.BlueprintTypePackages:
					log.Infof("Processing packages")
					err = ProcessPackages(resolvedBlueprint, nil, format, osInfo, initConfig)
				case types.BlueprintTypeFiles:
					log.Infof("Processing files")
					err = ProcessFiles(resolvedBlueprint, blueprintDir, format, osInfo, initConfig)
				case types.BlueprintTypeServices:
					log.Infof("Processing services")
					err = ProcessServices(resolvedBlueprint, format, osInfo, initConfig)
				case types.BlueprintTypeUsers:
					log.Infof("Processing users")
					err = ProcessUsers(resolvedBlueprint, format, initConfig)
				case types.BlueprintTypeGit:
					log.Infof("Processing git repositories")
					err = ProcessGitRepositories(resolvedBlueprint, format, initConfig)
				case types.BlueprintTypeScripts:
					log.Infof("Processing scripts")
					err = ProcessScripts(resolvedBlueprint, blueprintDir, format, osInfo, initConfig)
				case types.BlueprintTypeSSHKeys:
					log.Infof("Processing ssh keys")
					err = ProcessSSHKeys(resolvedBlueprint, format, osInfo, initConfig)
				case types.BlueprintTypeFonts:
					log.Info("Processing fonts")
					err = ProcessFonts(resolvedBlueprint, blueprintDir, format, osInfo, initConfig)
				case types.BlueprintTypeConfiguration:
					log.Infof("Processing configurations")
					err = ProcessConfiguration(resolvedBlueprint, blueprintDir, format, initConfig)
				default:
					log.Warnf("Unknown processor: %s", processor)
					continue
				}

				if err != nil {
					return fmt.Errorf("error processing %s: %w", processor, err)
				}
			}
		}
	}

	// Clean up package managers
	if !system.IsDryRun() {
		log.Infof("Cleaning up package managers")
		if err = system.CleanPackageManagers(osInfo, initConfig); err != nil {
			return fmt.Errorf("error cleaning package managers: %w", err)
		}
	}

	if system.IsDryRun() {
		log.Infof("")
		log.Infof("=== DRY-RUN SUMMARY ===")
		log.Infof("Processors that would run: %d", len(blueprintRunOrder))
		for _, p := range blueprintRunOrder {
			if files, ok := fileOrder[p]; ok {
				log.Infof("  %s: %d blueprint file(s)", p, len(files))
			}
		}
		log.Infof("=======================")
		log.Infof("")
		log.Infof("No changes were made to the system.")
	}

	// Processors that skip a failed item and continue record it rather than
	// aborting. Reporting completion without consulting that ledger is how a run
	// where every package failed still exited 0.
	if err := failureError(); err != nil {
		log.Errorf("RWR run finished with %d failure(s)", failureCount())
		return err
	}

	log.Info("RWR Run Complete!")
	return nil
}

// checkRequestedProfiles refuses a --profile the tree does not declare.
//
// A misspelled profile name was silent: FilterByProfiles matched nothing, every
// profile-scoped entry was skipped, and the run reported success having installed
// only the base items. A mistyped profile looked exactly like a working run.
func checkRequestedProfiles(initConfig *types.InitConfig) error {
	requested := initConfig.Variables.Flags.Profiles
	if len(requested) == 0 {
		return nil
	}

	summary, err := CollectProfiles(initConfig)
	if err != nil {
		// Discovery is a convenience, not a gate: if the tree cannot be walked the
		// processors below will report why, with better context than this can.
		log.Debugf("Could not collect profiles to validate --profile: %v", err)
		return nil
	}

	invalid := helpers.ValidateProfiles(requested, summary.Names)
	if len(invalid) == 0 {
		return nil
	}

	// Only a tree that declares profiles gives a trustworthy set to check against.
	// When it declares none, the discovery walk is as likely to be the thing at
	// fault as the operator, so say so rather than refusing to run.
	if len(summary.Names) == 0 {
		log.Warnf("--profile %v was given, but no blueprint in this tree declares any profiles; every profile-scoped entry will be skipped", invalid)
		return nil
	}

	return fmt.Errorf("no profile named %v exists in this blueprint tree; available profiles: %v", invalid, summary.Names)
}
