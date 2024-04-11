package types

type BootstrapData struct {
	Packages    []Package   `mapstructure:"packages,omitempty"`    // Packages data
	Files       []File      `mapstructure:"files,omitempty"`       // Files data
	Directories []Directory `mapstructure:"directories,omitempty"` // Directories data
	Git         []Git       `mapstructure:"git,omitempty"`         // Git data
	Services    []Service   `mapstructure:"services,omitempty"`    // Services data
	Groups      []Group     `mapstructure:"groups,omitempty"`      // Groups data
	Users       []User      `mapstructure:"users,omitempty"`       // Users data
}
