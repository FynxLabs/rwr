package types

import "fmt"

// OSInfo holds information about the detected OS, package managers, and tools.
type OSInfo struct {
	System         System         // System Info
	PackageManager PackageManager // Package managers available
	Tools          ToolList       // Common tools
}

type UserInfo struct {
	Username  string
	FirstName string
	LastName  string
	FullName  string
	GroupName string
	Home      string
	Shell     string
}

type Flags struct {
	Debug            bool
	LogLevel         string
	Interactive      bool
	ForceBootstrap   bool
	DryRun           bool
	GHAPIToken       string
	SSHKey           string
	SkipVersionCheck bool
	ConfigLocation   string
	RunOnceLocation  string
	Profiles         []string
}

type System struct {
	OS        string // Basic OS - Linux, macOS, Windows
	OSFamily  string // OS Family - Ubuntu, Fedora, Darwin
	OSVersion string // OS Version - 22.04
	OSArch    string // OS Arch - amd64, arm64
}

// Variables is the data every blueprint template renders against.
//
// Only UserDefined comes from the init file; Flags, User and System are filled in
// at runtime and are explicitly not decodable, so a blueprint cannot claim to be
// running as a different user or with different flags than it is.
type Variables struct {
	Flags       Flags                  `mapstructure:"-" yaml:"-" json:"-" toml:"-"`
	User        UserInfo               `mapstructure:"-" yaml:"-" json:"-" toml:"-"`
	System      System                 `mapstructure:"-" yaml:"-" json:"-" toml:"-"`
	UserDefined map[string]interface{} `mapstructure:"userDefined,omitempty" yaml:"userDefined,omitempty" json:"userDefined,omitempty" toml:"userDefined,omitempty"`
}

// InitConfig represents the configuration for the initialization processor.
type InitConfig struct {
	Init            Init                 `mapstructure:"blueprints" yaml:"blueprints" json:"blueprints" toml:"blueprints"`
	PackageManagers []PackageManagerInfo `mapstructure:"packageManagers,omitempty" yaml:"packageManagers,omitempty" json:"packageManagers,omitempty" toml:"packageManagers,omitempty"`
	// The inline resource sections (repositories, packages, services, files,
	// templates, directories, configuration) are gone: they were decoded,
	// validated, profile-counted - and never applied at runtime. Blueprints
	// are the single declaration path; under strict decode a leftover key is
	// an error naming it instead of a silent no-op.
	// Squashing this made the `variables:` block in the init file decode into
	// nothing at all, so every {{ .UserDefined.x }} in a blueprint rendered empty.
	Variables Variables `mapstructure:"variables,omitempty" yaml:"variables,omitempty" json:"variables,omitempty" toml:"variables,omitempty"`
	// ExposeCredentials names the credentials this tree's blueprints are allowed
	// to read, e.g. ["gh_api_token"]. Empty - the default - means blueprints get
	// none of them. See docs/credentials.md.
	ExposeCredentials []string `mapstructure:"exposeCredentials,omitempty" yaml:"exposeCredentials,omitempty" json:"exposeCredentials,omitempty" toml:"exposeCredentials,omitempty"`
	// Credentials declares the named credentials this tree's blueprints need and
	// where rwr should look for them. Declaring makes a credential managed
	// (resolved up front, redacted, withheld); exposing it to blueprints still
	// requires ExposeCredentials. See docs/credentials.md.
	Credentials []CredentialSpec `mapstructure:"credentials,omitempty" yaml:"credentials,omitempty" json:"credentials,omitempty" toml:"credentials,omitempty"`
}

func (u UserInfo) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"username":  u.Username,
		"firstName": u.FirstName,
		"lastName":  u.LastName,
		"fullName":  u.FullName,
		"groupName": u.GroupName,
		"home":      u.Home,
		"shell":     u.Shell,
	}
}

// ToMap exposes the flags to blueprint templates as {{ .Flags.* }}.
//
// ghAPIToken and sshKey are withheld unless the init file opts into them.
// Templates render from blueprint files cloned out of git repositories, and the
// result is written to a path the same blueprint chooses, so exposing a
// credential by default let any blueprint copy it anywhere. A blueprint that
// genuinely needs one - writing a .netrc, configuring gh - opts in by name; see
// exposeCredentials in the init file.
func (f Flags) ToMap() map[string]interface{} {
	out := map[string]interface{}{
		"debug":            f.Debug,
		"logLevel":         f.LogLevel,
		"interactive":      f.Interactive,
		"forceBootstrap":   f.ForceBootstrap,
		"dryRun":           f.DryRun,
		"skipVersionCheck": f.SkipVersionCheck,
		"configLocation":   f.ConfigLocation,
		"runOnceLocation":  f.RunOnceLocation,
		"profiles":         f.Profiles,
	}

	if IsCredentialExposed("gh_api_token") {
		out["ghAPIToken"] = f.GHAPIToken
	}
	if IsCredentialExposed("ssh_private_key") {
		out["sshKey"] = f.SSHKey
	}

	return out
}

// String redacts the credential fields so a Flags value cannot leak them through
// %v or %+v, which is how they were reaching debug logs.
func (f Flags) String() string {
	return fmt.Sprintf("Flags{debug:%v logLevel:%s interactive:%v forceBootstrap:%v "+
		"dryRun:%v ghAPIToken:%s sshKey:%s skipVersionCheck:%v configLocation:%s "+
		"runOnceLocation:%s profiles:%v}",
		f.Debug, f.LogLevel, f.Interactive, f.ForceBootstrap,
		f.DryRun, Redact(f.GHAPIToken), Redact(f.SSHKey), f.SkipVersionCheck,
		f.ConfigLocation, f.RunOnceLocation, f.Profiles)
}

func (f System) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"os":        f.OS,
		"osFamily":  f.OSFamily,
		"osVersion": f.OSVersion,
		"osArch":    f.OSArch,
	}
}
