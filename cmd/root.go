// Package cmd provides the command-line interface for rwr (Rinse, Wash, and Repeat).
// It implements CLI commands using the Cobra framework for managing Linux system
// packages, repositories, and configuration through blueprint files. The package
// handles initialization, configuration management, and command execution for the
// distrohopper's toolkit.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/processors"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"

	"charm.land/log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewRootCmd builds the whole command tree against one AppConfig. Flags bind
// to the struct's fields and every closure captures the same instance, so
// there is no package-level mutable state beyond the tree a caller builds.
func NewRootCmd(app *AppConfig) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "rwr",
		Short: "Rinse, Wash, and Repeat - Distrohopper's Friend",
		Long: `rwr provisions Linux, macOS and Windows machines from blueprint files:
packages, repositories, files and templates, services, users, SSH keys, fonts,
git checkouts, scripts, and desktop configuration.`,
		// A run that fails partway through is not a usage mistake. Printing the full
		// flag listing after "validation failed with 3 errors" buries the errors the
		// operator actually needs to read under a screen of help text. Errors are
		// silenced here too because Execute already reports them; cobra printing its
		// own line as well produced every failure twice.
		SilenceUsage:  true,
		SilenceErrors: true,
		// Cobra's default arg validation rejects anything that is not a declared
		// subcommand before RunE ever sees it; the task-runner shorthand
		// (`rwr packages`) needs the args to reach RunE.
		Args: cobra.ArbitraryArgs,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Logging, config directory, and the config file — previously a
			// cobra.OnInitialize hook, which is process-global and appends per
			// tree; a per-tree PreRun keeps instances isolated.
			if err := loadConfig(app); err != nil {
				return fmt.Errorf("configuration error: %w", err)
			}

			// Skip initialization for these commands
			skipInit := map[string]bool{
				"help":     true,
				"config":   true,
				"version":  true,
				"validate": true,
				"convert":  true,
			}

			// Check if the current command or any of its parents should skip init
			current := cmd
			for current != nil {
				if skipInit[current.Name()] {
					// For validate command, just detect OS
					if current.Name() == "validate" {
						if err := system.SetPaths(); err != nil {
							return fmt.Errorf("error setting paths: %w", err)
						}
						app.OSInfo = system.DetectOS()
						return nil
					}
					return nil
				}
				current = current.Parent()
			}

			checkForNewVersion(app)

			return initializeSystemInfo(app, selectedProcessorsFor(cmd, args)...)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Processor names work straight off the root, task-runner style:
			// `rwr packages` is `rwr run packages`. They are not part of the
			// prime command namespace, so there is nothing to collide with.
			if len(args) > 0 {
				if p, ok := processorShorthand(args[0]); ok {
					return runOneProcessor(app, p)
				}
				if err := cmd.Help(); err != nil {
					return err
				}
				return fmt.Errorf("unknown command or processor %q", args[0])
			}

			fmt.Println("Welcome to rwr - The Distrohopper's Friend!")
			log.Debugf("Variables: %+v", app.InitConfig.Variables)
			return cmd.Help()
		},
	}

	rootCmd.Version = buildInfo.Version
	rootCmd.SetVersionTemplate("rwr {{.Version}}\n")

	registerRootFlags(rootCmd, app)

	rootCmd.AddCommand(newAllCmd(app))
	rootCmd.AddCommand(newRunCmd(app))
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newValidateCmd(app))
	rootCmd.AddCommand(newVersionCmd(app))
	rootCmd.AddCommand(newProfilesCmd(app))
	rootCmd.AddCommand(newConvertCmd())

	return rootCmd
}

