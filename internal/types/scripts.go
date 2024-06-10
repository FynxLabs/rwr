package types

type Script struct {
	Name     string `mapstructure:"name" yaml:"name" json:"name" toml:"name"`                                                         // Name of the script
	Action   string `mapstructure:"action" yaml:"action" json:"action" toml:"action"`                                                 // Action to perform with the script
	Source   string `mapstructure:"source,omitempty" yaml:"source,omitempty" json:"source,omitempty" toml:"source,omitempty"`         // Source of the script
	Content  string `mapstructure:"content,omitempty" yaml:"content,omitempty" json:"content,omitempty" toml:"content,omitempty"`     // Content of the file
	Args     string `mapstructure:"args,omitempty" yaml:"args,omitempty" json:"args,omitempty" toml:"args,omitempty"`                 // Arguments for the script
	Exec     string `mapstructure:"exec" yaml:"exec" json:"exec" toml:"exec"`                                                         // Executable for the script
	Elevated bool   `mapstructure:"elevated,omitempty" yaml:"elevated,omitempty" json:"elevated,omitempty" toml:"elevated,omitempty"` // Whether the script requires elevated privileges
	Log      string `mapstructure:"log,omitempty" yaml:"log,omitempty" json:"log,omitempty" toml:"log,omitempty"`                     // Log for the script
}

type ScriptData struct {
	Scripts []Script `mapstructure:"script,omitempty" yaml:"script,omitempty" json:"script,omitempty" toml:"script,omitempty"`
}
