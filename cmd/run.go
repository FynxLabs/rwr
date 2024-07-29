package cmd

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/processors"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run individual processors",
}

var runPackageCmd = &cobra.Command{
	Use:   "package",
	Short: "Run package processor",
	Run: func(cmd *cobra.Command, args []string) {

		err := processors.All(initConfig, osInfo, []string{"packages"})
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

		err := processors.All(initConfig, osInfo, []string{"repositories"})
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

		err := processors.All(initConfig, osInfo, []string{"services"})
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

		err := processors.All(initConfig, osInfo, []string{"files"})
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

		err := processors.All(initConfig, osInfo, []string{"directories"})
		if err != nil {
			log.With("err", err).Errorf("Error running directories processor")
			os.Exit(1)
		}
	},
}

var runConfigurationCmd = &cobra.Command{
	Use:   "configuration",
	Short: "Run configuration processor",
	Run: func(cmd *cobra.Command, args []string) {

		err := processors.All(initConfig, osInfo, []string{"configuration"})
		if err != nil {
			log.With("err", err).Errorf("Error running configuration processor")
			os.Exit(1)
		}
	},
}

var runUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "Run users processor",
	Run: func(cmd *cobra.Command, args []string) {

		err := processors.All(initConfig, osInfo, []string{"users"})
		if err != nil {
			log.With("err", err).Errorf("Error running users processor")
			os.Exit(1)
		}
	},
}

var runGitCmd = &cobra.Command{
	Use:   "git",
	Short: "Run git processor",
	Run: func(cmd *cobra.Command, args []string) {

		err := processors.All(initConfig, osInfo, []string{"git"})
		if err != nil {
			log.With("err", err).Errorf("Error running git processor")
			os.Exit(1)
		}
	},
}

var runScriptsCmd = &cobra.Command{
	Use:   "scripts",
	Short: "Run scripts processor",
	Run: func(cmd *cobra.Command, args []string) {

		err := processors.All(initConfig, osInfo, []string{"scripts"})
		if err != nil {
			log.With("err", err).Errorf("Error running scripts processor")
			os.Exit(1)
		}
	},
}

var runSSHKeysCmd = &cobra.Command{
	Use:   "ssh_keys",
	Short: "Run SSH key processor",
	Run: func(cmd *cobra.Command, args []string) {
		err := processors.All(initConfig, osInfo, []string{"ssh_keys"})
		if err != nil {
			log.With("err", err).Errorf("Error running SSH key processor")
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
	runCmd.AddCommand(runConfigurationCmd)
	runCmd.AddCommand(runUsersCmd)
	runCmd.AddCommand(runGitCmd)
	runCmd.AddCommand(runScriptsCmd)
	runCmd.AddCommand(runSSHKeysCmd)
}
