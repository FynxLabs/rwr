package types

// PackageManagerInfo represents a package manager with its associated commands.
type PackageManagerInfo struct {
	Name     string `mapstructure:"name" yaml:"name" json:"name" toml:"name"`
	Bin      string `mapstructure:"bin" yaml:"bin" json:"bin" toml:"bin"`
	List     string `mapstructure:"list" yaml:"list" json:"list" toml:"list"`
	Search   string `mapstructure:"search" yaml:"search" json:"search" toml:"search"`
	Install  string `mapstructure:"install" yaml:"install" json:"install" toml:"install"`
	Update   string `mapstructure:"update" yaml:"update" json:"update" toml:"update"`
	Remove   string `mapstructure:"remove" yaml:"remove" json:"remove" toml:"remove"`
	Clean    string `mapstructure:"clean,omitempty" yaml:"clean,omitempty" json:"clean,omitempty" toml:"clean,omitempty"`
	Elevated bool   `mapstructure:"elevated" yaml:"elevated" json:"elevated" toml:"elevated"`
	Action   string `mapstructure:"action" yaml:"action" json:"action" toml:"action"`
	AsUser   string `mapstructure:"asUser,omitempty" yaml:"asUser,omitempty" json:"asUser,omitempty" toml:"asUser,omitempty"`
}

type PackageManager struct {
	Default PackageManagerInfo
	// Map of package manager name to info
	Managers map[string]PackageManagerInfo
}
