package cmd

import (
	"github.com/fynxlabs/rwr/internal/types"
)

// AppConfig owns what used to be seventeen package-level variables in this
// package: authentication, execution flags, paths, and the run's resolved
// InitConfig/OSInfo. One instance is built per command tree (NewRootCmd), so
// tests can construct isolated instances with no shared mutable state and no
// reset dance between cases.
type AppConfig struct {
	// Authentication
	GHAPIToken string // GitHub API token for repository operations
	GHAuth     bool   // Use OAuth device flow for GitHub authentication
	SSHKey     string // SSH private key for Git auth (path or base64)

	// Execution flags
	SkipVersionCheck bool
	ShowSecrets      bool
	Debug            bool
	Interactive      bool
	ForceBootstrap   bool
	DryRun           bool
	LogLevel         string
	Profiles         []string

	// Paths
	ConfigPath      string // --config: overrides where the config file is looked up
	ConfigLocation  string
	RunOnceLocation string
	InitFilePath    string

	// Display
	NoTUI     bool
	Theme     string
	ASCII     bool
	Unicode   bool
	NoNotify  bool
	TUIBuffer int
	LogFile   string

	// Resolved run state
	InitConfig *types.InitConfig
	OSInfo     *types.OSInfo
}

// NewAppConfig returns an AppConfig carrying the defaults the flags declare,
// so a struct used without flag parsing (tests) behaves like a bare `rwr`.
func NewAppConfig() *AppConfig {
	return &AppConfig{
		Interactive: true,
		LogLevel:    "info",
	}
}
