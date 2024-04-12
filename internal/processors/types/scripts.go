package types

type Script struct {
	Name     string `mapstructure:"name" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`                           // Name of the script
	Action   string `mapstructure:"action" yaml:"action,omitempty" json:"action,omitempty" toml:"action,omitempty"`                   // Action to perform with the script
	Source   string `mapstructure:"source" yaml:"source,omitempty" json:"source,omitempty" toml:"source,omitempty"`                   // Source of the script
	Args     string `mapstructure:"args,omitempty" yaml:"args,omitempty" json:"args,omitempty" toml:"args,omitempty"`                 // Arguments for the script
	Exec     string `mapstructure:"exec" yaml:"exec,omitempty" json:"exec,omitempty" toml:"exec,omitempty"`                           // Executable for the script
	Elevated bool   `mapstructure:"elevated,omitempty" yaml:"elevated,omitempty" json:"elevated,omitempty" toml:"elevated,omitempty"` // Whether the script requires elevated privileges
	Log      string `mapstructure:"log,omitempty" yaml:"log,omitempty" json:"log,omitempty" toml:"log,omitempty"`                     // Log for the script
}
