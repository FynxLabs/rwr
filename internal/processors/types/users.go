package types

type Group struct {
	Name   string `mapstructure:"name" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`         // Name of the group
	Action string `mapstructure:"action" yaml:"action,omitempty" json:"action,omitempty" toml:"action,omitempty"` // Action to perform with the group
}

type User struct {
	Name     string   `mapstructure:"name" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`                           // Name of the user
	Action   string   `mapstructure:"action" yaml:"action,omitempty" json:"action,omitempty" toml:"action,omitempty"`                   // Action to perform with the user
	Password string   `mapstructure:"password,omitempty" yaml:"password,omitempty" json:"password,omitempty" toml:"password,omitempty"` // Password of the user
	Groups   []string `mapstructure:"groups,omitempty" yaml:"groups,omitempty" json:"groups,omitempty" toml:"groups,omitempty"`         // Groups of the user
	Shell    string   `mapstructure:"shell,omitempty" yaml:"shell,omitempty" json:"shell,omitempty" toml:"shell,omitempty"`             // Shell of the user
	Home     string   `mapstructure:"home,omitempty" yaml:"home,omitempty" json:"home,omitempty" toml:"home,omitempty"`                 // Home directory of the user
}

type UsersData struct {
	Groups []Group `mapstructure:"groups,omitempty" yaml:"groups,omitempty" json:"groups,omitempty" toml:"groups,omitempty"` // Groups data
	Users  []User  `mapstructure:"users,omitempty" yaml:"users,omitempty" json:"users,omitempty" toml:"users,omitempty"`     // Users data
}
