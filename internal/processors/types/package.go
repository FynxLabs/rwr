package types

type Package struct {
	Name           string   `mapstructure:"name"`                      // Name of the package
	Elevated       bool     `mapstructure:"elevated,omitempty"`        // Whether the package requires elevated privileges
	Action         string   `mapstructure:"action"`                    // Action to perform with the package
	PackageManager string   `mapstructure:"package_manager,omitempty"` // Package manager to use
	Names          []string `mapstructure:"names,omitempty"`           // Names of the packages
	Bootstrap      bool     `mapstructure:"bootstrap,omitempty"`       // Whether to bootstrap the package
}
