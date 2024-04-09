package types

// OSInfo holds information about the detected OS, package managers, and tools.
type OSInfo struct {
	OS             string         // Operating system detected
	PackageManager PackageManager // Package managers available
	Tools          ToolList       // Common tools
}

// InitConfig represents the configuration for the initialization processor.
type InitConfig struct {
	Blueprint       Blueprints             `yaml:"blueprint" json:"blueprint" toml:"blueprint"`
	PackageManagers []PackageManagerInfo   `yaml:"packageManagers" json:"packageManagers" toml:"packageManagers"`
	Repositories    []Repository           `yaml:"repositories" json:"repositories" toml:"repositories"`
	Packages        []Package              `yaml:"packages" json:"packages" toml:"packages"`
	Services        []Service              `yaml:"services" json:"services" toml:"services"`
	Files           []File                 `yaml:"files" json:"files" toml:"files"`
	Directories     []Directory            `mapstructure:"yaml" json:"directories" toml:"directories"`
	Templates       []Template             `yaml:"templates" json:"templates" toml:"templates"`
	Configuration   []Configuration        `yaml:"configuration" json:"configuration" toml:"configuration"`
	Variables       map[string]interface{} `yaml:"variables" json:"variables" toml:"variables"`
}
