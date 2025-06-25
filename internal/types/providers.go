package types

// Provider represents a package manager provider
type Provider struct {
	Name         string                           `toml:"name"`
	Elevated     bool                             `toml:"elevated"`
	Detection    DetectionConfig                  `toml:"detection"`
	Commands     CommandConfig                    `toml:"commands"`
	Repository   RepositoryConfig                 `toml:"repository"`
	CorePackages map[string][]string              `toml:"corePackages"`
	Alternatives map[string]ProviderAlternatives  `toml:"alternatives"`
	Install      InstallConfig                    `toml:"install"`
	Remove       RemoveConfig                     `toml:"remove"`
	Environment  map[string]string                `toml:"environment"`
	BinPath      string
}

// ProviderAlternatives defines distribution-specific alternatives for package names
type ProviderAlternatives struct {
	CorePackages map[string][]string `toml:"corePackages"`
}

// InstallConfig defines steps for installing the package manager
type InstallConfig struct {
	Steps []ActionStep `toml:"steps"`
}

// RemoveConfig defines steps for removing the package manager
type RemoveConfig struct {
	Steps []ActionStep `toml:"steps"`
}

// DetectionConfig defines how to detect if a provider is available
type DetectionConfig struct {
	Binary        string   `toml:"binary"`
	Files         []string `toml:"files"`
	Distributions []string `toml:"distributions"`
}

// CommandConfig defines the standard commands for a package manager
type CommandConfig struct {
	Install string `toml:"install"`
	Update  string `toml:"update"`
	Remove  string `toml:"remove"`
	List    string `toml:"list"`
	Search  string `toml:"search"`
	Clean   string `toml:"clean"`
}

// RepositoryConfig defines repository management configuration
type RepositoryConfig struct {
	Paths  RepositoryPaths  `toml:"paths"`
	Add    RepositoryAction `toml:"add"`
	Remove RepositoryAction `toml:"remove"`
}

// RepositoryPaths defines paths used for repository management
type RepositoryPaths struct {
	Sources string `toml:"sources"`
	Keys    string `toml:"keys"`
	Config  string `toml:"config"`
}

// RepositoryAction defines steps for repository operations
type RepositoryAction struct {
	Steps []ActionStep `toml:"steps"`
}

// ActionStep defines a single step in a repository operation
type ActionStep struct {
	Action  string   `toml:"action"`
	Source  string   `toml:"source,omitempty"`
	Dest    string   `toml:"dest,omitempty"`
	Exec    string   `toml:"exec,omitempty"`
	Args    []string `toml:"args,omitempty"`
	Content string   `toml:"content,omitempty"`
}

// GetCorePackagesForDistro returns the appropriate core packages for a given distribution
// It will use alternatives if available, otherwise fall back to default packages
func (p *Provider) GetCorePackagesForDistro(distro string) map[string][]string {
	// Check if we have alternatives for this distribution
	if alternatives, exists := p.Alternatives[distro]; exists {
		// Merge default packages with alternatives, with alternatives taking precedence
		result := make(map[string][]string)

		// Start with default packages
		for key, packages := range p.CorePackages {
			result[key] = packages
		}

		// Override with alternatives
		for key, packages := range alternatives.CorePackages {
			result[key] = packages
		}

		return result
	}

	// No alternatives found, return default packages
	return p.CorePackages
}

// HasAlternativesForDistro checks if the provider has alternatives for the given distribution
func (p *Provider) HasAlternativesForDistro(distro string) bool {
	_, exists := p.Alternatives[distro]
	return exists
}
