package types

type Package struct {
	Name           string   `mapstructure:"name" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`                                                       // Name of the package
	Elevated       bool     `mapstructure:"elevated,omitempty" yaml:"elevated,omitempty" json:"elevated,omitempty" toml:"elevated,omitempty"`                             // Whether the package requires elevated privileges
	Action         string   `mapstructure:"action" yaml:"action,omitempty" json:"action,omitempty" toml:"action,omitempty"`                                               // Action to perform with the package
	PackageManager string   `mapstructure:"package_manager,omitempty" yaml:"package_manager,omitempty" json:"package_manager,omitempty" toml:"package_manager,omitempty"` // Package manager to use
	Names          []string `mapstructure:"names,omitempty" yaml:"names,omitempty" json:"names,omitempty" toml:"names,omitempty"`                                         // Names of the packages
}

type PackagesData struct {
	Packages []Package `mapstructure:"packages,omitempty" yaml:"packages,omitempty" json:"packages,omitempty" toml:"packages,omitempty"` // Packages data
}
