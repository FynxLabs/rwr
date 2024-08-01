package types

type Configuration struct {
	Name     string                 `mapstructure:"name" yaml:"name" json:"name" toml:"name"`
	Names    []string               `mapstructure:"names,omitempty" yaml:"names,omitempty" json:"names,omitempty" toml:"names,omitempty"`
	Action   string                 `mapstructure:"action" yaml:"action" json:"action" toml:"action"`
	Elevated bool                   `mapstructure:"elevated" yaml:"elevated" json:"elevated" toml:"elevated"`
	Tool     string                 `mapstructure:"tool" yaml:"tool" json:"tool" toml:"tool"`
	RunOnce  bool                   `mapstructure:"run_once" yaml:"run_once" json:"run_once" toml:"run_once"`
	Options  map[string]interface{} `mapstructure:"options" yaml:"options" json:"options" toml:"options"`
}

type DconfConfiguration struct {
	Configuration `mapstructure:",squash" yaml:",inline" json:",inline" toml:",inline"`
	File          string `mapstructure:"file" yaml:"file" json:"file" toml:"file"`
}

type GSettingsConfiguration struct {
	Configuration `mapstructure:",squash" yaml:",inline" json:",inline" toml:",inline"`
	Schema        string `mapstructure:"schema" yaml:"schema" json:"schema" toml:"schema"`
	Path          string `mapstructure:"path" yaml:"path" json:"path" toml:"path"`
	Key           string `mapstructure:"key" yaml:"key" json:"key" toml:"key"`
	Value         string `mapstructure:"value" yaml:"value" json:"value" toml:"value"`
}

type MacOSDefaultsConfiguration struct {
	Configuration `mapstructure:",squash" yaml:",inline" json:",inline" toml:",inline"`
	Domain        string      `mapstructure:"domain" yaml:"domain" json:"domain" toml:"domain"`
	Key           string      `mapstructure:"key" yaml:"key" json:"key" toml:"key"`
	Kind          string      `mapstructure:"kind" yaml:"kind" json:"kind" toml:"kind"`
	Value         interface{} `mapstructure:"value" yaml:"value" json:"value" toml:"value"`
}

type WindowsRegistryConfiguration struct {
	Configuration `mapstructure:",squash" yaml:",inline" json:",inline" toml:",inline"`
	Path          string      `mapstructure:"path" yaml:"path" json:"path" toml:"path"`
	Key           string      `mapstructure:"key" yaml:"key" json:"key" toml:"key"`
	Type          string      `mapstructure:"type" yaml:"type" json:"type" toml:"type"`
	Value         interface{} `mapstructure:"value" yaml:"value" json:"value" toml:"value"`
}

type ConfigData struct {
	Configurations []Configuration `mapstructure:"configurations" yaml:"configurations" json:"configurations" toml:"configurations"`
}
