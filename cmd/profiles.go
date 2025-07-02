package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/fynxlabs/rwr/internal/helpers"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "Discover and list available profiles in your configuration",
	Long: `The profiles command analyzes your configuration files and lists all available profiles.
This helps you understand what profiles are defined and can be activated using the --profile flag.

Profiles allow you to selectively install packages and configurations based on different contexts
(work, personal, development, gaming, etc.). Items without profiles are considered "base" items
and are always included.`,
	Run: func(cmd *cobra.Command, args []string) {
		if initConfig == nil {
			log.Error("Configuration not initialized. Please ensure you have a valid init file.")
			return
		}

		// Collect all profiles from all types
		allProfiles := make(map[string]bool)

		// Get profiles from packages
		if initConfig.Packages != nil {
			profiles := helpers.GetUniqueProfiles(initConfig.Packages)
			for _, profile := range profiles {
				allProfiles[profile] = true
			}
		}

		// Get profiles from services
		if initConfig.Services != nil {
			profiles := helpers.GetUniqueProfiles(initConfig.Services)
			for _, profile := range profiles {
				allProfiles[profile] = true
			}
		}

		// Get profiles from files
		if initConfig.Files != nil {
			profiles := helpers.GetUniqueProfiles(initConfig.Files)
			for _, profile := range profiles {
				allProfiles[profile] = true
			}
		}

		// Get profiles from templates
		if initConfig.Templates != nil {
			profiles := helpers.GetUniqueProfiles(initConfig.Templates)
			for _, profile := range profiles {
				allProfiles[profile] = true
			}
		}

		// Get profiles from directories
		if initConfig.Directories != nil {
			profiles := helpers.GetUniqueProfiles(initConfig.Directories)
			for _, profile := range profiles {
				allProfiles[profile] = true
			}
		}

		// Get profiles from repositories
		if initConfig.Repositories != nil {
			profiles := helpers.GetUniqueProfiles(initConfig.Repositories)
			for _, profile := range profiles {
				allProfiles[profile] = true
			}
		}

		// Convert map to sorted slice
		var profilesList []string
		for profile := range allProfiles {
			profilesList = append(profilesList, profile)
		}
		sort.Strings(profilesList)

		// Display results
		if len(profilesList) == 0 {
			fmt.Println("No profiles found in your configuration.")
			fmt.Println("All items are base items and will always be included.")
		} else {
			fmt.Printf("Available profiles (%d found):\n\n", len(profilesList))
			for _, profile := range profilesList {
				fmt.Printf("  • %s\n", profile)
			}
			fmt.Println()
			fmt.Println("Usage examples:")
			fmt.Printf("  rwr run --profile %s\n", profilesList[0])
			if len(profilesList) > 1 {
				fmt.Printf("  rwr run --profile %s --profile %s\n", profilesList[0], profilesList[1])
				fmt.Printf("  rwr run --profile %s\n", strings.Join(profilesList[:2], ","))
			}
			fmt.Println("  rwr run --profile all")
		}

		// Display profile statistics
		fmt.Println()
		fmt.Println("Profile Statistics:")
		fmt.Printf("  Base items (no profiles): %d\n", countBaseItems())
		for _, profile := range profilesList {
			count := countProfileItems(profile)
			fmt.Printf("  %s: %d items\n", profile, count)
		}
	},
}

func countBaseItems() int {
	count := 0

	if initConfig.Packages != nil {
		for _, pkg := range initConfig.Packages {
			if len(pkg.GetProfiles()) == 0 {
				count++
			}
		}
	}

	if initConfig.Services != nil {
		for _, svc := range initConfig.Services {
			if len(svc.GetProfiles()) == 0 {
				count++
			}
		}
	}

	if initConfig.Files != nil {
		for _, file := range initConfig.Files {
			if len(file.GetProfiles()) == 0 {
				count++
			}
		}
	}

	if initConfig.Templates != nil {
		for _, tpl := range initConfig.Templates {
			if len(tpl.GetProfiles()) == 0 {
				count++
			}
		}
	}

	if initConfig.Directories != nil {
		for _, dir := range initConfig.Directories {
			if len(dir.GetProfiles()) == 0 {
				count++
			}
		}
	}

	if initConfig.Repositories != nil {
		for _, repo := range initConfig.Repositories {
			if len(repo.GetProfiles()) == 0 {
				count++
			}
		}
	}

	return count
}

func countProfileItems(targetProfile string) int {
	count := 0

	if initConfig.Packages != nil {
		for _, pkg := range initConfig.Packages {
			for _, profile := range pkg.GetProfiles() {
				if profile == targetProfile {
					count++
					break
				}
			}
		}
	}

	if initConfig.Services != nil {
		for _, svc := range initConfig.Services {
			for _, profile := range svc.GetProfiles() {
				if profile == targetProfile {
					count++
					break
				}
			}
		}
	}

	if initConfig.Files != nil {
		for _, file := range initConfig.Files {
			for _, profile := range file.GetProfiles() {
				if profile == targetProfile {
					count++
					break
				}
			}
		}
	}

	if initConfig.Templates != nil {
		for _, tpl := range initConfig.Templates {
			for _, profile := range tpl.GetProfiles() {
				if profile == targetProfile {
					count++
					break
				}
			}
		}
	}

	if initConfig.Directories != nil {
		for _, dir := range initConfig.Directories {
			for _, profile := range dir.GetProfiles() {
				if profile == targetProfile {
					count++
					break
				}
			}
		}
	}

	if initConfig.Repositories != nil {
		for _, repo := range initConfig.Repositories {
			for _, profile := range repo.GetProfiles() {
				if profile == targetProfile {
					count++
					break
				}
			}
		}
	}

	return count
}

func init() {
	rootCmd.AddCommand(profilesCmd)
}
