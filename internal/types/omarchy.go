package types

const BlueprintTypeOmarchy = "omarchy"

type OmarchyData struct {
	SchemaVersion `mapstructure:",squash" yaml:",inline" json:",inline" toml:",inline"`
	Entries       []OmarchySetup `mapstructure:"omarchy,omitempty" yaml:"omarchy,omitempty" json:"omarchy,omitempty" toml:"omarchy,omitempty"`
}

type OmarchySetup struct {
	Name         string               `mapstructure:"name,omitempty" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`
	Profiles     []string             `mapstructure:"profiles,omitempty" yaml:"profiles,omitempty" json:"profiles,omitempty" toml:"profiles,omitempty"`
	Import       string               `mapstructure:"import,omitempty" yaml:"import,omitempty" json:"import,omitempty" toml:"import,omitempty"`
	Plugins      []OmarchyPlugin      `mapstructure:"plugins,omitempty" yaml:"plugins,omitempty" json:"plugins,omitempty" toml:"plugins,omitempty"`
	Shell        *OmarchyShell        `mapstructure:"shell,omitempty" yaml:"shell,omitempty" json:"shell,omitempty" toml:"shell,omitempty"`
	Theme        *OmarchyTheme        `mapstructure:"theme,omitempty" yaml:"theme,omitempty" json:"theme,omitempty" toml:"theme,omitempty"`
	Defaults     *OmarchyDefaults     `mapstructure:"defaults,omitempty" yaml:"defaults,omitempty" json:"defaults,omitempty" toml:"defaults,omitempty"`
	Hooks        []OmarchyHook        `mapstructure:"hooks,omitempty" yaml:"hooks,omitempty" json:"hooks,omitempty" toml:"hooks,omitempty"`
	Integrations *OmarchyIntegrations `mapstructure:"integrations,omitempty" yaml:"integrations,omitempty" json:"integrations,omitempty" toml:"integrations,omitempty"`
}

type OmarchySource struct {
	Git   string `mapstructure:"git,omitempty" yaml:"git,omitempty" json:"git,omitempty" toml:"git,omitempty"`
	Path  string `mapstructure:"path,omitempty" yaml:"path,omitempty" json:"path,omitempty" toml:"path,omitempty"`
	Clone string `mapstructure:"clone,omitempty" yaml:"clone,omitempty" json:"clone,omitempty" toml:"clone,omitempty"`
	Ref   string `mapstructure:"ref,omitempty" yaml:"ref,omitempty" json:"ref,omitempty" toml:"ref,omitempty"`
}

type OmarchyPlugin struct {
	ID       string         `mapstructure:"id,omitempty" yaml:"id,omitempty" json:"id,omitempty" toml:"id,omitempty"`
	Source   *OmarchySource `mapstructure:"source,omitempty" yaml:"source,omitempty" json:"source,omitempty" toml:"source,omitempty"`
	State    string         `mapstructure:"state,omitempty" yaml:"state,omitempty" json:"state,omitempty" toml:"state,omitempty"`
	Enabled  *bool          `mapstructure:"enabled,omitempty" yaml:"enabled,omitempty" json:"enabled,omitempty" toml:"enabled,omitempty"`
	Settings map[string]any `mapstructure:"settings,omitempty" yaml:"settings,omitempty" json:"settings,omitempty" toml:"settings,omitempty"`
	Unset    []string       `mapstructure:"unset,omitempty" yaml:"unset,omitempty" json:"unset,omitempty" toml:"unset,omitempty"`
	Widget   *OmarchyWidget `mapstructure:"widget,omitempty" yaml:"widget,omitempty" json:"widget,omitempty" toml:"widget,omitempty"`
	Update   bool           `mapstructure:"update,omitempty" yaml:"update,omitempty" json:"update,omitempty" toml:"update,omitempty"`
}

type OmarchyWidget struct {
	Visible *bool  `mapstructure:"visible,omitempty" yaml:"visible,omitempty" json:"visible,omitempty" toml:"visible,omitempty"`
	Section string `mapstructure:"section,omitempty" yaml:"section,omitempty" json:"section,omitempty" toml:"section,omitempty"`
	Before  string `mapstructure:"before,omitempty" yaml:"before,omitempty" json:"before,omitempty" toml:"before,omitempty"`
	After   string `mapstructure:"after,omitempty" yaml:"after,omitempty" json:"after,omitempty" toml:"after,omitempty"`
	Index   *int   `mapstructure:"index,omitempty" yaml:"index,omitempty" json:"index,omitempty" toml:"index,omitempty"`
}

