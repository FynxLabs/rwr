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

// InitProviders loads provider definitions from embedded resources and the filesystem.
// Filesystem providers override embedded ones with the same name. This is a no-op
// if providers have already been initialized.
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

// GetPackageManagerInfo builds a types.PackageManagerInfo from a provider definition
// by combining the binary path with each command template.
func GetPackageManagerInfo(provider *types.Provider, binPath string) types.PackageManagerInfo {
	return types.PackageManagerInfo{
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

// GetAvailableProviders returns providers whose binaries exist on the system and
// whose distribution lists match the current OS.
func GetAvailableProviders() map[string]*types.Provider {
	available := make(map[string]*types.Provider)

	if err := InitProviders(); err != nil {
		log.Errorf("GetAvailableProviders: Error initializing providers: %v", err)
		return available
	}

	currentOS, currentDistro := getSystemInfo()
	log.Debugf("GetAvailableProviders: Loaded %d providers, OS: %s, distro: %s", len(providers), currentOS, currentDistro)

	for name, provider := range providers {
		if binPath, ok := isProviderAvailable(provider, currentOS, currentDistro); ok {
			provider.BinPath = binPath
			available[name] = provider
			log.Infof("GetAvailableProviders: Provider %s is available with binary at %s", name, binPath)
		}
	}

	log.Debugf("GetAvailableProviders: Found %d available providers: %v", len(available), getAvailableProviderNames(available))
	if len(available) == 0 {
		log.Errorf("GetAvailableProviders: No package managers detected! Current system: %s/%s", currentOS, currentDistro)
		logDetectionSummary(currentOS, currentDistro)
	}

	return available
}

// getSystemInfo returns the current OS and Linux distribution (empty for non-Linux).
func getSystemInfo() (string, string) {
	currentOS := runtime.GOOS
	var currentDistro string
	if currentOS == "linux" {
		currentDistro = getLinuxDistro()
	}
	return currentOS, currentDistro
}

// isProviderAvailable checks if a provider is usable on the current system by
// verifying OS/distro support, binary availability, and required files.
// Returns the binary path and true if available.
func isProviderAvailable(provider *types.Provider, currentOS, currentDistro string) (string, bool) {
	if !supportsSystem(provider, currentOS, currentDistro) {
		log.Debugf("GetAvailableProviders: Provider %s does not support %s/%s", provider.Name, currentOS, currentDistro)
		return "", false
	}

	tool := FindTool(provider.Detection.Binary)
	if !tool.Exists {
		log.Debugf("GetAvailableProviders: Provider %s binary '%s' not found", provider.Name, provider.Detection.Binary)
		return "", false
	}

	if !areRequiredFilesPresent(provider) {
		return "", false
	}

	return tool.Bin, true
}

// supportsSystem checks if a provider's distribution list matches the current OS/distro.
func supportsSystem(provider *types.Provider, currentOS, currentDistro string) bool {
	for _, dist := range provider.Detection.Distributions {
		if currentOS == "linux" && (dist == types.OSLinux || dist == currentDistro || IsDistroInFamily(currentDistro, dist)) {
			return true
		}
		if dist == currentOS {
			return true
		}
	}
	return false
}

// areRequiredFilesPresent checks that all files listed in provider.Detection.Files exist.
func areRequiredFilesPresent(provider *types.Provider) bool {
	for _, file := range provider.Detection.Files {
		expandedFile := file
		if file[0] == '~' {
			home, err := os.UserHomeDir()
			if err != nil {
				log.Debugf("GetAvailableProviders: Provider %s - error expanding %s: %v", provider.Name, file, err)
				return false
			}
			expandedFile = filepath.Join(home, file[1:])
		}
		if _, err := os.Stat(expandedFile); err != nil {
			log.Debugf("GetAvailableProviders: Provider %s missing required file: %s", provider.Name, expandedFile)
			return false
		}
	}
	return true
}

// GetProvider returns a specific provider by name from the available providers.
// The second return value indicates whether the provider was found.
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

// GetProvidersPath returns the absolute path to the provider definitions directory,
// searching the executable's directory and common installation paths.
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

// LoadProviders reads all TOML provider definitions from the given directory
// and registers them in the global providers map.
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

// LoadProviderDefinition parses a single TOML provider definition file
// and returns the resulting Provider struct.
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

// GetDefaultProviderFromOSRelease determines the default package manager by
// reading the ID field from /etc/os-release and mapping it to a known provider.
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

// GetProviderForDistro returns the first available provider whose detection
// distributions list includes the given distro name.
func GetProviderForDistro(distro string) (*types.Provider, bool) {
	if err := InitProviders(); err != nil {
		log.Errorf("GetProviderForDistro: Error initializing providers: %v", err)
		return nil, false
	}

	currentOS := runtime.GOOS
	for _, provider := range providers {
		if supportsSystem(provider, currentOS, distro) {
			if tool := FindTool(provider.Detection.Binary); tool.Exists {
				provider.BinPath = tool.Bin
				return provider, true
			}
		}
	}
	return nil, false
}

// GetProviderWithAlternatives returns a provider by name with distro-specific
// alternative package names resolved for the current system.
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

// GetProviderForDistroWithAlternatives returns a provider matching the given
// distro with distro-specific alternative package names resolved.
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

		compatible := supportsSystem(provider, currentOS, currentDistro)
		log.Errorf("  System compatible: %v", compatible)

		if compatible {
			tool := FindTool(provider.Detection.Binary)
			log.Errorf("  Binary found: %v", tool.Exists)
			if tool.Exists {
				log.Errorf("  Binary path: %s", tool.Bin)
			}
			log.Errorf("  All files exist: %v", areRequiredFilesPresent(provider))
		}
		log.Errorf("  ---")
	}
	log.Errorf("=== END DETECTION SUMMARY ===")
}
