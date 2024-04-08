package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/thefynx/rwr/internal/processors"
	"os"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run individual processors",
}

var runPackageCmd = &cobra.Command{
	Use:   "package",
	Short: "Run package processor",
	Run: func(cmd *cobra.Command, args []string) {
		err := processors.All(systemInfo, []string{"packages"})
		if err != nil {
			log.With("err", err).Errorf("Error running package processor")
			os.Exit(1)
		}
	},
}

var runRepositoryCmd = &cobra.Command{
	Use:   "repository",
	Short: "Run repository processor",
	Run: func(cmd *cobra.Command, args []string) {
		err := processors.All(systemInfo, []string{"repositories"})
		if err != nil {
			log.With("err", err).Errorf("Error running repository processor")
			os.Exit(1)
		}
	},
}

var runServicesCmd = &cobra.Command{
	Use:   "services",
	Short: "Run services processor",
	Run: func(cmd *cobra.Command, args []string) {
		err := processors.All(systemInfo, []string{"services"})
		if err != nil {
			log.With("err", err).Errorf("Error running services processor")
			os.Exit(1)
		}
	},
}

var runFilesCmd = &cobra.Command{
	Use:   "files",
	Short: "Run files processor",
	Run: func(cmd *cobra.Command, args []string) {
		err := processors.All(systemInfo, []string{"files"})
		if err != nil {
			log.With("err", err).Errorf("Error running files processor")
			os.Exit(1)
		}
	},
}

var runDirectoriesCmd = &cobra.Command{
	Use:   "directories",
	Short: "Run directories processor",
	Run: func(cmd *cobra.Command, args []string) {
		err := processors.All(systemInfo, []string{"directories"})
		if err != nil {
			log.With("err", err).Errorf("Error running directories processor")
			os.Exit(1)
		}
	},
}

var runTemplatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "Run templates processor",
	Run: func(cmd *cobra.Command, args []string) {
		err := processors.All(systemInfo, []string{"templates"})
		if err != nil {
			log.With("err", err).Errorf("Error running templates processor")
			os.Exit(1)
		}
	},
}

var runConfigurationCmd = &cobra.Command{
	Use:   "configuration",
	Short: "Run configuration processor",
	Run: func(cmd *cobra.Command, args []string) {
		err := processors.All(systemInfo, []string{"configuration"})
		if err != nil {
			log.With("err", err).Errorf("Error running configuration processor")
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.AddCommand(runPackageCmd)
	runCmd.AddCommand(runRepositoryCmd)
	runCmd.AddCommand(runServicesCmd)
	runCmd.AddCommand(runFilesCmd)
	runCmd.AddCommand(runDirectoriesCmd)
	runCmd.AddCommand(runTemplatesCmd)
	runCmd.AddCommand(runConfigurationCmd)
}
