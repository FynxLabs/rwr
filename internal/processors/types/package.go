package types

type Package struct {
	Name           string   `yaml:"name" json:"name" toml:"name"`
	Elevated       bool     `yaml:"elevated" json:"elevated" toml:"elevated"`
	Action         string   `yaml:"action" json:"action" toml:"action"`
	PackageManager string   `yaml:"package_manager" json:"package_manager" toml:"package_manager"`
	Names          []string `yaml:"names" json:"names" toml:"names"`
	Bootstrap      bool     `yaml:"bootstrap" json:"bootstrap" toml:"bootstrap"`
}
