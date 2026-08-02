package types

type Font struct {
	Name     string   `mapstructure:"name" yaml:"name" json:"name" toml:"name"`
	Names    []string `mapstructure:"names,omitempty" yaml:"names,omitempty" json:"names,omitempty" toml:"names,omitempty"`
	Profiles []string `mapstructure:"profiles,omitempty" yaml:"profiles,omitempty" json:"profiles,omitempty" toml:"profiles,omitempty"`
	Action   string   `mapstructure:"action" yaml:"action" json:"action" toml:"action"`
	Provider string   `mapstructure:"provider,omitempty" yaml:"provider,omitempty" json:"provider,omitempty" toml:"provider,omitempty"`
	Location string   `mapstructure:"location,omitempty" yaml:"location,omitempty" json:"location,omitempty" toml:"location,omitempty"`
}

type FontsData struct {
	// SchemaVersion, when set, overrides the tree-wide version from the init file.
	SchemaVersion `mapstructure:",squash" yaml:",inline" json:",inline" toml:",inline"`
	Fonts         []Font `yaml:"fonts" json:"fonts" toml:"fonts"`
}

// GetProfiles returns the profiles for this font.
func (f Font) GetProfiles() []string {
	return f.Profiles
}
