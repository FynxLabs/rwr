package types

// OSInfo holds information about the detected OS, package managers, and tools.
type OSInfo struct {
	OS             string         // Operating system detected
	PackageManager PackageManager // Package managers available
	Tools          ToolList       // Common tools
}

// InitConfig represents the configuration for the initialization processor.
type InitConfig struct {
	Init            Init                   `mapstructure:"blueprints" yaml:"blueprints,omitempty" json:"blueprints,omitempty" toml:"blueprints,omitempty"`
	PackageManagers []PackageManagerInfo   `mapstructure:"packageManagers,omitempty" yaml:"packageManagers,omitempty" json:"packageManagers,omitempty" toml:"packageManagers,omitempty"`
	Repositories    []Repository           `mapstructure:"repositories,omitempty" yaml:"repositories,omitempty" json:"repositories,omitempty" toml:"repositories,omitempty"`
	Packages        []Package              `mapstructure:"packages,omitempty" yaml:"packages,omitempty" json:"packages,omitempty" toml:"packages,omitempty"`
	Services        []Service              `mapstructure:"services,omitempty" yaml:"services,omitempty" json:"services,omitempty" toml:"services,omitempty"`
	Files           []File                 `mapstructure:"files,omitempty" yaml:"files,omitempty" json:"files,omitempty" toml:"files,omitempty"`
	Directories     []Directory            `mapstructure:"directories,omitempty" yaml:"directories,omitempty" json:"directories,omitempty" toml:"directories,omitempty"`
	Templates       []Template             `mapstructure:"templates,omitempty" yaml:"templates,omitempty" json:"templates,omitempty" toml:"templates,omitempty"`
	Configuration   []Configuration        `mapstructure:"configuration,omitempty" yaml:"configuration,omitempty" json:"configuration,omitempty" toml:"configuration,omitempty"`
	Variables       map[string]interface{} `mapstructure:"variables,omitempty" yaml:"variables,omitempty" json:"variables,omitempty" toml:"variables,omitempty"`
}
