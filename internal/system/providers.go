package system

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/types"
)

var (
	providers     map[string]*types.Provider
	providersInit bool
)

// InitProviders initializes the providers map if not already initialized
func InitProviders() error {
	if providersInit {
		return nil
	}

	providers = make(map[string]*types.Provider)

	// First try loading embedded providers
	embeddedProvs, err := LoadEmbeddedProviders()
	if err != nil {
		log.Errorf("Failed to load embedded providers: %v", err)
		// Don't return error here, try filesystem providers first
	} else {
		log.Debugf("Loaded %d embedded providers", len(embeddedProvs))
		for name, provider := range embeddedProvs {
			providers[name] = provider
			log.Debugf("Added embedded provider: %s", name)
		}
	}

	// Then try filesystem providers (these will override embedded ones)
	providersPath, err := GetProvidersPath()
	if err != nil {
		log.Debugf("No filesystem providers found: %v", err)
	} else {
		if err := LoadProviders(providersPath); err != nil {
			log.Warnf("Failed to load filesystem providers: %v", err)
		}
	}

	providersInit = true

	// Only return error if we have no providers at all
	if len(providers) == 0 {
		return fmt.Errorf("no providers found (embedded or filesystem)")
	}
	return nil
}

// PackageManagerInfo represents a package manager's configuration
type PackageManagerInfo struct {
	Name     string
	Bin      string
	List     string
	Search   string
	Install  string
	Remove   string
	Update   string
	Clean    string
	Elevated bool
}

// GetPackageManagerInfo converts a provider's commands into PackageManagerInfo
func GetPackageManagerInfo(provider *types.Provider, binPath string) PackageManagerInfo {
	return PackageManagerInfo{
		Name:     provider.Name,
		Bin:      binPath,
		List:     fmt.Sprintf("%s %s", binPath, provider.Commands.List),
		Search:   fmt.Sprintf("%s %s", binPath, provider.Commands.Search),
		Install:  fmt.Sprintf("%s %s", binPath, provider.Commands.Install),
		Remove:   fmt.Sprintf("%s %s", binPath, provider.Commands.Remove),
		Update:   fmt.Sprintf("%s %s", binPath, provider.Commands.Update),
		Clean:    fmt.Sprintf("%s %s", binPath, provider.Commands.Clean),
		Elevated: provider.Elevated,
	}
}

