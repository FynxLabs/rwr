package types

type Service struct {
	Name     string `mapstructure:"name"`               // Name of the service
	Action   string `mapstructure:"action"`             // Action to perform with the service
	Elevated bool   `mapstructure:"elevated,omitempty"` // Whether the service requires elevated privileges
	Target   string `mapstructure:"target,omitempty"`   // Target of the service
	Content  string `mapstructure:"content,omitempty"`  // Content of the service
	Source   string `mapstructure:"source,omitempty"`   // Source of the service
	File     string `mapstructure:"file,omitempty"`     // File of the service
}
