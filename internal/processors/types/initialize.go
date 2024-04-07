package types

// OSInfo holds information about the detected OS, package managers, and tools.
type OSInfo struct {
	OS             string         // Operating system detected
	PackageManager PackageManager // Package managers available
	Tools          ToolList       // Common tools
}

// InitConfig represents the configuration for the initialization processor.
type InitConfig struct {
	Blueprints struct {
		Format   string   `mapstructure:"format"`
		Location string   `mapstructure:"location"`
		Order    []string `mapstructure:"order"`
	} `mapstructure:"blueprints"`
	PackageManagers []PackageManagerInfo   `mapstructure:"packageManagers"`
	Repositories    []Repository           `mapstructure:"repositories"`
	Packages        []Package              `mapstructure:"packages"`
	Services        []Service              `mapstructure:"services"`
	Files           []File                 `mapstructure:"files"`
	Directories     []Directory            `mapstructure:"directories"`
	Variables       map[string]interface{} `mapstructure:"variables"`
}
