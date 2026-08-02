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

var rootCmd = &cobra.Command{
	Use:   "rwr",
	Short: "Rinse, Wash, and Repeat - Distrohopper's Friend",
	Long:  `rwr is a cli to manage your Linux system's package manager and repositories.`,
	// A run that fails partway through is not a usage mistake. Printing the full
	// flag listing after "validation failed with 3 errors" buries the errors the
	// operator actually needs to read under a screen of help text. Errors are
	// silenced here too because Execute already reports them; cobra printing its
	// own line as well produced every failure twice.
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip initialization for these commands
		skipInit := map[string]bool{
			"help":     true,
			"config":   true,
			"version":  true,
			"validate": true,
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
					osInfo = system.DetectOS()
					return nil
				}
				return nil
			}
			current = current.Parent()
		}

		checkForNewVersion()

		return initializeSystemInfo()
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to rwr - The Distrohopper's Friend!")
		log.Debugf("Variables: %+v", initConfig.Variables)
		err := cmd.Help()
		if err != nil {
			return
		}
	},
}

var (
	ghApiToken       string // GitHub API token for repository operations
	ghAuth           bool   // Use OAuth device flow for GitHub authentication
	sshKey           string // SSH private key for Git auth (path or base64)
	skipVersionCheck bool
	showSecrets      bool
	debug            bool
	interactive      bool
	forceBootstrap   bool
	dryRun           bool
	logLevel         string
	configPath       string // --config: overrides where the config file is looked up
	configLocation   string
	runOnceLocation  string
	profiles         []string // Global variable for active profiles
	initConfig       *types.InitConfig
	initFilePath     string
	osInfo           *types.OSInfo
)

// initializeSystemInfo initializes system configuration and loads the init file.
// It searches for init files in the configured location or current directory,
// sets up system paths, processes the initialization configuration, retrieves
// blueprints from Git if configured, and detects the operating system.
func initializeSystemInfo() error {
	var err error

	// If no init file is specified via flag, check config
	initFilePath = configuredInitFile(initFilePath)

	// One resolver for every accepted form — local path, directory, https URL,
	// GitHub blob URL, owner/repo[/path][@ref] shorthand. See its doc comment
	// for the precedence.
	resolved, err := helpers.ResolveInitSource(initFilePath)
	if err != nil {
		return fmt.Errorf("error resolving init source %q: %w", initFilePath, err)
	}
	initFilePath = resolved

	flags := types.Flags{
		Debug:            debug,
		LogLevel:         logLevel,
		ForceBootstrap:   forceBootstrap,
		Interactive:      interactive,
		DryRun:           dryRun,
		GHAPIToken:       ghApiToken,
		SSHKey:           sshKey,
		SkipVersionCheck: skipVersionCheck,
		ConfigLocation:   configLocation,
		RunOnceLocation:  runOnceLocation,
		Profiles:         profiles,
	}

	types.SetShowSecrets(showSecrets)
	if showSecrets {
		log.Warnf("--show-secrets is set: credential values will appear in logs")
	}

	if dryRun {
		system.SetDryRun(true)
		log.Infof("Dry-run mode enabled - no changes will be made")
	}

	if err = system.SetPaths(); err != nil {
		return fmt.Errorf("error setting paths: %w", err)
	}

	log.Debugf("Initializing system information with init file: %s", initFilePath)
	initConfig, err = processors.Initialize(initFilePath, flags)
	if err != nil {
		return fmt.Errorf("error initializing system information: %w", err)
	}

	log.Debugf("Checking for blueprints git configuration")
	initFilePath, err = processors.GetBlueprints(initConfig)
	if err != nil {
		return fmt.Errorf("error running GetBlueprints: %w", err)
	}

	osInfo = system.DetectOS()
	return nil
}