// GetAvailableProviders returns providers that match the current system and are available
func GetAvailableProviders() map[string]*types.Provider {
	available := make(map[string]*types.Provider)

	log.Debugf("GetAvailableProviders: Starting provider detection")

	// Initialize providers if needed
	if err := InitProviders(); err != nil {
		log.Errorf("GetAvailableProviders: Error initializing providers: %v", err)
		return available
	}

	log.Debugf("GetAvailableProviders: Loaded %d total providers: %v", len(providers), getProviderNames())

	// Get current OS
	currentOS := runtime.GOOS
	log.Debugf("GetAvailableProviders: Current OS: %s", currentOS)

	// Get Linux distribution if on Linux
	var currentDistro string
	if currentOS == "linux" {
		currentDistro = getLinuxDistro()
		log.Debugf("GetAvailableProviders: Current Linux distribution: %s", currentDistro)
	}

	// Check each provider
	for name, provider := range providers {
		log.Debugf("GetAvailableProviders: Checking provider %s", name)
		log.Debugf("GetAvailableProviders: Provider %s binary: %s", name, provider.Detection.Binary)
		log.Debugf("GetAvailableProviders: Provider %s supported distributions: %v", name, provider.Detection.Distributions)
		log.Debugf("GetAvailableProviders: Provider %s required files: %v", name, provider.Detection.Files)

		// Check if provider supports current OS/distro
		supportsSystem := false
		for _, dist := range provider.Detection.Distributions {
			// For Linux, any provider that supports "linux" works for all distros
			if currentOS == "linux" && (dist == "linux" || dist == currentDistro || IsDistroInFamily(currentDistro, dist)) {
				log.Debugf("GetAvailableProviders: Provider %s supports Linux (dist: %s matches %s)", name, dist, currentDistro)
				supportsSystem = true
				break
			}

			// For non-Linux, check exact OS match
			if dist == currentOS {
				log.Debugf("GetAvailableProviders: Provider %s supports platform %s", name, currentOS)
				supportsSystem = true
				break
			}
		}
		if !supportsSystem {
			log.Debugf("GetAvailableProviders: Provider %s does not support current system %s/%s", name, currentOS, currentDistro)
			continue
		}

		log.Debugf("GetAvailableProviders: Provider %s is system-compatible, checking binary availability", name)

		// Check if binary exists using FindTool
		tool := FindTool(provider.Detection.Binary)
		if !tool.Exists {
			log.Debugf("GetAvailableProviders: Provider %s binary '%s' not found in PATH", name, provider.Detection.Binary)
			continue
		}
		binPath := tool.Bin
		log.Debugf("GetAvailableProviders: Provider %s binary found at: %s", name, binPath)

		// Check if required files exist
		filesExist := true
		var missingFiles []string
		for _, file := range provider.Detection.Files {
			// Expand ~ to home directory if needed
			expandedFile := file
			if file[0] == '~' {
				home, err := os.UserHomeDir()
				if err != nil {
					log.Debugf("GetAvailableProviders: Provider %s - error getting home directory for file %s: %v", name, file, err)
					filesExist = false
					missingFiles = append(missingFiles, file)
					continue
				}
				expandedFile = filepath.Join(home, file[1:])
			}
			if _, err := os.Stat(expandedFile); err != nil {
				log.Debugf("GetAvailableProviders: Provider %s required file missing: %s (expanded: %s) - %v", name, file, expandedFile, err)
				filesExist = false
				missingFiles = append(missingFiles, file)
			} else {
				log.Debugf("GetAvailableProviders: Provider %s required file found: %s", name, expandedFile)
			}
		}
		if !filesExist {
			log.Debugf("GetAvailableProviders: Provider %s missing required files: %v", name, missingFiles)
			continue
		}

		// Provider is available
		provider.BinPath = binPath
		available[name] = provider
		log.Infof("GetAvailableProviders: Provider %s is available with binary at %s", name, binPath)
	}

	log.Debugf("GetAvailableProviders: Found %d available providers: %v", len(available), getAvailableProviderNames(available))
	if len(available) == 0 {
		log.Errorf("GetAvailableProviders: No package managers detected! Current system: %s/%s", currentOS, currentDistro)
		logDetectionSummary(currentOS, currentDistro)
	}

	return available
}

// GetProvider returns a provider by name if it exists and is available
func GetProvider(name string) (*types.Provider, bool) {
	log.Debugf("Getting provider for %s", name)

	log.Debugf("GetProvider: Looking for provider %s", name)

	// Initialize providers if needed
	if err := InitProviders(); err != nil {
		log.Errorf("GetProvider: Error initializing providers: %v", err)
		return nil, false
	}

	// Check if the provider exists in the loaded providers
	provider, exists := providers[name]
	if !exists {
		names := make([]string, 0, len(providers))
		for name := range providers {
			names = append(names, name)
		}
		log.Errorf("GetProvider: Provider %s not found in loaded providers. Available providers: %v", name, names)
		return nil, false
	}
	log.Debugf("GetProvider: Found provider %s in loaded providers", name)

	// Check if binary exists using FindTool
	tool := FindTool(provider.Detection.Binary)
	if !tool.Exists {
		log.Errorf("GetProvider: Binary %s not found for provider %s", provider.Detection.Binary, name)
		return nil, false
	}
	binPath := tool.Bin

	// Provider is available
	provider.BinPath = binPath
	log.Debugf("GetProvider: Provider %s is available with binary at %s", name, binPath)
	return provider, true
}

