package types

// Blueprint type identifiers used in processor routing and validation.
const (
	BlueprintTypePackages        = "packages"
	BlueprintTypeRepositories    = "repositories"
	BlueprintTypeFiles           = "files"
	BlueprintTypeDirectories     = "directories"
	BlueprintTypeGit             = "git"
	BlueprintTypeScripts         = "scripts"
	BlueprintTypeSSHKeys         = "ssh_keys"
	BlueprintTypeFonts           = "fonts"
	BlueprintTypeUsers           = "users"
	BlueprintTypePackageManagers = "packageManagers"
	BlueprintTypeConfiguration   = "configuration"
	BlueprintTypeBootstrap       = "bootstrap"
	BlueprintTypeServices        = "services"
)

// Supported file formats for blueprint parsing.
const (
	FormatYAML    = "yaml"
	FormatYAMLAlt = "yml"
	FormatJSON    = "json"
	FormatTOML    = "toml"

	FormatExtYAML    = ".yaml"
	FormatExtYAMLAlt = ".yml"
	FormatExtJSON    = ".json"
	FormatExtTOML    = ".toml"
)

// OS identifiers matching runtime.GOOS values.
const (
	OSLinux   = "linux"
	OSDarwin  = "darwin"
	OSWindows = "windows"
)

// Package actions for package management operations.
const (
	ActionInstall = "install"
	ActionRemove  = "remove"
	ActionUpdate  = "update"
)

// Service actions for service management operations.
const (
	ServiceActionEnable  = "enable"
	ServiceActionDisable = "disable"
	ServiceActionStart   = "start"
	ServiceActionStop    = "stop"
	ServiceActionRestart = "restart"
)

// File actions for file management operations.
const (
	FileActionCreate   = "create"
	FileActionDelete   = "delete"
	FileActionAppend   = "append"
	FileActionTemplate = "template"
)

// Repository actions for repository management operations.
const (
	RepoActionAdd    = "add"
	RepoActionRemove = "remove"
)

// User actions for user management operations.
const (
	UserActionCreate = "create"
	UserActionModify = "modify"
	UserActionDelete = "delete"
)
