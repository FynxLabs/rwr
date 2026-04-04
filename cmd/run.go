package cmd

import (
	"fmt"

	"github.com/fynxlabs/rwr/internal/processors"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run individual processors",
}

var runPackageCmd = &cobra.Command{
	Use:   "packages",
	Short: "Run packages processor",
	RunE: func(cmd *cobra.Command, args []string) error {
		return processors.All(initConfig, osInfo, []string{"packages"})
	},
}

var runRepositoryCmd = &cobra.Command{
	Use:   "repository",
	Short: "Run repository processor",
	RunE: func(cmd *cobra.Command, args []string) error {
		return processors.All(initConfig, osInfo, []string{"repositories"})
	},
}

var runServicesCmd = &cobra.Command{
	Use:   "services",
	Short: "Run services processor",
	RunE: func(cmd *cobra.Command, args []string) error {
		return processors.All(initConfig, osInfo, []string{"services"})
	},
}

var runFilesCmd = &cobra.Command{
	Use:   "files",
	Short: "Run files processor",
	RunE: func(cmd *cobra.Command, args []string) error {
		return processors.All(initConfig, osInfo, []string{"files"})
	},
}

var runDirectoriesCmd = &cobra.Command{
	Use:   "directories",
	Short: "Run directories processor",
	RunE: func(cmd *cobra.Command, args []string) error {
		return processors.All(initConfig, osInfo, []string{"directories"})
	},
}

var runConfigurationCmd = &cobra.Command{
	Use:   "configuration",
	Short: "Run configuration processor",
	RunE: func(cmd *cobra.Command, args []string) error {
		return processors.All(initConfig, osInfo, []string{"configuration"})
	},
}

var runUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "Run users processor",
	RunE: func(cmd *cobra.Command, args []string) error {
		return processors.All(initConfig, osInfo, []string{"users"})
	},
}

var runGitCmd = &cobra.Command{
	Use:   "git",
	Short: "Run git processor",
	RunE: func(cmd *cobra.Command, args []string) error {
		return processors.All(initConfig, osInfo, []string{"git"})
	},
}

var runScriptsCmd = &cobra.Command{
	Use:   "scripts",
	Short: "Run scripts processor",
	RunE: func(cmd *cobra.Command, args []string) error {
		return processors.All(initConfig, osInfo, []string{"scripts"})
	},
}

var runSSHKeysCmd = &cobra.Command{
	Use:   "ssh_keys",
	Short: "Run SSH key processor",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle GitHub OAuth authentication if --gh-auth flag is set
		if ghAuth {
			token, err := processors.AuthenticateWithGitHub(initConfig)
			if err != nil {
				return fmt.Errorf("GitHub authentication failed: %w", err)
			}
			// Update the token in both global var and initConfig
			ghApiToken = token
			initConfig.Variables.Flags.GHAPIToken = token
		}

		return processors.All(initConfig, osInfo, []string{"ssh_keys"})
	},
}

var runFontsCmd = &cobra.Command{
	Use:   "fonts",
	Short: "Run fonts processor",
	RunE: func(cmd *cobra.Command, args []string) error {
		return processors.All(initConfig, osInfo, []string{"fonts"})
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
	runCmd.AddCommand(runFontsCmd)
}