// GetProvidersPath returns the absolute path to the providers directory
func GetProvidersPath() (string, error) {
	// Get the executable's directory
	execDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", err
	}

	// First check current working directory
	cwd, err := os.Getwd()
	if err == nil {
		providerPath := filepath.Join(cwd, "providers")
		if _, err := os.Stat(providerPath); err == nil {
			log.Debugf("Found providers directory in current working directory: %s", providerPath)
			return providerPath, nil
		}
	}

	// Check common locations for the providers directory
	locations := []string{
		filepath.Join(execDir, "providers"),                       // Next to executable
		"/usr/local/share/rwr/providers",                          // System-wide installation
		"/usr/share/rwr/providers",                                // System-wide installation
		filepath.Join(os.Getenv("HOME"), ".config/rwr/providers"), // RWR Config Path
	}

	// Add macOS-specific paths
	if runtime.GOOS == "darwin" {
		locations = append(locations,
			"/opt/homebrew/share/rwr/providers", // Homebrew on Apple Silicon
			"/usr/local/Cellar/rwr/providers",   // Homebrew on Intel
			"/Applications/rwr/providers",       // App bundle
		)
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			log.Debugf("Found providers directory at: %s", loc)
			return loc, nil
		}
	}

	return "", fmt.Errorf("providers directory not found in common locations")
}

// LoadProviders loads all provider definitions from the given directory
func LoadProviders(definitionsPath string) error {
	log.Debugf("LoadProviders: Loading providers from %s", definitionsPath)
	entries, err := os.ReadDir(definitionsPath)
	if err != nil {
		return fmt.Errorf("error reading definitions directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".toml" {
			path := filepath.Join(definitionsPath, entry.Name())
			log.Debugf("LoadProviders: Loading provider from %s", path)
			provider, err := LoadProviderDefinition(path)
			if err != nil {
				log.Errorf("LoadProviders: Error loading provider %s: %v", entry.Name(), err)
				return fmt.Errorf("error loading provider %s: %w", entry.Name(), err)
			}
			log.Debugf("LoadProviders: Successfully loaded provider %s with binary %s", provider.Name, provider.Detection.Binary)
			providers[provider.Name] = provider
		}
	}
	count := len(providers)
	log.Debugf("LoadProviders: Loaded %d providers: %v", count, getProviderNames())
	return nil
}

// LoadProviderDefinition loads a provider definition from a file
func LoadProviderDefinition(path string) (*types.Provider, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// The TOML has a [provider] section
	var config struct {
		Provider types.Provider `toml:"provider"`
	}
	if _, err := toml.Decode(string(data), &config); err != nil {
		log.Errorf("LoadProviderDefinition: Failed to decode TOML %s: %v", path, err)
		return nil, fmt.Errorf("failed to decode TOML: %w", err)
	}

	provider := config.Provider

	// Ensure provider name is set from the TOML
	if provider.Name == "" {
		log.Errorf("LoadProviderDefinition: Provider name not set in %s", path)
		return nil, fmt.Errorf("provider name not set in %s", path)
	}

	log.Debugf("LoadProviderDefinition: Loaded provider %s with binary %s", provider.Name, provider.Detection.Binary)
	return &provider, nil
}

// GetDefaultProviderFromOSRelease returns the default provider based on /etc/os-release
func GetDefaultProviderFromOSRelease() string {
	// Read the contents of the /etc/os-release file
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		log.Warnf("Error reading /etc/os-release file: %s", err)
		return ""
	}

	// Parse the contents of the file
	osRelease := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
			osRelease[key] = value
		}
	}

	// Check the ID field first
	id := osRelease["ID"]
	if id != "" {
		if prov, exists := GetProviderForDistro(id); exists {
			return prov.Name
		}
	}

	// If ID doesn't match any known distribution, check ID_LIKE
	idLike := osRelease["ID_LIKE"]
	if idLike != "" {
		for _, distro := range strings.Split(idLike, " ") {
			if prov, exists := GetProviderForDistro(distro); exists {
				return prov.Name
			}
		}
	}

	return ""
}

// GetProviderForDistro returns a provider that supports the given distribution
func GetProviderForDistro(distro string) (*types.Provider, bool) {
	// Initialize providers if needed
	if err := InitProviders(); err != nil {
		log.Errorf("GetProviderForDistro: Error initializing providers: %v", err)
		return nil, false
	}

	// Check each provider's supported distributions
	for _, provider := range providers {
		for _, dist := range provider.Detection.Distributions {
			// For Linux, any provider that supports "linux" works for all distros
			if runtime.GOOS == "linux" && (dist == "linux" || dist == distro || IsDistroInFamily(distro, dist)) {
				if tool := FindTool(provider.Detection.Binary); tool.Exists {
					provider.BinPath = tool.Bin
					return provider, true
				}
			}

			// For non-Linux, check exact OS match
			if dist == runtime.GOOS {
				if tool := FindTool(provider.Detection.Binary); tool.Exists {
					provider.BinPath = tool.Bin
					return provider, true
				}
			}
		}
	}
	return nil, false
}