// registerRootFlags binds every persistent flag to its AppConfig field and
// wires the viper keys and environment handling.
func registerRootFlags(rootCmd *cobra.Command, app *AppConfig) {
	flags := rootCmd.PersistentFlags()

	flags.StringVar(&app.ConfigPath, "config", "", "Path to the config file, or to a directory containing config.yaml (default ~/.config/rwr)")
	flags.StringVar(&app.ConfigName, "config-name", "", "Manifest configuration to use in a multi-configuration blueprint repo")
	flags.BoolVarP(&app.Debug, "debug", "d", false, "Enable debug mode")
	flags.StringVar(&app.LogLevel, "log-level", "", "Set the log level (debug, info, warn, error)")
	flags.BoolVar(&app.ForceBootstrap, "force-bootstrap", false, "Force Bootstrap to be ran again")
	flags.BoolVar(&app.DryRun, "dry-run", false, "Log operations without executing (no-op mode)")
	flags.BoolVar(&app.DryRun, "no-op", false, "Alias for --dry-run")

	flags.BoolVarP(&app.Interactive, "interactive", "I", true, "Enable interactive mode (use --interactive=false to disable)")

	// Flag for the init file path. Bound to repository.init-file — the key the
	// config file and docs have always used. It was bound to rwr.init-file,
	// which nothing read: a two-key split where the flag wrote one key and the
	// config lookup read the other. rwr.init-file still works via a deprecation
	// shim in configuredInitFile.
	flags.StringVarP(&app.InitFilePath, "init-file", "i", "", "Path to the init file")
	mustBindFlag(rootCmd, "repository.init-file", "init-file")
	mustBindFlag(rootCmd, "log.level", "log-level")

	viper.SetDefault("log.level", "info") // Default log level

	// GitHub API Key flags
	flags.StringVar(&app.GHAPIToken, "gh-api-key", "", "GitHub API token (stored under repository.gh_api_token)")
	mustBindFlag(rootCmd, "repository.gh_api_token", "gh-api-key")
	// --gh-key wrote to the same variable and viper key, so when both flags
	// were given one silently won. Deprecated rather than removed: existing
	// scripts keep working and get told what to change.
	flags.StringVar(&app.GHAPIToken, "gh-key", "", "GitHub API token (alias for --gh-api-key)")
	if err := flags.MarkDeprecated("gh-key", "use --gh-api-key instead"); err != nil {
		log.Fatalf("marking --gh-key deprecated: %v", err)
	}

	// GitHub OAuth authentication flag
	flags.BoolVar(&app.GHAuth, "gh-auth", false, "Authenticate with GitHub using OAuth device flow")

	flags.StringVar(&app.SSHKey, "ssh-key", "", "Path to the SSH key file or Base64-encoded SSH key for Git authentication (stored under repository.ssh_private_key)")
	mustBindFlag(rootCmd, "repository.ssh_private_key", "ssh-key")

	// Adding skipVersionCheck as a global flag
	flags.BoolVar(&app.SkipVersionCheck, "skip-version-check", false, "Skip checking for the latest version of rwr")
	mustBindFlag(rootCmd, "rwr.skipVersionCheck", "skip-version-check")
	viper.SetDefault("rwr.skipVersionCheck", false)

	// Display flags. The TUI activates only on a real, capable, non-CI
	// terminal; everything else keeps the byte-identical streaming output.
	flags.BoolVar(&app.NoTUI, "no-tui", false, "Disable the interactive dashboard; stream logs")
	flags.StringVar(&app.Theme, "theme", "", "Dashboard theme (rwr, mocha, nord, gruvbox, ...)")
	flags.BoolVar(&app.ASCII, "ascii", false, "Force ASCII glyphs in the dashboard")
	flags.BoolVar(&app.Unicode, "unicode", false, "Force unicode glyphs in the dashboard")
	flags.BoolVar(&app.NoNotify, "no-notify", false, "Disable the completion notification")
	flags.IntVar(&app.TUIBuffer, "tui-buffer", 50000, "Dashboard log buffer size in lines")
	flags.StringVar(&app.LogFile, "log-file", "", "Write the run log to this path (default: a temp file)")

	// Secrets are redacted in logs by default. This exists because "is rwr even
	// reading my token?" has no other answer, but it has to be asked for.
	flags.BoolVar(&app.ShowSecrets, "show-secrets", false, "Show credential values in logs instead of redacting them")

	// Profile selection flag
	flags.StringSliceVarP(&app.Profiles, "profile", "p", []string{}, "Specify profiles to activate (can be used multiple times)")
	mustBindFlag(rootCmd, "rwr.profiles", "profile")

	viper.SetEnvPrefix("RWR")
	// Config keys are nested ("log.level"), but a dot is not legal in an
	// environment variable name. Without this replacer viper would look up
	// "RWR_LOG.LEVEL" and the documented RWR_LOG_LEVEL could never resolve.
	// "-" maps to "_" as well: keys like rwr.init-file were otherwise only
	// reachable through an env name containing a hyphen, which POSIX shells
	// cannot export (RWR_RWR_INIT_FILE now works).
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
}

