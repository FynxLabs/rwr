package providers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/log"
)

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

var providers = make(map[string]*Provider)

// GetPackageManagerInfo converts a provider's commands into PackageManagerInfo
func GetPackageManagerInfo(provider *Provider, binPath string) PackageManagerInfo {
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
func GetAvailableProviders() map[string]*Provider {
	available := make(map[string]*Provider)

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
			if dist == currentOS || dist == currentDistro {
				supportsSystem = true
				break
			}
		}
		if !supportsSystem {
			continue
		}

		// Check if binary exists
		binPath, err := exec.LookPath(provider.Detection.Binary)
		if err != nil {
			continue
		}

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
		provider.BinPath = binPath // Store resolved binary path
		available[name] = provider
	}

	return available
}

// GetProvider returns a provider by name if it exists and is available
func GetProvider(name string) (*Provider, bool) {
	available := GetAvailableProviders()
	provider, exists := available[name]
	return provider, exists
}

// GetProviderForDistro returns a provider that supports the given distribution
func GetProviderForDistro(distro string) (*Provider, bool) {
	// Load providers if not already loaded
	if len(providers) == 0 {
		providersPath, err := GetProvidersPath()
		if err != nil {
			return nil, false
		}
		if err := LoadProviders(providersPath); err != nil {
			return nil, false
		}
	}

	// Check each provider's supported distributions
	for _, provider := range providers {
		for _, dist := range provider.Detection.Distributions {
			if dist == distro {
				// Check if binary exists
				if binPath, err := exec.LookPath(provider.Detection.Binary); err == nil {
					provider.BinPath = binPath
					return provider, true
				}
			}
		}
	}
	return nil, false
}

// LoadProviders loads all provider definitions from the given directory
func LoadProviders(definitionsPath string) error {
	// Clear existing providers
	providers = make(map[string]*Provider)

	entries, err := os.ReadDir(definitionsPath)
	if err != nil {
		return fmt.Errorf("error reading definitions directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".toml" {
			path := filepath.Join(definitionsPath, entry.Name())
			provider, err := loadProviderDefinition(path)
			if err != nil {
				return fmt.Errorf("error loading provider %s: %w", entry.Name(), err)
			}
			providers[provider.Name] = provider
		}
	}
	return nil
}

func loadProviderDefinition(path string) (*Provider, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var provider Provider
	if _, err := toml.Decode(string(data), &provider); err != nil {
		return nil, err
	}

	return &provider, nil
}

// GetProvidersPath returns the absolute path to the providers directory
func GetProvidersPath() (string, error) {
	// Get the executable's directory
	execDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", err
	}

	// Check common locations for the providers directory
	locations := []string{
		filepath.Join(execDir, "providers"),                // Next to executable
		"/usr/local/share/rwr/providers",                   // System-wide installation
		"/usr/share/rwr/providers",                         // System-wide installation
		filepath.Join(os.Getenv("HOME"), ".rwr/providers"), // User's home directory
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc, nil
		}
	}

	return "", fmt.Errorf("providers directory not found in common locations")
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

// getLinuxDistro returns the Linux distribution name from /etc/os-release
func getLinuxDistro() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}

	// Parse os-release file
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "ID=") {
			return strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
		}
	}

	return ""
}
