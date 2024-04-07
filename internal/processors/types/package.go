package types

type Package struct {
	Name           string   `mapstructure:"name"`
	Elevated       bool     `mapstructure:"elevated"`
	Action         string   `mapstructure:"action"`
	PackageManager string   `mapstructure:"package_manager"`
	Names          []string `mapstructure:"names"`
	Bootstrap      bool     `mapstructure:"bootstrap"`
}
