package types

// Provider represents a package manager provider
type Provider struct {
	Name         string              `toml:"name"`
	Elevated     bool                `toml:"elevated"`
	Detection    DetectionConfig     `toml:"detection"`
	Commands     CommandConfig       `toml:"commands"`
	Repository   RepositoryConfig    `toml:"repository"`
	CorePackages map[string][]string `toml:"corePackages"`
	Install      InstallConfig       `toml:"install"`
	Remove       RemoveConfig        `toml:"remove"`
	Environment  map[string]string   `toml:"environment"`
	BinPath      string
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
