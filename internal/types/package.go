package types

type Package struct {
	Name           string   `mapstructure:"name" yaml:"name" json:"name" toml:"name"`
	Elevated       bool     `mapstructure:"elevated,omitempty" yaml:"elevated,omitempty" json:"elevated,omitempty" toml:"elevated,omitempty"`
	Action         string   `mapstructure:"action" yaml:"action" json:"action" toml:"action"`
	PackageManager string   `mapstructure:"package_manager,omitempty" yaml:"package_manager,omitempty" json:"package_manager,omitempty" toml:"package_manager,omitempty"`
	Names          []string `mapstructure:"names,omitempty" yaml:"names,omitempty" json:"names,omitempty" toml:"names,omitempty"`
	Args           []string `mapstructure:"args,omitempty" yaml:"args,omitempty" json:"args,omitempty" toml:"args,omitempty"`
}

type PackagesData struct {
	Packages []Package `mapstructure:"packages,omitempty" yaml:"packages,omitempty" json:"packages,omitempty" toml:"packages,omitempty"`
}
