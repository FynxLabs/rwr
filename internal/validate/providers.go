package validate

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// ValidateProviders validates provider configuration files
func ValidateProviders(path string, verbose bool, results *types.ValidationResults, osInfo *types.OSInfo) error {
	log.Infof("Validating providers in %s", path)

	// Check if path is a file or directory
	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("error accessing path: %w", err)
	}

	// Load all provider definitions from the standard providers path for reference
	providersPath, err := system.GetProvidersPath()
	if err == nil {
		if err := system.LoadProviders(providersPath); err != nil {
			log.Debugf("Failed to load standard providers: %s", err)
		}
	}

	// Get available providers
	availableProviders := system.GetAvailableProviders()
	if len(availableProviders) == 0 {
		AddIssue(results, types.ValidationWarning, "No providers available for the current system", "", 0, "Install required package managers")
	}

	if fileInfo.IsDir() {
		// Validate all provider files in directory
		entries, err := os.ReadDir(path)
		if err != nil {
			return fmt.Errorf("error reading directory: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".toml" {
				providerPath := filepath.Join(path, entry.Name())
				if err := validateProviderFile(providerPath, results, osInfo); err != nil {
					AddIssue(results, types.ValidationError, fmt.Sprintf("Error validating provider file: %s", err), providerPath, 0, "")
				}
			}
		}
	} else {
		// Validate single provider file
		if filepath.Ext(path) != ".toml" {
			AddIssue(results, types.ValidationError, "Not a TOML file", path, 0, "Provider files must be TOML format")
			return nil
		}
		if err := validateProviderFile(path, results, osInfo); err != nil {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Error validating provider file: %s", err), path, 0, "")
		}
	}

	if verbose {
		log.Infof("Provider validation completed")
	}

	return nil
}

// validateProviderFile validates a provider configuration file
func validateProviderFile(providerFile string, results *types.ValidationResults, osInfo *types.OSInfo) error {
	log.Debugf("Validating provider file: %s", providerFile)

	// Read and parse the provider file
	provider, err := system.LoadProviderDefinition(providerFile)
	if err != nil {
		return fmt.Errorf("error loading provider definition: %w", err)
	}

	// Validate provider name
	if provider.Name == "" {
		AddIssue(results, types.ValidationError, "Missing required field 'provider.name'", providerFile, 0, "Add name field to provider section")
	}

	// Validate detection section
	if provider.Detection.Binary == "" {
		AddIssue(results, types.ValidationWarning, "Missing binary in detection section", providerFile, 0, "Add binary field to detection section")
	}

	if len(provider.Detection.Distributions) == 0 {
		AddIssue(results, types.ValidationWarning, "No distributions specified in detection section", providerFile, 0, "Add distributions to detection section")
	}

	// Validate commands section
	if provider.Commands.Install == "" {
		AddIssue(results, types.ValidationWarning, "Missing install command", providerFile, 0, "Add install command to commands section")
	}

	if provider.Commands.Update == "" {
		AddIssue(results, types.ValidationWarning, "Missing update command", providerFile, 0, "Add update command to commands section")
	}

	if provider.Commands.Remove == "" {
		AddIssue(results, types.ValidationWarning, "Missing remove command", providerFile, 0, "Add remove command to commands section")
	}

	// Validate repository section if present
	if provider.Repository.Paths.Sources != "" {
		// Only check paths for providers that support the current OS
		isSupported := false
		for _, dist := range provider.Detection.Distributions {
			if dist == "linux" || dist == osInfo.System.OSFamily {
				isSupported = true
				break
			}
		}

		if isSupported {
			// Check if sources path exists on the system
			if _, err := os.Stat(provider.Repository.Paths.Sources); os.IsNotExist(err) {
				AddIssue(results, types.ValidationWarning, fmt.Sprintf("Repository sources path does not exist: %s", provider.Repository.Paths.Sources), providerFile, 0, "")
			}
		}
	}

	// Validate install steps
	for i, step := range provider.Install.Steps {
		if step.Action == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing action in install step %d", i), providerFile, 0, "Add action field to install step")
		}

		validateActionStep(step, i, "install", providerFile, results)
	}

	// Validate remove steps
	for i, step := range provider.Remove.Steps {
		if step.Action == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing action in remove step %d", i), providerFile, 0, "Add action field to remove step")
		}

		validateActionStep(step, i, "remove", providerFile, results)
	}

	// Validate repository add steps
	for i, step := range provider.Repository.Add.Steps {
		if step.Action == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing action in repository add step %d", i), providerFile, 0, "Add action field to repository add step")
		}

		validateActionStep(step, i, "repository.add", providerFile, results)
	}

	// Validate repository remove steps
	for i, step := range provider.Repository.Remove.Steps {
		if step.Action == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing action in repository remove step %d", i), providerFile, 0, "Add action field to repository remove step")
		}

		validateActionStep(step, i, "repository.remove", providerFile, results)
	}

	return nil
}

// validateActionStep validates a provider action step
func validateActionStep(step types.ActionStep, index int, stepType string, file string, results *types.ValidationResults) {
	switch step.Action {
	case "command":
		if step.Exec == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing exec in %s step %d", stepType, index), file, 0, "Add exec field to command action")
		}
	case "download":
		if step.Source == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing source in %s step %d", stepType, index), file, 0, "Add source field to download action")
		}
		if step.Dest == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing dest in %s step %d", stepType, index), file, 0, "Add dest field to download action")
		}
	case "write":
		if step.Dest == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing dest in %s step %d", stepType, index), file, 0, "Add dest field to write action")
		}
		if step.Content == "" {
			AddIssue(results, types.ValidationWarning, fmt.Sprintf("Empty content in %s step %d", stepType, index), file, 0, "Add content to write action")
		}
	case "remove":
		if step.Dest == "" && step.Source == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing path in %s step %d", stepType, index), file, 0, "Add path field to remove action")
		}
	}
}