// initializeSystemInfo initializes system configuration and loads the init file.
// It searches for init files in the configured location or current directory,
// sets up system paths, processes the initialization configuration, retrieves
// blueprints from Git if configured, and detects the operating system.
func initializeSystemInfo(app *AppConfig, selectedProcessors ...string) error {
	var err error

	// If no init file is specified via flag, check config
	app.InitFilePath = configuredInitFile(app.InitFilePath)

	// One resolver for every accepted form — local path, directory, https URL,
	// GitHub blob URL, owner/repo[/path][@ref] shorthand. See its doc comment
	// for the precedence.
	resolved, err := helpers.ResolveInitSource(app.InitFilePath)
	if err != nil {
		return fmt.Errorf("error resolving init source %q: %w", app.InitFilePath, err)
	}
	app.InitFilePath = resolved

	// A multi-configuration repo resolves to its manifest; entry selection
	// happens here, before resolve stage 1 touches anything.
	if isManifestPath(app.InitFilePath) {
		selected, err := selectFromManifest(app, app.InitFilePath)
		if err != nil {
			return err
		}
		app.InitFilePath = selected
	}

	flags := types.Flags{
		Debug:            app.Debug,
		LogLevel:         app.LogLevel,
		ForceBootstrap:   app.ForceBootstrap,
		Interactive:      app.Interactive,
		DryRun:           app.DryRun,
		GHAPIToken:       app.GHAPIToken,
		SSHKey:           app.SSHKey,
		SkipVersionCheck: app.SkipVersionCheck,
		ConfigLocation:   app.ConfigLocation,
		RunOnceLocation:  app.RunOnceLocation,
		Profiles:         app.Profiles,
	}

	types.SetShowSecrets(app.ShowSecrets)
	if app.ShowSecrets {
		log.Warnf("--show-secrets is set: credential values will appear in logs")
	}

	if app.DryRun {
		system.SetDryRun(true)
		log.Infof("Dry-run mode enabled - no changes will be made")
	}

	if err = system.SetPaths(); err != nil {
		return fmt.Errorf("error setting paths: %w", err)
	}

	log.Debugf("Initializing system information with init file: %s", app.InitFilePath)
	app.InitConfig, err = processors.Initialize(app.InitFilePath, flags, selectedProcessors...)
	if err != nil {
		return fmt.Errorf("error initializing system information: %w", err)
	}

	log.Debugf("Checking for blueprints git configuration")
	app.InitFilePath, err = processors.GetBlueprints(app.InitConfig)
	if err != nil {
		return fmt.Errorf("error running GetBlueprints: %w", err)
	}

	app.OSInfo = system.DetectOS()
	return nil
}

// configuredInitFile resolves where the init file comes from: the --init-file
// flag, then the repository.init-file config key (which the flag is also bound
// to), then the deprecated rwr.init-file key — honored with a warning so
// configs written against the old two-key split keep working.
func configuredInitFile(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if v := viper.GetString("repository.init-file"); v != "" {
		return v
	}
	if legacy := viper.GetString("rwr.init-file"); legacy != "" {
		log.Warnf("rwr.init-file is deprecated; use repository.init-file (honoring the value this run)")
		return legacy
	}
	return ""
}

