package types

type Script struct {
	Name     string `mapstructure:"name"`               // Name of the script
	Action   string `mapstructure:"action"`             // Action to perform with the script
	Source   string `mapstructure:"source"`             // Source of the script
	Args     string `mapstructure:"args,omitempty"`     // Arguments for the script
	Exec     string `mapstructure:"exec"`               // Executable for the script
	Elevated bool   `mapstructure:"elevated,omitempty"` // Whether the script requires elevated privileges
	Log      string `mapstructure:"log,omitempty"`      // Log for the script
}
