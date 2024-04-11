package types

// OSInfo holds information about the detected OS, package managers, and tools.
type OSInfo struct {
	OS             string         // Operating system detected
	PackageManager PackageManager // Package managers available
	Tools          ToolList       // Common tools
}

// InitConfig represents the configuration for the initialization processor.
type InitConfig struct {
	Init            Init                   `mapstructure:"blueprints"`
	PackageManagers []PackageManagerInfo   `mapstructure:"packageManagers,omitempty"`
	Repositories    []Repository           `mapstructure:"repositories,omitempty"`
	Packages        []Package              `mapstructure:"packages,omitempty"`
	Services        []Service              `mapstructure:"services,omitempty"`
	Files           []File                 `mapstructure:"files,omitempty"`
	Directories     []Directory            `mapstructure:"directories,omitempty"`
	Templates       []Template             `mapstructure:"templates,omitempty"`
	Configuration   []Configuration        `mapstructure:"configuration,omitempty"`
	Variables       map[string]interface{} `mapstructure:"variables,omitempty"`
}
