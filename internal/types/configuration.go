package types

type Configuration struct {
	Name         string                 `mapstructure:"name,omitempty" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`
	Names        []string               `mapstructure:"names,omitempty" yaml:"names,omitempty" json:"names,omitempty" toml:"names,omitempty"`
	Profiles     []string               `mapstructure:"profiles,omitempty" yaml:"profiles,omitempty" json:"profiles,omitempty" toml:"profiles,omitempty"`
	Import       string                 `mapstructure:"import,omitempty" yaml:"import,omitempty" json:"import,omitempty" toml:"import,omitempty"`
	Action       string                 `mapstructure:"action" yaml:"action" json:"action" toml:"action"`
	Elevated     bool                   `mapstructure:"elevated,omitempty" yaml:"elevated,omitempty" json:"elevated,omitempty" toml:"elevated,omitempty"`
	Tool         string                 `mapstructure:"tool" yaml:"tool" json:"tool" toml:"tool"`
	RunOnce      bool                   `mapstructure:"run_once,omitempty" yaml:"run_once,omitempty" json:"run_once,omitempty" toml:"run_once,omitempty"`
	File         string                 `mapstructure:"file,omitempty" yaml:"file,omitempty" json:"file,omitempty" toml:"file,omitempty"`
	Schema       string                 `mapstructure:"schema,omitempty" yaml:"schema,omitempty" json:"schema,omitempty" toml:"schema,omitempty"`
	Path         string                 `mapstructure:"path,omitempty" yaml:"path,omitempty" json:"path,omitempty" toml:"path,omitempty"`
	Key          string                 `mapstructure:"key,omitempty" yaml:"key,omitempty" json:"key,omitempty" toml:"key,omitempty"`
	Value        interface{}            `mapstructure:"value,omitempty" yaml:"value,omitempty" json:"value,omitempty" toml:"value,omitempty"`
	Domain       string                 `mapstructure:"domain,omitempty" yaml:"domain,omitempty" json:"domain,omitempty" toml:"domain,omitempty"`
	Kind         string                 `mapstructure:"kind,omitempty" yaml:"kind,omitempty" json:"kind,omitempty" toml:"kind,omitempty"`
	Type         string                 `mapstructure:"type,omitempty" yaml:"type,omitempty" json:"type,omitempty" toml:"type,omitempty"`
	Settings     map[string]interface{} `mapstructure:"settings,omitempty" yaml:"settings,omitempty" json:"settings,omitempty" toml:"settings,omitempty"`
	Plugins      []OmarchyPlugin        `mapstructure:"plugins,omitempty" yaml:"plugins,omitempty" json:"plugins,omitempty" toml:"plugins,omitempty"`
	Shell        *OmarchyShell          `mapstructure:"shell,omitempty" yaml:"shell,omitempty" json:"shell,omitempty" toml:"shell,omitempty"`
	Theme        *OmarchyTheme          `mapstructure:"theme,omitempty" yaml:"theme,omitempty" json:"theme,omitempty" toml:"theme,omitempty"`
	Defaults     *OmarchyDefaults       `mapstructure:"defaults,omitempty" yaml:"defaults,omitempty" json:"defaults,omitempty" toml:"defaults,omitempty"`
	Hooks        []OmarchyHook          `mapstructure:"hooks,omitempty" yaml:"hooks,omitempty" json:"hooks,omitempty" toml:"hooks,omitempty"`
	Integrations *OmarchyIntegrations   `mapstructure:"integrations,omitempty" yaml:"integrations,omitempty" json:"integrations,omitempty" toml:"integrations,omitempty"`
}

type ConfigData struct {
	// SchemaVersion, when set, overrides the tree-wide version from the init file.
	SchemaVersion  `mapstructure:",squash" yaml:",inline" json:",inline" toml:",inline"`
	Configurations []Configuration `mapstructure:"configurations" yaml:"configurations" json:"configurations" toml:"configurations"`
}

// GetProfiles returns the profiles for this configuration entry.
func (c Configuration) GetProfiles() []string {
	return c.Profiles
}

// AsOmarchySetup returns the Omarchy-specific part of a configuration entry.
// Omarchy is a configuration tool, so its wire format remains under the
// configurations list even though its reconciler uses a focused typed model.
func (c Configuration) AsOmarchySetup() OmarchySetup {
	return OmarchySetup{
		Name:         c.Name,
		Plugins:      c.Plugins,
		Shell:        c.Shell,
		Theme:        c.Theme,
		Defaults:     c.Defaults,
		Hooks:        c.Hooks,
		Integrations: c.Integrations,
	}
}
