package types

type Command struct {
	Exec        string            `mapstructure:"exec" yaml:"exec" json:"exec" toml:"exec"`
	Variables   map[string]string `mapstructure:"variables,omitempty" yaml:"variables,omitempty" json:"variables,omitempty" toml:"variables,omitempty"`
	Args        []string          `mapstructure:"args,omitempty" yaml:"args,omitempty" json:"args,omitempty" toml:"args,omitempty"`
	LogName     string            `mapstructure:"logName,omitempty" yaml:"logName,omitempty" json:"logName,omitempty" toml:"logName,omitempty"`
	AsUser      string            `mapstructure:"asUser,omitempty" yaml:"asUser,omitempty" json:"asUser,omitempty" toml:"asUser,omitempty"`
	Interactive bool              `mapstructure:"interactive" yaml:"interactive" json:"interactive" toml:"interactive"`
	Elevated    bool              `mapstructure:"elevated" yaml:"elevated" json:"elevated" toml:"elevated"`
}
