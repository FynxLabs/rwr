package cmd

import (
	"fmt"
	"github.com/thefynx/rwr/internal/processors"
	"github.com/thefynx/rwr/internal/processors/types"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "rwr",
	Short: "Rinse, Wash, and Repeat - Distrohopper's Friend",
	Long:  `rwr is a cli to manage your Linux system's package manager and repositories.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Use != "help" {
			initializeSystemInfo()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to rwr - The Distrohopper's Friend!")
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
	logLevel         string
	systemInfo       *types.InitConfig
	initFilePath     string
)

func initializeSystemInfo() {
	var err error

	// Initialize the system information
	if initFilePath == "" {
		log.Debugf("No init file path specified. Using default path")
		initFilePath, err = processors.GetBlueprintsLocation(false)
		if err != nil {
			log.With("err", err).Errorf("Error determining blueprints location")
			os.Exit(1)
		}
	}

	systemInfo, err = processors.Initialize(initFilePath)
	if err != nil {
		log.With("err", err).Errorf("Error initializing system information")
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(config)
	var err error

	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "Set the log level (debug, info, warn, error)")

	// Flag for the init.yaml file path
	rootCmd.PersistentFlags().StringVarP(&initFilePath, "init-file", "i", "", "Path to the init.yaml file")
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
		log.With("err", err).Errorf("Error initializing system information")
		os.Exit(1)
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
	configDir := filepath.Join(homeDir, ".config", "rwr")
	err = os.MkdirAll(configDir, os.ModePerm)
	if err != nil {
		log.With("err", err).Errorf("Error creating config directory")
		os.Exit(1)
	}
	viper.AddConfigPath(configDir)
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
