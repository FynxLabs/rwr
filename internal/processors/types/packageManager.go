package types

// PackageManagerInfo represents a package manager with its associated commands.
type PackageManagerInfo struct {
	Name     string `mapstructure:"name,omitempty"`     // Name of the package manager
	Bin      string `mapstructure:"bin,omitempty"`      // Package manager binary
	List     string `mapstructure:"list,omitempty"`     // Command to list installed packages
	Search   string `mapstructure:"search,omitempty"`   // Command to search for a package
	Install  string `mapstructure:"install,omitempty"`  // Command to install a package
	Update   string `mapstructure:"update,omitempty"`   // Command to update package lists
	Remove   string `mapstructure:"remove,omitempty"`   // Command to remove a package
	Clean    string `mapstructure:"clean,omitempty"`    // Command to clean package cache
	Elevated bool   `mapstructure:"elevated,omitempty"` // Whether the package manager requires elevated privileges
	Action   string `mapstructure:"action,omitempty"`   // Action to perform with the package manager (install, remove)
	AsUser   string `mapstructure:"asUser,omitempty"`   // User to run the package manager as (macOS only)
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
}
