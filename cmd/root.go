package cmd

import (
	"fmt"
	"github.com/thefynx/rwr/internal/actions"
	"os"
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
	skipVersionCheck bool
	highlight        bool // Initially set to true by default.
	noHighlight      bool // Used to explicitly disable highlighting.
	output           string
	debug            bool
	logLevel         string
	systemInfo       *actions.InitConfig
	initFilePath     string
)

func initializeSystemInfo() {
	var err error
	systemInfo, err = actions.Initialize(initFilePath)
	if err != nil {
		log.With("err", err).Errorf("Error initializing system information")
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(config)

	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "Set the log level (debug, info, warn, error)")

	err := viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
	if err != nil {
		return
	}

	viper.SetDefault("log.level", "info") // Default log level

	// Adjusting API key flag
	rootCmd.PersistentFlags().StringVar(&ghApiToken, "api-key", "", "Github's API Key (stored under repository.gh_api_token)")
	err = viper.BindPFlag("repository.gh_api_token", rootCmd.PersistentFlags().Lookup("api-key"))
	if err != nil {
		return
	}

	// Adding skipVersionCheck as a global flag
	rootCmd.PersistentFlags().BoolVar(&skipVersionCheck, "skip-version-check", false, "Skip checking for the latest version of rwr")
	err = viper.BindPFlag("rwr.skipVersionCheck", rootCmd.PersistentFlags().Lookup("skip-version-check"))
	if err != nil {
		return
	}

	// Adding highlight as a global flag
	rootCmd.PersistentFlags().BoolVar(&highlight, "highlight", true, "Highlight output for readability")
	// Introducing no-highlight as a global flag
	rootCmd.PersistentFlags().BoolVar(&noHighlight, "no-highlight", false, "Disable output highlighting")

	// Upon flag parsing, check if no-highlight was specified and override highlight value
	cobra.OnInitialize(func() {
		if noHighlight {
			highlight = false
		}
	})

	err = viper.BindPFlag("rwr.highlight", rootCmd.PersistentFlags().Lookup("highlight"))
	if err != nil {
		return
	}

	rootCmd.PersistentFlags().StringVar(&output, "output", "json", "Set the output format for rwr")
	err = viper.BindPFlag("rwr.output", rootCmd.PersistentFlags().Lookup("output"))
	if err != nil {
		return
	}

	// Flag for the init.yaml file path
	rootCmd.PersistentFlags().StringVarP(&initFilePath, "init-file", "i", "./init.yaml", "Path to the init.yaml file")

	viper.SetEnvPrefix("RWR")
	viper.AutomaticEnv()

	if err != nil {
		log.With("err", err).Errorf("Error initializing system information")
		os.Exit(1)
	}
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
	viper.AddConfigPath(homeDir)
	viper.SetConfigName(".rwr")

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
