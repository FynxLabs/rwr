package types

// Blueprint type identifiers used in processor routing and validation.
const (
	BlueprintTypePackages        = "packages"
	BlueprintTypeRepositories    = "repositories"
	BlueprintTypeFiles           = "files"
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
	// ConfigurationActionSet is the only action the configuration tools implement.
	ConfigurationActionSet = "set"

	ServiceActionEnable  = "enable"
	ServiceActionDisable = "disable"
	ServiceActionStart   = "start"
	ServiceActionStop    = "stop"
	ServiceActionRestart = "restart"
)

// File actions for file management operations.
//
// This is the set the files processor actually dispatches on. It previously
// listed append and template, which no branch handles, and omitted copy, move,
// chmod, chown, chgrp and symlink, which every one of them does — so validation
// rejected the actions the examples use and accepted two that do nothing.
const (
	FileActionCreate  = "create"
	FileActionDelete  = "delete"
	FileActionCopy    = "copy"
	FileActionMove    = "move"
	FileActionChmod   = "chmod"
	FileActionChown   = "chown"
	FileActionChgrp   = "chgrp"
	FileActionSymlink = "symlink"
)

// FileActions is every action a file or template entry may declare.
var FileActions = []string{
	FileActionCreate,
	FileActionDelete,
	FileActionCopy,
	FileActionMove,
	FileActionChmod,
	FileActionChown,
	FileActionChgrp,
	FileActionSymlink,
}

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