// init initializes the Cobra command structure and sets up persistent flags.
// It registers the config function to run on initialization, configures all
// command-line flags including debug mode, init file path, GitHub authentication,
// SSH keys, profiles, and version checking. Flags are bound to viper for
// configuration file integration.
func init() {
	cobra.OnInitialize(func() {
		if err := config(); err != nil {
			log.Fatalf("Configuration error: %v", err)
		}
	})

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to the config file, or to a directory containing config.yaml (default ~/.config/rwr)")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "Set the log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolVar(&forceBootstrap, "force-bootstrap", false, "Force Bootstrap to be ran again")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Log operations without executing (no-op mode)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "no-op", false, "Alias for --dry-run")

	rootCmd.PersistentFlags().BoolVarP(&interactive, "interactive", "I", true, "Enable interactive mode (use --interactive=false to disable)")

	// Flag for the init file path. Bound to repository.init-file — the key the
	// config file and docs have always used. It was bound to rwr.init-file,
	// which nothing read: a two-key split where the flag wrote one key and the
	// config lookup read the other. rwr.init-file still works via a deprecation
	// shim in configuredInitFile.
	rootCmd.PersistentFlags().StringVarP(&initFilePath, "init-file", "i", "", "Path to the init file")
	mustBindFlag("repository.init-file", "init-file")
	mustBindFlag("log.level", "log-level")

	viper.SetDefault("log.level", "info") // Default log level

	// GitHub API Key flags
	rootCmd.PersistentFlags().StringVar(&ghApiToken, "gh-api-key", "", "GitHub API token (stored under repository.gh_api_token)")
	mustBindFlag("repository.gh_api_token", "gh-api-key")
	// --gh-key wrote to the same variable and viper key, so when both flags
	// were given one silently won. Deprecated rather than removed: existing
	// scripts keep working and get told what to change.
	rootCmd.PersistentFlags().StringVar(&ghApiToken, "gh-key", "", "GitHub API token (alias for --gh-api-key)")
	if err := rootCmd.PersistentFlags().MarkDeprecated("gh-key", "use --gh-api-key instead"); err != nil {
		log.Fatalf("marking --gh-key deprecated: %v", err)
	}

	// GitHub OAuth authentication flag
	rootCmd.PersistentFlags().BoolVar(&ghAuth, "gh-auth", false, "Authenticate with GitHub using OAuth device flow")

	rootCmd.PersistentFlags().StringVar(&sshKey, "ssh-key", "", "Path to the SSH key file or Base64-encoded SSH key for Git authentication (stored under repository.ssh_private_key)")
	mustBindFlag("repository.ssh_private_key", "ssh-key")

	// Adding skipVersionCheck as a global flag
	rootCmd.PersistentFlags().BoolVar(&skipVersionCheck, "skip-version-check", false, "Skip checking for the latest version of rwr")
	mustBindFlag("rwr.skipVersionCheck", "skip-version-check")
	viper.SetDefault("rwr.skipVersionCheck", false)

	// Secrets are redacted in logs by default. This exists because "is rwr even
	// reading my token?" has no other answer, but it has to be asked for.
	rootCmd.PersistentFlags().BoolVar(&showSecrets, "show-secrets", false, "Show credential values in logs instead of redacting them")

	// Profile selection flag
	rootCmd.PersistentFlags().StringSliceVarP(&profiles, "profile", "p", []string{}, "Specify profiles to activate (can be used multiple times)")
	mustBindFlag("rwr.profiles", "profile")

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
func mustBindFlag(viperKey, flagName string) {
	if err := viper.BindPFlag(viperKey, rootCmd.PersistentFlags().Lookup(flagName)); err != nil {
		log.Fatalf("Error binding flag %s: %v", flagName, err)
	}
}

// config sets up logging configuration and initializes application directories.
// It creates the config directory at ~/.config/rwr and the run_once directory
// for tracking bootstrap operations. The function also configures the logger
// with appropriate output settings and log levels based on flags and configuration.
// It reads the config file if available and sets up GitHub API tokens and SSH keys.
func config() error {
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
	configLocation, configFile = resolveConfigLocation(homeDir, configPath, viper.GetString("rwr.configdir"))

	runOnceLocation = filepath.Join(configLocation, "run_once")

	// helpers.Bootstrap and the config creator resolve their paths from this key.
	// Without it they fell back to ~/.config/rwr regardless of --config, so
	// --config moved run_once but left the bootstrap marker behind in the default
	// location — the two disagreed about where the config directory was.
	viper.Set("rwr.configdir", configLocation)

	if err = helpers.EnsureConfigDir(configLocation); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	if err = helpers.EnsureConfigDir(runOnceLocation); err != nil {
		return fmt.Errorf("error creating bootstrap directory: %w", err)
	}

	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.AddConfigPath(configLocation)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	if err := viper.ReadInConfig(); err == nil {
		log.Debugf("Using config file: %s", viper.ConfigFileUsed())
	} else {
		log.Debugf("No config file found. Using default settings")
	}

	// Check if debug flag is set to enable debug level logging
	if debug {
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

	ghApiToken = viper.GetString("repository.gh_api_token")
	sshKey = viper.GetString("repository.ssh_private_key")
	return nil
}

// Execute runs the root command and handles any errors that occur during execution.
// This is the main entry point for the CLI application and should be called from main.
// It exits with status code 1 if an error occurs.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("%v", err)
	}
}
