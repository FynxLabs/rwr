package types

type Group struct {
	Name   string `mapstructure:"name"`   // Name of the group
	Action string `mapstructure:"action"` // Action to perform with the group
}

type User struct {
	Name     string   `mapstructure:"name"`               // Name of the user
	Action   string   `mapstructure:"action"`             // Action to perform with the user
	Password string   `mapstructure:"password,omitempty"` // Password of the user
	Groups   []string `mapstructure:"groups,omitempty"`   // Groups of the user
	Shell    string   `mapstructure:"shell,omitempty"`    // Shell of the user
	Home     string   `mapstructure:"home,omitempty"`     // Home directory of the user
}

type UsersData struct {
	Groups []Group `mapstructure:"groups,omitempty"` // Groups data
	Users  []User  `mapstructure:"users,omitempty"`  // Users data
}
