package types

type BootstrapData struct {
	Packages    []Package   `mapstructure:"packages,omitempty" yaml:"packages,omitempty" json:"packages,omitempty" toml:"packages,omitempty"`             // Packages data
	Files       []File      `mapstructure:"files,omitempty" yaml:"files,omitempty" json:"files,omitempty" toml:"files,omitempty"`                         // Files data
	Directories []Directory `mapstructure:"directories,omitempty" yaml:"directories,omitempty" json:"directories,omitempty" toml:"directories,omitempty"` // Directories data
	Git         []Git       `mapstructure:"git,omitempty" yaml:"git,omitempty" json:"git,omitempty" toml:"git,omitempty"`                                 // Git data
	SSHKeys     []SSHKey    `mapstructure:"ssh_keys,omitempty" yaml:"ssh_keys,omitempty" json:"ssh_keys,omitempty" toml:"ssh_keys,omitempty"`             // SSHKey Data
	Services    []Service   `mapstructure:"services,omitempty" yaml:"services,omitempty" json:"services,omitempty" toml:"services,omitempty"`             // Services data
	Groups      []Group     `mapstructure:"groups,omitempty" yaml:"groups,omitempty" json:"groups,omitempty" toml:"groups,omitempty"`                     // Groups data
	Users       []User      `mapstructure:"users,omitempty" yaml:"users,omitempty" json:"users,omitempty" toml:"users,omitempty"`                         // Users data
}
