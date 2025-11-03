package types

type Service struct {
	Name     string   `mapstructure:"name" yaml:"name" json:"name" toml:"name"`                                                         // Name of the service
	Profiles []string `mapstructure:"profiles,omitempty" yaml:"profiles,omitempty" json:"profiles,omitempty" toml:"profiles,omitempty"` // Profiles this service belongs to
	Action   string   `mapstructure:"action" yaml:"action" json:"action" toml:"action"`                                                 // Action to perform with the service
	Elevated bool     `mapstructure:"elevated,omitempty" yaml:"elevated,omitempty" json:"elevated,omitempty" toml:"elevated,omitempty"` // Whether the service requires elevated privileges
	Target   string   `mapstructure:"target,omitempty" yaml:"target,omitempty" json:"target,omitempty" toml:"target,omitempty"`         // Target of the service
	Content  string   `mapstructure:"content,omitempty" yaml:"content,omitempty" json:"content,omitempty" toml:"content,omitempty"`     // Content of the service
	Source   string   `mapstructure:"source,omitempty" yaml:"source,omitempty" json:"source,omitempty" toml:"source,omitempty"`         // Source of the service
	File     string   `mapstructure:"file,omitempty" yaml:"file,omitempty" json:"file,omitempty" toml:"file,omitempty"`                 // File of the service
	Import   string   `mapstructure:"import,omitempty" yaml:"import,omitempty" json:"import,omitempty" toml:"import,omitempty"`         // Import path for external service definitions
}

type ServiceData struct {
	Services []Service `mapstructure:"services" yaml:"services" json:"services" toml:"services"`
}

// GetProfiles returns the profiles for this service
func (s Service) GetProfiles() []string {
	return s.Profiles
}
