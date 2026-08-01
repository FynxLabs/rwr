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

type Variables struct {
	Flags       Flags
	User        UserInfo
	System      System
	UserDefined map[string]interface{}
}

// InitConfig represents the configuration for the initialization processor.
type InitConfig struct {
	Init            Init                 `mapstructure:"blueprints" yaml:"blueprints" json:"blueprints" toml:"blueprints"`
	PackageManagers []PackageManagerInfo `mapstructure:"packageManagers,omitempty" yaml:"packageManagers,omitempty" json:"packageManagers,omitempty" toml:"packageManagers,omitempty"`
	Repositories    []Repository         `mapstructure:"repositories,omitempty" yaml:"repositories,omitempty" json:"repositories,omitempty" toml:"repositories,omitempty"`
	Packages        []Package            `mapstructure:"packages,omitempty" yaml:"packages,omitempty" json:"packages,omitempty" toml:"packages,omitempty"`
	Services        []Service            `mapstructure:"services,omitempty" yaml:"services,omitempty" json:"services,omitempty" toml:"services,omitempty"`
	Files           []File               `mapstructure:"files,omitempty" yaml:"files,omitempty" json:"files,omitempty" toml:"files,omitempty"`
	Templates       []File               `mapstructure:"templates,omitempty" yaml:"templates,omitempty" json:"templates,omitempty" toml:"templates,omitempty"`
	Directories     []Directory          `mapstructure:"directories,omitempty" yaml:"directories,omitempty" json:"directories,omitempty" toml:"directories,omitempty"`
	Configuration   []Configuration      `mapstructure:"configuration,omitempty" yaml:"configuration,omitempty" json:"configuration,omitempty" toml:"configuration,omitempty"`
	Variables       Variables            `mapstructure:",squash"`
	// ExposeCredentials names the credentials this tree's blueprints are allowed
	// to read, e.g. ["gh_api_token"]. Empty — the default — means blueprints get
	// none of them. See docs/credentials.md.
	ExposeCredentials []string `mapstructure:"exposeCredentials,omitempty" yaml:"exposeCredentials,omitempty" json:"exposeCredentials,omitempty" toml:"exposeCredentials,omitempty"`
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
// genuinely needs one — writing a .netrc, configuring gh — opts in by name; see
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
