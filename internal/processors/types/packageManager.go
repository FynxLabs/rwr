package types

// PackageManagerInfo represents a package manager with its associated commands.
type PackageManagerInfo struct {
	Name     string `mapstructure:"name,omitempty" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`
	Bin      string `mapstructure:"bin,omitempty" yaml:"bin,omitempty" json:"bin,omitempty" toml:"bin,omitempty"`
	List     string `mapstructure:"list,omitempty" yaml:"list,omitempty" json:"list,omitempty" toml:"list,omitempty"`
	Search   string `mapstructure:"search,omitempty" yaml:"search,omitempty" json:"search,omitempty" toml:"search,omitempty"`
	Install  string `mapstructure:"install,omitempty" yaml:"install,omitempty" json:"install,omitempty" toml:"install,omitempty"`
	Update   string `mapstructure:"update,omitempty" yaml:"update,omitempty" json:"update,omitempty" toml:"update,omitempty"`
	Remove   string `mapstructure:"remove,omitempty" yaml:"remove,omitempty" json:"remove,omitempty" toml:"remove,omitempty"`
	Clean    string `mapstructure:"clean,omitempty" yaml:"clean,omitempty" json:"clean,omitempty" toml:"clean,omitempty"`
	Elevated bool   `mapstructure:"elevated,omitempty" yaml:"elevated,omitempty" json:"elevated,omitempty" toml:"elevated,omitempty"`
	Action   string `mapstructure:"action,omitempty" yaml:"action,omitempty" json:"action,omitempty" toml:"action,omitempty"`
	AsUser   string `mapstructure:"asUser,omitempty" yaml:"asUser,omitempty" json:"asUser,omitempty" toml:"asUser,omitempty"`
}

type PackageManager struct {
	Default    PackageManagerInfo
	Apt        PackageManagerInfo
	Dnf        PackageManagerInfo
	Eopkg      PackageManagerInfo
	Pacman     PackageManagerInfo
	Yay        PackageManagerInfo
	Paru       PackageManagerInfo
	Trizen     PackageManagerInfo
	Yaourt     PackageManagerInfo
	Pamac      PackageManagerInfo
	Aura       PackageManagerInfo
	Zypper     PackageManagerInfo
	Emerge     PackageManagerInfo
	Brew       PackageManagerInfo
	Nix        PackageManagerInfo
	MAS        PackageManagerInfo
	Chocolatey PackageManagerInfo
	Scoop      PackageManagerInfo
	Npm        PackageManagerInfo
	Yarn       PackageManagerInfo
	Pnpm       PackageManagerInfo
	Pip        PackageManagerInfo
	Gem        PackageManagerInfo
	Cargo      PackageManagerInfo
	Snap       PackageManagerInfo
	Flatpak    PackageManagerInfo
	Apk        PackageManagerInfo
	Winget     PackageManagerInfo
	MacPorts   PackageManagerInfo
}
