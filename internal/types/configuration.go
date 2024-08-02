package types

type Configuration struct {
	Name     string                 `mapstructure:"name,omitempty" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`
	Names    []string               `mapstructure:"names,omitempty" yaml:"names,omitempty" json:"names,omitempty" toml:"names,omitempty"`
	Action   string                 `mapstructure:"action" yaml:"action" json:"action" toml:"action"`
	Elevated bool                   `mapstructure:"elevated,omitempty" yaml:"elevated,omitempty" json:"elevated,omitempty" toml:"elevated,omitempty"`
	Tool     string                 `mapstructure:"tool" yaml:"tool" json:"tool" toml:"tool"`
	RunOnce  bool                   `mapstructure:"run_once,omitempty" yaml:"run_once,omitempty" json:"run_once,omitempty" toml:"run_once,omitempty"`
	File     string                 `mapstructure:"file,omitempty" yaml:"file,omitempty" json:"file,omitempty" toml:"file,omitempty"`
	Schema   string                 `mapstructure:"schema,omitempty" yaml:"schema,omitempty" json:"schema,omitempty" toml:"schema,omitempty"`
	Path     string                 `mapstructure:"path,omitempty" yaml:"path,omitempty" json:"path,omitempty" toml:"path,omitempty"`
	Key      string                 `mapstructure:"key,omitempty" yaml:"key,omitempty" json:"key,omitempty" toml:"key,omitempty"`
	Value    interface{}            `mapstructure:"value,omitempty" yaml:"value,omitempty" json:"value,omitempty" toml:"value,omitempty"`
	Domain   string                 `mapstructure:"domain,omitempty" yaml:"domain,omitempty" json:"domain,omitempty" toml:"domain,omitempty"`
	Kind     string                 `mapstructure:"kind,omitempty" yaml:"kind,omitempty" json:"kind,omitempty" toml:"kind,omitempty"`
	Type     string                 `mapstructure:"type,omitempty" yaml:"type,omitempty" json:"type,omitempty" toml:"type,omitempty"`
	Settings map[string]interface{} `mapstructure:"settings,omitempty" yaml:"settings,omitempty" json:"settings,omitempty" toml:"settings,omitempty"`
}

type ConfigData struct {
	Configurations []Configuration `mapstructure:"configurations" yaml:"configurations" json:"configurations" toml:"configurations"`
}
