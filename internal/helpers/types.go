package helpers

// PackageManagerInfo represents a package manager with its associated commands.
type PackageManagerInfo struct {
	Bin      string // Package manager binary
	List     string // Command to list installed packages
	Search   string // Command to search for a package
	Install  string // Command to install a package
	Remove   string // Command to remove a package
	Clean    string // Command to clean package cache
	Elevated bool   // Whether the package manager requires elevated privileges
}

// ToolInfo represents information about a tool.
type ToolInfo struct {
	Exists bool   // Whether the tool exists
	Bin    string // Path to the tool binary
}

type ToolList struct {
	Git    ToolInfo
	Pip    ToolInfo
	Gem    ToolInfo
	Npm    ToolInfo
	Yarn   ToolInfo
	Pnpm   ToolInfo
	Bun    ToolInfo
	Cargo  ToolInfo
	Docker ToolInfo
	Curl   ToolInfo
	Wget   ToolInfo
	Make   ToolInfo
	GCC    ToolInfo
	Clang  ToolInfo
	Python ToolInfo
	Ruby   ToolInfo
	Java   ToolInfo
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
}

// OSInfo holds information about the detected OS, package managers, and tools.
type OSInfo struct {
	OS             string         // Operating system detected
	PackageManager PackageManager // Package managers available
	Tools          ToolList       // Common tools
}
