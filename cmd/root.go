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
	"time"

	"github.com/fynxlabs/rwr/internal/processors"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "rwr",
	Short: "Rinse, Wash, and Repeat - Distrohopper's Friend",
	Long:  `rwr is a cli to manage your Linux system's package manager and repositories.`,
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
	debug            bool
	interactive      bool
	forceBootstrap   bool
	dryRun           bool
	logLevel         string
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
	if initFilePath == "" {
		initFilePath = viper.GetString("repository.init-file")
	}

	// If we have a path, check if it's a directory
	if initFilePath != "" {
		// Check if path exists
		fileInfo, err := os.Stat(initFilePath)
		if err == nil && fileInfo.IsDir() {
			// If it's a directory, look for init files
			possibleFiles := []string{
				filepath.Join(initFilePath, "init.yaml"),
				filepath.Join(initFilePath, "init.yml"),
				filepath.Join(initFilePath, "init.json"),
				filepath.Join(initFilePath, "init.toml"),
			}
			for _, file := range possibleFiles {
				if _, err := os.Stat(file); err == nil {
					initFilePath = file
					log.Debugf("Found init file in directory: %s", initFilePath)
					break
				}
			}
		}
	} else {
		// If no path specified, look in current directory
		possibleFiles := []string{"init.yaml", "init.yml", "init.json", "init.toml"}
		for _, file := range possibleFiles {
			if _, err := os.Stat(file); err == nil {
				initFilePath = file
				break
			}
		}
	}

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

	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "Set the log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolVar(&forceBootstrap, "force-bootstrap", false, "Force Bootstrap to be ran again")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Log operations without executing (no-op mode)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "no-op", false, "Alias for --dry-run")

	rootCmd.PersistentFlags().BoolVarP(&interactive, "interactive", "I", true, "Enable interactive mode (use --interactive=false to disable)")

	// Flag for the init file path
	rootCmd.PersistentFlags().StringVarP(&initFilePath, "init-file", "i", "", "Path to the init file")
	mustBindFlag("rwr.init-file", "init-file")
	mustBindFlag("log.level", "log-level")

	viper.SetDefault("log.level", "info") // Default log level

	// GitHub API Key flags
	rootCmd.PersistentFlags().StringVar(&ghApiToken, "gh-api-key", "", "GitHub API token (stored under repository.gh_api_token)")
	rootCmd.PersistentFlags().StringVar(&ghApiToken, "gh-key", "", "GitHub API token (alias for --gh-api-key)")
	mustBindFlag("repository.gh_api_token", "gh-api-key")
	mustBindFlag("repository.gh_api_token", "gh-key")

	// GitHub OAuth authentication flag
	rootCmd.PersistentFlags().BoolVar(&ghAuth, "gh-auth", false, "Authenticate with GitHub using OAuth device flow")

	rootCmd.PersistentFlags().StringVar(&sshKey, "ssh-key", "", "Path to the SSH key file or Base64-encoded SSH key for Git authentication (stored under repository.ssh_private_key)")
	mustBindFlag("repository.ssh_private_key", "ssh-key")

	// Adding skipVersionCheck as a global flag
	rootCmd.PersistentFlags().BoolVar(&skipVersionCheck, "skip-version-check", false, "Skip checking for the latest version of rwr")
	mustBindFlag("rwr.skipVersionCheck", "skip-version-check")

	// Profile selection flag
	rootCmd.PersistentFlags().StringSliceVarP(&profiles, "profile", "p", []string{}, "Specify profiles to activate (can be used multiple times)")
	mustBindFlag("rwr.profiles", "profile")

	viper.SetEnvPrefix("RWR")
	viper.AutomaticEnv()
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
	configLocation = filepath.Join(homeDir, ".config", "rwr")
	runOnceLocation = filepath.Join(configLocation, "run_once")

	if err = os.MkdirAll(configLocation, os.ModePerm); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	if err = os.MkdirAll(runOnceLocation, os.ModePerm); err != nil {
		return fmt.Errorf("error creating bootstrap directory: %w", err)
	}

	viper.AddConfigPath(configLocation)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

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
		log.With("err", err).Fatalf("Error executing command")
	}
}
