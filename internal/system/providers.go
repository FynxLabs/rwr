package system

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"charm.land/log/v2"
	"github.com/BurntSushi/toml"
	"github.com/fynxlabs/rwr/internal/types"
)

var (
	providers     map[string]*types.Provider
	providersInit bool
	providersMu   sync.Mutex
)

// InitProviders loads provider definitions from embedded resources and the filesystem.
// Filesystem providers override embedded ones with the same name. This is a no-op
// if providers have already been initialized.
func InitProviders() error {
	providersMu.Lock()
	defer providersMu.Unlock()

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

	providersMu.Lock()
	defer providersMu.Unlock()

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

// isProviderAvailable checks if a provider is usable on the current system,
// returning the binary path and true if it is.
//
// Availability is decided by evidence on the machine rather than by what the
// distribution calls itself. A named distro match is accepted, but so is the
// combination of "the binary is installed" plus "every file the provider declares
// as proof of itself exists" — for pacman that is /etc/pacman.conf and
// /var/lib/pacman. There are far more Arch and Debian derivatives than any list
// can track (this was found on PrismLinux, which no provider names and which sets
// no ID_LIKE), and a machine that has pacman's binary and pacman's database is
// running pacman whatever /etc/os-release says.
//
// The reverse case is covered too: a system that calls itself Arch but installs
// packages with apt has no /etc/apt and no apt binary, so apt stays unavailable.
// The OS gate below still applies, so Windows providers never surface on Linux.
func isProviderAvailable(provider *types.Provider, currentOS, currentDistro string) (string, bool) {
	if !targetsOS(provider, currentOS) {
		log.Debugf("GetAvailableProviders: Provider %s does not target %s", provider.Name, currentOS)
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

	if supportsSystem(provider, currentOS, currentDistro) {
		return tool.Bin, true
	}

	// Unrecognised distribution. Accept it when the provider declares files that
	// prove it is genuinely in use, and say so — a provider with no such files has
	// only its distro list to go on, so it stays unavailable.
	if len(provider.Detection.Files) > 0 {
		log.Debugf("GetAvailableProviders: Provider %s not listed for %s/%s but its binary and required files are present; treating as available",
			provider.Name, currentOS, currentDistro)
		return tool.Bin, true
	}

	log.Debugf("GetAvailableProviders: Provider %s does not support %s/%s", provider.Name, currentOS, currentDistro)
	return "", false
}

// osTokens are the distribution entries that name an operating system rather than
// a Linux distribution.
var osTokens = map[string]bool{
	types.OSLinux:   true,
	types.OSDarwin:  true,
	types.OSWindows: true,
}

// targetsOS reports whether a provider is meant for the current operating system.
//
// A provider that names any OS token must name this one. A provider that lists
// only distributions (pacman, apt, dnf) is not OS-specific in its declaration, so
// the file and binary checks decide — those paths simply do not exist on the wrong
// OS.
func targetsOS(provider *types.Provider, currentOS string) bool {
	namedAnOS := false
	for _, dist := range provider.Detection.Distributions {
		if !osTokens[dist] {
			continue
		}
		namedAnOS = true
		if dist == currentOS {
			return true
		}
	}
	return !namedAnOS
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
		// Not an error: every caller asks about providers the machine may not have,
		// and "is brew installed?" is answered by the false return. Logging it at
		// error level filled `rwr validate` output with red lines about package
		// managers the operator never asked for.
		log.Debugf("GetProvider: Binary %s not found for provider %s", provider.Detection.Binary, name)
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
//
// The current working directory is deliberately not searched. A provider file
// declares exec, args and elevated=true and those are run verbatim, so honouring
// ./providers would hand root-level execution to any directory rwr happens to be
// run from — a cloned blueprint repo, /tmp, a shared downloads folder. Provider
// overrides belong in the user's config directory, which only the user can write.
func GetProvidersPath() (string, error) {
	// Get the executable's directory
	execDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", err
	}

	// Check common locations for the providers directory
	locations := []string{
		filepath.Join(execDir, "providers"), // Next to executable
		"/usr/local/share/rwr/providers",    // System-wide installation
		"/usr/share/rwr/providers",          // System-wide installation
	}

	// $HOME via Getenv would be empty under systemd/cron/sudo env -i (and always
	// on Windows), and joining "" yields a relative path that os.Stat resolves
	// against the CWD — exactly the directory this search must never honour.
	if home, err := os.UserHomeDir(); err == nil {
		locations = append(locations, filepath.Join(home, ".config/rwr/providers")) // RWR Config Path
	} else {
		log.Debugf("Skipping user provider path: no home directory: %v", err)
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
		// A relative candidate would resolve against the CWD; never probe one.
		if !filepath.IsAbs(loc) {
			continue
		}
		if _, err := os.Stat(loc); err == nil { // #nosec G703 -- TODO(PR8): path derived from operator blueprint input; containment added in PR8
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
			if !isProviderFileTrusted(path) {
				continue
			}
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

// isProviderFileTrusted reports whether a provider definition may be loaded.
//
// A provider's exec, args and elevated flag are run as written, so a definition
// anyone on the box can edit is a root shell for anyone on the box. Group- and
// world-writable files are skipped rather than rejected outright: one bad file in
// a directory should not cost the user every other provider in it.
func isProviderFileTrusted(path string) bool {
	return isProviderFileTrustedOn(runtime.GOOS, path)
}

// isProviderFileTrustedOn is isProviderFileTrusted with the platform as data,
// so both branches are testable from any platform.
func isProviderFileTrustedOn(goos, path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		log.Warnf("LoadProviders: Skipping provider %s: %v", path, err)
		return false
	}

	// Windows has no unix permission bits: Go synthesizes 0666 for any file
	// that is not read-only, so the writability check below would reject every
	// normal provider file — and its chmod advice is not runnable there. File
	// security on Windows is ACLs, which os.FileMode cannot express.
	if goos == "windows" {
		return true
	}

	if mode := info.Mode(); mode&0o022 != 0 {
		log.Warnf("LoadProviders: Skipping provider %s: it is group- or world-writable (mode %04o); "+
			"provider files run commands as root, so tighten it with chmod go-w", path, mode.Perm())
		return false
	}
	return true
}

// LoadProviderDefinition parses a single TOML provider definition file
// and returns the resulting Provider struct.
func LoadProviderDefinition(path string) (*types.Provider, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path is operator-supplied blueprint/config input; containment added in PR8
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

// getProviderNames returns a sorted list of provider names for logging.
func getProviderNames() []string {
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	return names
}

// getAvailableProviderNames returns a list of available provider names for logging.
func getAvailableProviderNames(available map[string]*types.Provider) []string {
	names := make([]string, 0, len(available))
	for name := range available {
		names = append(names, name)
	}
	return names
}

// logDetectionSummary provides a detailed summary of why provider detection failed.
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

// SetProvidersForTest replaces the loaded provider set and returns a function
// restoring the previous one.
//
// It exists because the processors resolve a package manager through the real
// provider registry, which asks the host what it has installed. Tests written
// against that answer pass on the machine they were written on and fail
// everywhere else: the schema and import tests passed on an Arch workstation and
// failed on CI's Ubuntu runner, because pacman is not there. A test asserting how
// rwr builds a command should not depend on which package manager the runner
// happens to have.
//
//	defer system.SetProvidersForTest(map[string]*types.Provider{...})()
func SetProvidersForTest(provs map[string]*types.Provider) (restore func()) {
	providersMu.Lock()
	defer providersMu.Unlock()

	previous, previousInit := providers, providersInit
	providers, providersInit = provs, true

	return func() {
		providersMu.Lock()
		defer providersMu.Unlock()
		providers, providersInit = previous, previousInit
	}
}
