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
	// AsUser runs the script as another account (sudo -u). Elevated takes
	// precedence when both are set, since sudo cannot do both at once.
	AsUser      string `mapstructure:"asUser,omitempty" yaml:"asUser,omitempty" json:"asUser,omitempty" toml:"asUser,omitempty"`
	Log         string `mapstructure:"log,omitempty" yaml:"log,omitempty" json:"log,omitempty" toml:"log,omitempty"`
	Interactive *bool  `mapstructure:"interactive,omitempty" yaml:"interactive,omitempty" json:"interactive,omitempty" toml:"interactive,omitempty"`
	Import      string `mapstructure:"import,omitempty" yaml:"import,omitempty" json:"import,omitempty" toml:"import,omitempty"`
}

type ScriptData struct {
	// SchemaVersion, when set, overrides the tree-wide version from the init file.
	SchemaVersion `mapstructure:",squash" yaml:",inline" json:",inline" toml:",inline"`
	Scripts       []Script `mapstructure:"scripts,omitempty" yaml:"scripts,omitempty" json:"scripts,omitempty" toml:"scripts,omitempty"`
}

// GetProfiles returns the profiles for this script.
func (s Script) GetProfiles() []string {
	return s.Profiles
}
