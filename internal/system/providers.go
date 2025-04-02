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

	// Initialize providers if needed
	if err := InitProviders(); err != nil {
		log.Errorf("GetAvailableProviders: Error initializing providers: %v", err)
		return available
	}

	// Get current OS
	currentOS := runtime.GOOS

	// Get Linux distribution if on Linux
	var currentDistro string
	if currentOS == "linux" {
		currentDistro = getLinuxDistro()
	}

	// Check each provider
	for name, provider := range providers {
		// Check if provider supports current OS/distro
		supportsSystem := false
		for _, dist := range provider.Detection.Distributions {
			// For Linux, any provider that supports "linux" works for all distros
			if currentOS == "linux" && (dist == "linux" || dist == currentDistro || IsDistroInFamily(currentDistro, dist)) {
				log.Debugf("Provider %s supports Linux (dist: %s)", name, dist)
				supportsSystem = true
				break
			}

			// For non-Linux, check exact OS match
			if dist == currentOS {
				log.Debugf("Provider %s supports platform %s", name, currentOS)
				supportsSystem = true
				break
			}
		}
		if !supportsSystem {
			log.Debugf("Provider %s does not support current system %s/%s", name, currentOS, currentDistro)
			continue
		}

		// Check if binary exists using FindTool
		tool := FindTool(provider.Detection.Binary)
		if !tool.Exists {
			continue
		}
		binPath := tool.Bin

		// Check if required files exist
		filesExist := true
		for _, file := range provider.Detection.Files {
			// Expand ~ to home directory if needed
			if file[0] == '~' {
				home, err := os.UserHomeDir()
				if err != nil {
					continue
				}
				file = filepath.Join(home, file[1:])
			}
			if _, err := os.Stat(file); err != nil {
				filesExist = false
				break
			}
		}
		if !filesExist {
			continue
		}

		// Provider is available
		provider.BinPath = binPath
		available[name] = provider
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

// getProviderNames returns a sorted list of provider names for logging
func getProviderNames() []string {
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	return names
}