// GetProviderWithAlternatives returns a provider with distribution-specific packages resolved
func GetProviderWithAlternatives(name string) (*types.Provider, bool) {
	provider, exists := GetProvider(name)
	if !exists {
		return nil, false
	}

	// Get current distribution
	var currentDistro string
	if runtime.GOOS == "linux" {
		currentDistro = getLinuxDistro()
	}

	// Create a copy of the provider to avoid modifying the original
	providerCopy := *provider

	// If we have alternatives for this distribution, apply them
	if currentDistro != "" && provider.HasAlternativesForDistro(currentDistro) {
		providerCopy.CorePackages = provider.GetCorePackagesForDistro(currentDistro)
		log.Debugf("Applied alternatives for distribution %s to provider %s", currentDistro, name)
	}

	return &providerCopy, true
}

// GetProviderForDistroWithAlternatives returns a provider for a specific distribution with alternatives applied
func GetProviderForDistroWithAlternatives(distro string) (*types.Provider, bool) {
	provider, exists := GetProviderForDistro(distro)
	if !exists {
		return nil, false
	}

	// Create a copy of the provider to avoid modifying the original
	providerCopy := *provider

	// If we have alternatives for this distribution, apply them
	if provider.HasAlternativesForDistro(distro) {
		providerCopy.CorePackages = provider.GetCorePackagesForDistro(distro)
		log.Debugf("Applied alternatives for distribution %s to provider %s", distro, provider.Name)
	}

	return &providerCopy, true
}

// getProviderNames returns a sorted list of provider names for logging
func getProviderNames() []string {
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	return names
}

// getAvailableProviderNames returns a list of available provider names for logging
func getAvailableProviderNames(available map[string]*types.Provider) []string {
	names := make([]string, 0, len(available))
	for name := range available {
		names = append(names, name)
	}
	return names
}

// logDetectionSummary provides a detailed summary of why provider detection failed
func logDetectionSummary(currentOS, currentDistro string) {
	log.Errorf("=== PROVIDER DETECTION SUMMARY ===")
	log.Errorf("System: %s", currentOS)
	if currentOS == "linux" {
		log.Errorf("Distribution: %s", currentDistro)
	}
	log.Errorf("Total providers loaded: %d", len(providers))

	for name, provider := range providers {
		log.Errorf("Provider: %s", name)
		log.Errorf("  Binary: %s", provider.Detection.Binary)
		log.Errorf("  Supported distributions: %v", provider.Detection.Distributions)
		log.Errorf("  Required files: %v", provider.Detection.Files)

		// Check system compatibility
		compatible := false
		for _, dist := range provider.Detection.Distributions {
			if currentOS == "linux" && (dist == "linux" || dist == currentDistro || IsDistroInFamily(currentDistro, dist)) {
				compatible = true
				break
			}
			if dist == currentOS {
				compatible = true
				break
			}
		}
		log.Errorf("  System compatible: %v", compatible)

		if compatible {
			// Check binary
			tool := FindTool(provider.Detection.Binary)
			log.Errorf("  Binary found: %v", tool.Exists)
			if tool.Exists {
				log.Errorf("  Binary path: %s", tool.Bin)
			}

			// Check files
			allFilesExist := true
			for _, file := range provider.Detection.Files {
				expandedFile := file
				if file[0] == '~' {
					if home, err := os.UserHomeDir(); err == nil {
						expandedFile = filepath.Join(home, file[1:])
					}
				}
				if _, err := os.Stat(expandedFile); err != nil {
					log.Errorf("  Missing file: %s", expandedFile)
					allFilesExist = false
				}
			}
			log.Errorf("  All files exist: %v", allFilesExist)
		}
		log.Errorf("  ---")
	}
	log.Errorf("=== END DETECTION SUMMARY ===")
}