type OmarchyShell struct {
	Idle *OmarchyIdle `mapstructure:"idle,omitempty" yaml:"idle,omitempty" json:"idle,omitempty" toml:"idle,omitempty"`
	Bar  *OmarchyBar  `mapstructure:"bar,omitempty" yaml:"bar,omitempty" json:"bar,omitempty" toml:"bar,omitempty"`
}

type OmarchyIdle struct {
	Screensaver *int `mapstructure:"screensaver,omitempty" yaml:"screensaver,omitempty" json:"screensaver,omitempty" toml:"screensaver,omitempty"`
	Lock        *int `mapstructure:"lock,omitempty" yaml:"lock,omitempty" json:"lock,omitempty" toml:"lock,omitempty"`
}

type OmarchyBar struct {
	Position     *string `mapstructure:"position,omitempty" yaml:"position,omitempty" json:"position,omitempty" toml:"position,omitempty"`
	Transparent  *bool   `mapstructure:"transparent,omitempty" yaml:"transparent,omitempty" json:"transparent,omitempty" toml:"transparent,omitempty"`
	CenterAnchor *string `mapstructure:"centerAnchor,omitempty" yaml:"centerAnchor,omitempty" json:"centerAnchor,omitempty" toml:"centerAnchor,omitempty"`
}

type OmarchyTheme struct {
	Name       string         `mapstructure:"name,omitempty" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`
	Source     *OmarchySource `mapstructure:"source,omitempty" yaml:"source,omitempty" json:"source,omitempty" toml:"source,omitempty"`
	Active     *bool          `mapstructure:"active,omitempty" yaml:"active,omitempty" json:"active,omitempty" toml:"active,omitempty"`
	Background string         `mapstructure:"background,omitempty" yaml:"background,omitempty" json:"background,omitempty" toml:"background,omitempty"`
}

type OmarchyDefaults struct {
	Browser  string `mapstructure:"browser,omitempty" yaml:"browser,omitempty" json:"browser,omitempty" toml:"browser,omitempty"`
	Terminal string `mapstructure:"terminal,omitempty" yaml:"terminal,omitempty" json:"terminal,omitempty" toml:"terminal,omitempty"`
	Editor   string `mapstructure:"editor,omitempty" yaml:"editor,omitempty" json:"editor,omitempty" toml:"editor,omitempty"`
}

type OmarchyHook struct {
	Event  string `mapstructure:"event,omitempty" yaml:"event,omitempty" json:"event,omitempty" toml:"event,omitempty"`
	Name   string `mapstructure:"name,omitempty" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`
	Source string `mapstructure:"source,omitempty" yaml:"source,omitempty" json:"source,omitempty" toml:"source,omitempty"`
	State  string `mapstructure:"state,omitempty" yaml:"state,omitempty" json:"state,omitempty" toml:"state,omitempty"`
}

type OmarchyIntegrations struct {
	Screensaver *OmarchyScreensaver `mapstructure:"screensaver,omitempty" yaml:"screensaver,omitempty" json:"screensaver,omitempty" toml:"screensaver,omitempty"`
}

type OmarchyScreensaver struct {
	Mode        string       `mapstructure:"mode,omitempty" yaml:"mode,omitempty" json:"mode,omitempty" toml:"mode,omitempty"`
	Launch      *OmarchyExec `mapstructure:"launch,omitempty" yaml:"launch,omitempty" json:"launch,omitempty" toml:"launch,omitempty"`
	Check       *OmarchyExec `mapstructure:"check,omitempty" yaml:"check,omitempty" json:"check,omitempty" toml:"check,omitempty"`
	WindowClass string       `mapstructure:"windowClass,omitempty" yaml:"windowClass,omitempty" json:"windowClass,omitempty" toml:"windowClass,omitempty"`
}

type OmarchyExec struct {
	Exec string   `mapstructure:"exec,omitempty" yaml:"exec,omitempty" json:"exec,omitempty" toml:"exec,omitempty"`
	Args []string `mapstructure:"args,omitempty" yaml:"args,omitempty" json:"args,omitempty" toml:"args,omitempty"`
}

func (o OmarchySetup) GetProfiles() []string { return o.Profiles }
