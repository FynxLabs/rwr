package processors

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// ProcessPackages processes package management operations
func ProcessPackages(data []byte, packages *types.PackagesData, format string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	// If data is provided, unmarshal it
	if data != nil {
		var pkgData types.PackagesData
		if err := helpers.UnmarshalBlueprint(data, format, &pkgData); err != nil {
			return fmt.Errorf("error unmarshaling package blueprint: %w", err)
		}
		packages = &pkgData
	}

	// If no packages provided, nothing to do
	if packages == nil || len(packages.Packages) == 0 {
		return nil
	}

	// Process imports and merge imported packages
	blueprintDir := initConfig.Init.Location
	allPackages := make([]types.Package, 0)
	visited := make(map[string]bool)

	for _, pkg := range packages.Packages {
		if pkg.Import != "" {
			// This is an import directive
			log.Debugf("Processing package import: %s", pkg.Import)

			importPath := filepath.Join(blueprintDir, pkg.Import)
			absPath, err := filepath.Abs(importPath)
			if err != nil {
				return fmt.Errorf("error resolving import path %s: %w", importPath, err)
			}

			// Check for circular import
			if visited[absPath] {
				log.Warnf("Circular import detected, skipping: %s", absPath)
				continue
			}
			visited[absPath] = true

			// Read the import file
			importData, err := os.ReadFile(importPath)
			if err != nil {
				return fmt.Errorf("error reading import file %s: %w", importPath, err)
			}

			// Determine format from file extension if not explicitly provided
			fileFormat := format
			if fileFormat == "" {
				ext := filepath.Ext(importPath)
				fileFormat = ext
			}

			// Unmarshal the imported package data
			var importedPkgData types.PackagesData
			if err := helpers.UnmarshalBlueprint(importData, fileFormat, &importedPkgData); err != nil {
				return fmt.Errorf("error unmarshaling import file %s: %w", importPath, err)
			}

			// Add imported packages to our list
			allPackages = append(allPackages, importedPkgData.Packages...)
			log.Debugf("Imported %d packages from %s", len(importedPkgData.Packages), pkg.Import)
		} else {
			// Regular package entry
			allPackages = append(allPackages, pkg)
		}
	}

	// Update packages with merged list
	packages.Packages = allPackages

	// Initialize providers if needed
	if err := system.InitProviders(); err != nil {
		return fmt.Errorf("error initializing providers: %w", err)
	}

	// Get available providers
	available := system.GetAvailableProviders()
	if len(available) == 0 {
		return fmt.Errorf("no package managers available - check debug logs for detailed provider detection information. Common issues: missing binaries in PATH, missing config files, or unsupported platform")
	}

	// Filter packages based on active profiles
	filteredPackages := helpers.FilterByProfiles(packages.Packages, initConfig.Variables.Flags.Profiles)

	log.Debugf("Filtering packages: %d total, %d matching active profiles %v",
		len(packages.Packages), len(filteredPackages), initConfig.Variables.Flags.Profiles)

	// Process each filtered package
	for _, pkg := range filteredPackages {
		// Get provider
		var provider *types.Provider
		var exists bool

		if pkg.PackageManager != "" {
			// Use specified package manager
			provider, exists = system.GetProvider(pkg.PackageManager)
			if !exists {
				log.Warnf("Specified package manager %s not available, skipping package %s", pkg.PackageManager, pkg.Name)
				continue
			}
		} else {
			for _, p := range available {
				provider = p
				break
			}
		}

		// Get package names
		var names []string
		if pkg.Name != "" {
			names = []string{pkg.Name}
		} else {
			names = pkg.Names
		}

		// Process each package
		for _, name := range names {
			// Build command arguments
			var args []string
			switch pkg.Action {
			case "install":
				args = append(args, strings.Fields(provider.Commands.Install)...)
			case "remove":
				args = append(args, strings.Fields(provider.Commands.Remove)...)
			default:
				log.Warnf("Unknown action %s for package %s", pkg.Action, name)
				continue
			}

			// Add package name
			args = append(args, name)

			// Add any additional arguments
			if len(pkg.Args) > 0 {
				args = append(args, pkg.Args...)
			}

			// Execute command directly with environment variables
			cmd := types.Command{
				Exec:      provider.BinPath,
				Args:      args,
				Elevated:  provider.Elevated,
				Variables: provider.Environment,
			}
			if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
				log.Warnf("Error %s package %s: %v", pkg.Action, name, err)
				continue
			}

			log.Infof("Successfully %sed package %s via %s", pkg.Action, name, provider.Name)
		}
	}

	return nil
}
