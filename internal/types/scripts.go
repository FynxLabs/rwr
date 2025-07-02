package types

type Script struct {
	Name     string   `mapstructure:"name" yaml:"name" json:"name" toml:"name"`
	Profiles []string `mapstructure:"profiles,omitempty" yaml:"profiles,omitempty" json:"profiles,omitempty" toml:"profiles,omitempty"`
	Action   string   `mapstructure:"action" yaml:"action" json:"action" toml:"action"`
	Exec     string   `mapstructure:"exec" yaml:"exec" json:"exec" toml:"exec"`
	Source   string   `mapstructure:"source,omitempty" yaml:"source,omitempty" json:"source,omitempty" toml:"source,omitempty"`
	Content  string   `mapstructure:"content,omitempty" yaml:"content,omitempty" json:"content,omitempty" toml:"content,omitempty"`
	Args     string   `mapstructure:"args,omitempty" yaml:"args,omitempty" json:"args,omitempty" toml:"args,omitempty"`
	Elevated bool     `mapstructure:"elevated,omitempty" yaml:"elevated,omitempty" json:"elevated,omitempty" toml:"elevated,omitempty"`
	Log      string   `mapstructure:"log,omitempty" yaml:"log,omitempty" json:"log,omitempty" toml:"log,omitempty"`
}

type ScriptData struct {
	Scripts []Script `mapstructure:"scripts,omitempty" yaml:"scripts,omitempty" json:"scripts,omitempty" toml:"scripts,omitempty"`
}

// GetProfiles returns the profiles for this script
func (s Script) GetProfiles() []string {
	return s.Profiles
}
