package types

type Package struct {
	Name           string   `toml:"name" yaml:"name" json:"name"`
	Elevated       bool     `toml:"elevated" yaml:"elevated" json:"elevated"`
	Action         string   `toml:"action" yaml:"action" json:"action"`
	PackageManager string   `toml:"package_manager" yaml:"package_manager" json:"package_manager"`
	Names          []string `toml:"names" yaml:"names" json:"names"`
	Bootstrap      bool     `toml:"bootstrap" yaml:"bootstrap" json:"bootstrap"`
}

type Config struct {
	Packages []Package `toml:"packages" yaml:"packages" json:"packages"`
}
