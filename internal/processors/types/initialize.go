package types

// OSInfo holds information about the detected OS, package managers, and tools.
type OSInfo struct {
	OS             string         // Operating system detected
	PackageManager PackageManager // Package managers available
	Tools          ToolList       // Common tools
}

// InitConfig represents the configuration for the initialization processor.
type InitConfig struct {
	Blueprint       Blueprints             `mapstructure:"blueprint"`
	PackageManagers []PackageManagerInfo   `mapstructure:"packageManagers"`
	Repositories    []Repository           `mapstructure:"repositories"`
	Packages        []Package              `mapstructure:"packages"`
	Services        []Service              `mapstructure:"services"`
	Files           []File                 `mapstructure:"files"`
	Directories     []Directory            `mapstructure:"directories"`
	Templates       []Template             `mapstructure:"templates"`
	Configuration   []Configuration        `mapstructure:"configuration"`
	Variables       map[string]interface{} `mapstructure:"variables"`
}