// resolveConfigLocation decides the config directory and, when --config named
// a file, the config file path. Precedence: the --config flag, then the
// rwr.configdir key (env: RWR_RWR_CONFIGDIR — it has to come from the
// environment, since the config file cannot name its own directory), then
// ~/.config/rwr. rwr.configdir used to be read and then unconditionally
// overwritten, so documenting it was a lie.
func resolveConfigLocation(homeDir, configFlag, configuredDir string) (location, file string) {
	location = filepath.Join(homeDir, ".config", "rwr")
	if configuredDir != "" {
		location = system.ExpandPath(configuredDir)
	}

	// --config either names a config file directly or names the directory to
	// look for config.yaml in. A file keeps its directory as the config
	// location, so run_once stays next to the config it belongs to.
	if configFlag != "" {
		expanded := system.ExpandPath(configFlag)
		if info, statErr := os.Stat(expanded); statErr == nil && info.IsDir() {
			location = expanded
		} else {
			file = expanded
			location = filepath.Dir(expanded)
		}
	}
	return location, file
}

// mustBindFlag binds a viper config key to a persistent flag, panicking on failure.
// Flag binding errors indicate a programming error (e.g., referencing a non-existent flag).
func mustBindFlag(rootCmd *cobra.Command, viperKey, flagName string) {
	if err := viper.BindPFlag(viperKey, rootCmd.PersistentFlags().Lookup(flagName)); err != nil {
		log.Fatalf("Error binding flag %s: %v", flagName, err)
	}
}

// loadConfig sets up logging configuration and initializes application directories.
// It creates the config directory at ~/.config/rwr and the run_once directory
// for tracking bootstrap operations. The function also configures the logger
// with appropriate output settings and log levels based on flags and configuration.
// It reads the config file if available and sets up GitHub API tokens and SSH keys.
func loadConfig(app *AppConfig) error {
	// Create a new logger
	log.SetTimeFormat(time.Kitchen)
	log.SetReportCaller(true)
	log.SetReportTimestamp(true)
	log.SetPrefix("rwr: ")
	log.SetOutput(os.Stderr)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error finding home directory: %w", err)
	}

	// rwr.configdir is read before the config file is (it decides where that
	// file lives, so only the environment can supply it); --config wins.
	var configFile string
	app.ConfigLocation, configFile = resolveConfigLocation(homeDir, app.ConfigPath, viper.GetString("rwr.configdir"))

	app.RunOnceLocation = filepath.Join(app.ConfigLocation, "run_once")

	// helpers.Bootstrap and the config creator resolve their paths from this key.
	// Without it they fell back to ~/.config/rwr regardless of --config, so
	// --config moved run_once but left the bootstrap marker behind in the default
	// location — the two disagreed about where the config directory was.
	viper.Set("rwr.configdir", app.ConfigLocation)

	if err = helpers.EnsureConfigDir(app.ConfigLocation); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	if err = helpers.EnsureConfigDir(app.RunOnceLocation); err != nil {
		return fmt.Errorf("error creating bootstrap directory: %w", err)
	}

	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.AddConfigPath(app.ConfigLocation)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	if err := viper.ReadInConfig(); err == nil {
		log.Debugf("Using config file: %s", viper.ConfigFileUsed())
	} else {
		log.Debugf("No config file found. Using default settings")
	}

	// Check if debug flag is set to enable debug level logging
	if app.Debug {
		log.SetLevel(log.DebugLevel)
	} else {
		// Otherwise, set the log level based on the "log.level" configuration
		switch viper.GetString("log.level") {
		case "debug":
			log.SetLevel(log.DebugLevel)
		case "info":
			log.SetLevel(log.InfoLevel)
		case "warn":
			log.SetLevel(log.WarnLevel)
		case "error":
			log.SetLevel(log.ErrorLevel)
		default:
			log.SetLevel(log.InfoLevel) // Default to info level if unspecified
		}
	}

	app.GHAPIToken = viper.GetString("repository.gh_api_token")
	app.SSHKey = viper.GetString("repository.ssh_private_key")
	return nil
}

// Execute runs the root command and handles any errors that occur during execution.
// This is the main entry point for the CLI application and should be called from main.
// It exits with status code 1 if an error occurs.
func Execute() {
	app := NewAppConfig()
	if err := NewRootCmd(app).Execute(); err != nil {
		log.Fatalf("%v", err)
	}
}
