package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/processors"
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
		if cmd.Use != "help" && cmd.Use != "config" && cmd.Use != "version" {
			initializeSystemInfo()
		}
		return nil
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
	ghApiToken       string // Global variable for API Key
	sshKey           string // Global variable for SSH Key
	skipVersionCheck bool
	debug            bool
	interactive      bool
	forceBootstrap   bool
	logLevel         string
	configLocation   string
	runOnceLocation  string
	initConfig       *types.InitConfig
	initFilePath     string
	osInfo           *types.OSInfo
)

func initializeSystemInfo() {
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
		GHAPIToken:       ghApiToken,
		SSHKey:           sshKey,
		SkipVersionCheck: skipVersionCheck,
		ConfigLocation:   configLocation,
		RunOnceLocation:  runOnceLocation,
	}

	err = helpers.SetPaths()
	if err != nil {
		log.With("err", err).Errorf("Error setting paths")
		os.Exit(1)
	}

	log.Debugf("Initializing system information with init file: %s", initFilePath)
	initConfig, err = processors.Initialize(initFilePath, flags)
	if err != nil {
		log.With("err", err).Errorf("Error initializing system information")
		os.Exit(1)
	}

	log.Debugf("Checking for blueprints git configuration")
	initFilePath, err = processors.GetBlueprints(initConfig)
	if err != nil {
		log.With("err", err).Errorf("Error running GetBlueprints")
		os.Exit(1)
	}

	osInfo = helpers.DetectOS()
}

func init() {
	cobra.OnInitialize(config)
	var err error

	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "Set the log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolVar(&forceBootstrap, "force-bootstrap", false, "Force Bootstrap to be ran again")

	rootCmd.PersistentFlags().BoolVar(&interactive, "interactive", false, "Enable interactive mode")

	// Flag for the init file path
	rootCmd.PersistentFlags().StringVarP(&initFilePath, "init-file", "i", "", "Path to the init file")
	err = viper.BindPFlag("rwr.init-file", rootCmd.PersistentFlags().Lookup("init-file"))
	if err != nil {
		log.With("err", err).Errorf("Error initializing system information")
		os.Exit(1)
	}

	err = viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
	if err != nil {
		log.With("err", err).Errorf("Error initializing system information")
		os.Exit(1)
	}

	viper.SetDefault("log.level", "info") // Default log level

	// GitHub API Key flag
	rootCmd.PersistentFlags().StringVar(&ghApiToken, "gh-api-key", "", "Github's API Key (stored under repository.gh_api_token)")
	err = viper.BindPFlag("repository.gh_api_token", rootCmd.PersistentFlags().Lookup("api-key"))
	if err != nil {
		return
	}

	//
	rootCmd.PersistentFlags().StringVar(&sshKey, "ssh-key", "", "Pass in the ssh key Base64 encoded (stored under repository.ssh_private_key)")
	err = viper.BindPFlag("repository.ssh_private_key", rootCmd.PersistentFlags().Lookup("ssh-key"))
	if err != nil {
		log.With("err", err).Errorf("Error initializing system information")
		os.Exit(1)
	}

	// Adding skipVersionCheck as a global flag
	rootCmd.PersistentFlags().BoolVar(&skipVersionCheck, "skip-version-check", false, "Skip checking for the latest version of rwr")
	err = viper.BindPFlag("rwr.skipVersionCheck", rootCmd.PersistentFlags().Lookup("skip-version-check"))
	if err != nil {
		log.With("err", err).Errorf("Error initializing system information")
		os.Exit(1)
	}

	viper.SetEnvPrefix("RWR")
	viper.AutomaticEnv()

}

func config() {
	// Create a new logger
	log.SetTimeFormat(time.Kitchen)
	log.SetReportCaller(true)
	log.SetReportTimestamp(true)
	log.SetPrefix("rwr: ")
	log.SetOutput(os.Stderr)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.With("err", err).Errorf("Error finding home directory")
		os.Exit(1)
	}
	configLocation = filepath.Join(homeDir, ".config", "rwr")
	runOnceLocation = filepath.Join(configLocation, "run_once")

	err = os.MkdirAll(configLocation, os.ModePerm)
	if err != nil {
		log.With("err", err).Errorf("Error creating config directory")
		os.Exit(1)
	}

	err = os.MkdirAll(runOnceLocation, os.ModePerm)
	if err != nil {
		log.With("err", err).Errorf("Error creating bootstrap directory")
		os.Exit(1)
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
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.With("err", err).Fatalf("Error executing command")
		os.Exit(1)
	}
}
