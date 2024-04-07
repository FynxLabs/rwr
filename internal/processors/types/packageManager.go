package types

// PackageManagerInfo represents a package manager with its associated commands.
type PackageManagerInfo struct {
	Name     string // Package manager name
	Bin      string // Package manager binary
	List     string // Command to list installed packages
	Search   string // Command to search for a package
	Install  string // Command to install a package
	Update   string // Command to update package lists
	Remove   string // Command to remove a package
	Clean    string // Command to clean package cache
	Elevated bool   // Whether the package manager requires elevated privileges
	Action   string // Action to perform with the package manager (install, remove)
	AsUser   string // User to run the package manager as (macOS only)
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
