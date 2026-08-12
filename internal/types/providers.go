package types

// Provider represents a package manager provider.
type Provider struct {
	Name         string                          `toml:"name"`
	Elevated     bool                            `toml:"elevated"`
	Detection    DetectionConfig                 `toml:"detection"`
	Commands     CommandConfig                   `toml:"commands"`
	Repository   RepositoryConfig                `toml:"repository"`
	CorePackages map[string][]string             `toml:"corePackages"`
	Alternatives map[string]ProviderAlternatives `toml:"alternatives"`
	Install      InstallConfig                   `toml:"install"`
	Remove       RemoveConfig                    `toml:"remove"`
	Environment  map[string]string               `toml:"environment"`
	BinPath      string
}

// ProviderAlternatives defines distribution-specific alternatives for package names.
type ProviderAlternatives struct {
	CorePackages map[string][]string `toml:"corePackages"`
}

// InstallConfig defines steps for installing the package manager.
type InstallConfig struct {
	Steps []ActionStep `toml:"steps"`
}

// RemoveConfig defines steps for removing the package manager.
type RemoveConfig struct {
	Steps []ActionStep `toml:"steps"`
}

// DetectionConfig defines how to detect if a provider is available.
type DetectionConfig struct {
	Binary        string   `toml:"binary"`
	Files         []string `toml:"files"`
	Distributions []string `toml:"distributions"`
}

// CommandConfig defines the standard commands for a package manager.
type CommandConfig struct {
	Install string `toml:"install"`
	Update  string `toml:"update"`
	Remove  string `toml:"remove"`
	List    string `toml:"list"`
	// ListExplicit is the manager's explicitly-installed query (pacman -Qe,
	// apt-mark showmanual); scan consumers prefer it over List.
	ListExplicit string `toml:"list_explicit"`
	Search       string `toml:"search"`
	Clean        string `toml:"clean"`
}

// RepositoryConfig defines repository management configuration.
type RepositoryConfig struct {
	Paths  RepositoryPaths  `toml:"paths"`
	Add    RepositoryAction `toml:"add"`
	Remove RepositoryAction `toml:"remove"`
}

// RepositoryPaths defines paths used for repository management.
type RepositoryPaths struct {
	Sources string `toml:"sources"`
	Keys    string `toml:"keys"`
	Config  string `toml:"config"`
}

// RepositoryAction defines steps for repository operations.
type RepositoryAction struct {
	Steps []ActionStep `toml:"steps"`
}

// ActionStep defines a single step in a repository operation.
type ActionStep struct {
	Action string `toml:"action"`
	// Path is what a "remove" step deletes. Provider definitions have always
	// spelled it "path"; without a field to decode into, every shipped remove step
	// arrived with nothing to act on.
	Path string `toml:"path,omitempty"`
	// Match is the line a "remove_line" step takes out of Path, Section the
	// ini-style section a "remove_section" step takes out of it.
	Match   string `toml:"match,omitempty"`
	Section string `toml:"section,omitempty"`
	Source  string `toml:"source,omitempty"`
	Dest    string `toml:"dest,omitempty"`
	// Sha256, when set on a "download" step, is the hex digest the fetched
	// content must match - the download is discarded on a mismatch. It is a
	// template like every other step field, so a provider can take it from the
	// blueprint with `sha256 = "{{ .SHA256 }}"`.
	Sha256  string   `toml:"sha256,omitempty"`
	Exec    string   `toml:"exec,omitempty"`
	Args    []string `toml:"args,omitempty"`
	Content string   `toml:"content,omitempty"`
	// Condition gates the step: it is a template rendered against the same data
	// as the rest of the step, and the step runs only when it renders to a
	// truthy value. Without a field to decode into, every shipped `condition`
	// was dropped and mutually exclusive steps all ran - flatpak added a remote
	// both --user and --system, chocolatey added a source twice.
	Condition string `toml:"condition,omitempty"`
	// Optional marks a step whose failure is logged but does not fail the
	// action. For steps that only exist on some tool versions: brew 6's
	// `brew trust` does not exist on older brew, and the tap add must not
	// fail there because the trust step after it cannot run.
	Optional bool `toml:"optional,omitempty"`
}

// GetCorePackagesForDistro returns the appropriate core packages for a given distribution
// It will use alternatives if available, otherwise fall back to default packages.
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

// HasAlternativesForDistro checks if the provider has alternatives for the given distribution.
func (p *Provider) HasAlternativesForDistro(distro string) bool {
	_, exists := p.Alternatives[distro]
	return exists
}
