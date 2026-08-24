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
	"time"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/reporting"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// findBootstrapFile returns the bootstrap blueprint in dir, in whichever format it
// is written, or "" when the tree has none.
func findBootstrapFile(dir string) string {
	for _, name := range helpers.CandidateFilenames("bootstrap") {
		candidate := filepath.Join(dir, name)
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
	var stepErrs []types.StepError

	resetFailures()
	// Cancellation is started by Execute, before cobra runs, so a signal that
	// arrives during initialization is not lost. Starting it here would
	// replace that context and discard a cancellation already requested.
	openJournal(initConfig.Init.Location)
	defer closeJournal()

	// Imported blueprint files resolve templates against the same variables
	// as the files that import them (see helpers.SetTemplateVariables).
	defer helpers.SetTemplateVariables(&initConfig.Variables)()

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

	// Make sure the blueprint location exists and parse the entire tree before
	// bootstrap, package-manager setup, or any processor can mutate the machine.
	if _, err := os.Stat(initConfig.Init.Location); err != nil {
		return fmt.Errorf("blueprint location does not exist: %s", initConfig.Init.Location)
	}
	preflight, err := ResolveStage1(initConfig)
	if err != nil {
		return err
	}
	if err := Stage1Error(preflight); err != nil {
		return err
	}

	// Check if macOS and no package manager is installed. A tree that declares
	// packageManagers in its init file has already said what to install - the
	// ProcessPackageManagers call below handles those (installing any that are
	// missing), so the ask-and-install fallback is only for trees that declare
	// nothing.
	if osInfo.System.OS == types.OSDarwin && len(initConfig.PackageManagers) == 0 {
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

			// PromptUserChoice runs under the terminal lease, so it is safe
			// both headless and under the TUI (the dashboard suspends around
			// it instead of deadlocking the stdin read).
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
		started := time.Now()
		reporting.SetCurrentProcessor(types.BlueprintTypeBootstrap)
		reporting.Emit(reporting.ProcStarted{Processor: types.BlueprintTypeBootstrap, Files: 1})
		err = ProcessBootstrap(bootstrapFile, initConfig, osInfo)
		reporting.Emit(reporting.ProcFinished{Processor: types.BlueprintTypeBootstrap, Err: err, Dur: time.Since(started)})
		if err != nil {
			return fmt.Errorf("error processing bootstrap: %w", err)
		}
	}

	// Process each blueprint in order
	for _, processor := range blueprintRunOrder {
		if files, ok := fileOrder[processor]; ok {
			// One ProcStarted per processor, one ProcFinished when its files
			// are done. Started used to fire once per FILE, and Finished was
			// never emitted at all - so the dashboard showed every processor
			// that had ever started as still running, forever.
			procStarted := time.Now()
			reporting.SetCurrentProcessor(processor)
			reporting.Emit(reporting.ProcStarted{Processor: processor, Files: len(files)})
			var procErr error
			// Every abort between ProcStarted and the loop's end must emit
			// the matching ProcFinished, or the display counts this
			// processor as running forever - spinner, clock, taskbar
			// progress all wrong on the final frame.
			fatal := func(ferr error) error {
				reporting.Emit(reporting.ProcFinished{Processor: processor, Err: ferr, Dur: time.Since(procStarted)})
				return ferr
			}
			for _, file := range files {
				blueprintFile := filepath.Join(initConfig.Init.Location, file)
				log.Debugf("Processing blueprint file: %s", blueprintFile)

				// Verify file exists
				if _, err := os.Stat(blueprintFile); err != nil {
					log.Warnf("Blueprint file does not exist: %s", blueprintFile)
					continue
				}

				blueprintDir := filepath.Dir(blueprintFile)
				// Per-file, via the registry: `filepath.Ext(file)[1:]` panicked
				// outright on an extensionless file.
				format, err := helpers.FormatForPath(blueprintFile)
				if err != nil {
					return fatal(err)
				}

				blueprintData, err := os.ReadFile(blueprintFile) // #nosec G304 -- path is operator-supplied blueprint/config input
				if err != nil {
					return fatal(fmt.Errorf("error reading blueprint file %s: %w", blueprintFile, err))
				}

				resolvedBlueprint, err := helpers.ResolveTemplate(blueprintData, initConfig.Variables)
				if err != nil {
					return fatal(fmt.Errorf("error resolving variables in %s: %w", processor, err))
				}

				// A multi-type file (content-routed into several buckets) is cut
				// down to this processor's sections; single-type files pass
				// through untouched and keep strict decode's typo protection.
				resolvedBlueprint, format, err = subsetForProcessor(resolvedBlueprint, format, processor)
				if err != nil {
					return fatal(fmt.Errorf("error preparing %s for the %s processor: %w", blueprintFile, processor, err))
				}

				// Checked per file rather than only per command: a cancelled
				// run should stop reading and decoding blueprints too, not
				// grind through the rest of the tree refusing one command at a
				// time.
				if system.Cancelled() {
					break
				}

				dispatch := func() error {
					switch processor {
					case types.BlueprintTypeRepositories:
						return ProcessRepositories(resolvedBlueprint, blueprintDir, format, osInfo, initConfig)
					case types.BlueprintTypePackages:
						return ProcessPackages(resolvedBlueprint, nil, blueprintDir, format, osInfo, initConfig)
					case types.BlueprintTypeFiles:
						return ProcessFiles(resolvedBlueprint, blueprintDir, format, osInfo, initConfig)
					case types.BlueprintTypeServices:
						return ProcessServices(resolvedBlueprint, blueprintDir, format, osInfo, initConfig)
					case types.BlueprintTypeUsers:
						return ProcessUsers(resolvedBlueprint, blueprintDir, format, initConfig)
					case types.BlueprintTypeGit:
						return ProcessGitRepositories(resolvedBlueprint, blueprintDir, format, initConfig)
					case types.BlueprintTypeScripts:
						return ProcessScripts(resolvedBlueprint, blueprintDir, format, osInfo, initConfig)
					case types.BlueprintTypeSSHKeys:
						return ProcessSSHKeys(resolvedBlueprint, blueprintDir, format, osInfo, initConfig)
					case types.BlueprintTypeFonts:
						return ProcessFonts(resolvedBlueprint, blueprintDir, format, osInfo, initConfig)
					case types.BlueprintTypeConfiguration:
						return ProcessConfiguration(resolvedBlueprint, blueprintDir, format, initConfig)
					default:
						reporting.Emit(reporting.ProcSkipped{Processor: processor, Reason: "unknown processor"})
						return nil
					}
				}

				for {
					err = dispatch()
					if err == nil {
						break
					}
					if !initConfig.Variables.Flags.Interactive {
						// Headless pushes through, collects, and exits
						// nonzero - the first error aborting used to leave
						// every later processor silently unrun in CI.
						if procErr == nil {
							procErr = err
						}
						stepErrs = append(stepErrs, types.StepError{Processor: processor, Err: err})
						break
					}
					// Interactive halts and asks: retry re-runs this file
					// (providers are idempotent, so completed items skip fast
					// and only the failures re-attempt), skip records the
					// error and moves on, abort ends the run.
					switch reporting.RequestHalt(processor, err) {
					case reporting.HaltRetry:
						log.Warnf("Retrying %s after error: %v", processor, err)
						continue
					case reporting.HaltSkip:
						log.Warnf("Skipping past %s error: %v", processor, err)
						if procErr == nil {
							procErr = err
						}
						stepErrs = append(stepErrs, types.StepError{Processor: processor, Err: err})
					default: // abort
						return fatal(fmt.Errorf("error processing %s: %w", processor, err))
					}
					break
				}
			}
			reporting.Emit(reporting.ProcFinished{Processor: processor, Err: procErr, Dur: time.Since(procStarted)})
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

	// A cancelled run is not a failed one. Whatever was in flight when the
	// operator stopped it was killed and reported as an error by the
	// processors that were mid-item, and listing those as failures says rwr
	// broke when in fact it was told to stop. The exit code still says the run
	// did not complete.
	if system.Cancelled() {
		log.Warnf("Run cancelled before it finished; anything already applied is recorded in the run journal")
		return system.ErrCancelled
	}

	// Collected processor errors from a push-through run: each was reported
	// when it happened; the run's exit code has to carry them too.
	if len(stepErrs) > 0 {
		for _, stepErr := range stepErrs {
			log.Errorf("Processor %s failed: %v", stepErr.Processor, stepErr.Err)
		}
		reporting.Emit(reporting.RunFinished{Errs: stepErrs})
		return fmt.Errorf("%d processor(s) failed", len(stepErrs))
	}

	// Processors that skip a failed item and continue record it rather than
	// aborting. Reporting completion without consulting that ledger is how a run
	// where every package failed still exited 0.
	if err := failureError(); err != nil {
		log.Errorf("RWR run finished with %d failure(s)", failureCount())
		if initConfig.Variables.Flags.Interactive {
			reporting.RequestFinalHalt(err)
		}
		return err
	}

	reporting.Emit(reporting.RunFinished{Errs: stepErrs})
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
