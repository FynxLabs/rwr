package types

type Group struct {
	Name     string   `mapstructure:"name" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`                           // Name of the group
	Profiles []string `mapstructure:"profiles,omitempty" yaml:"profiles,omitempty" json:"profiles,omitempty" toml:"profiles,omitempty"` // Profiles this group belongs to
	NewName  string   `mapstructure:"new_name,omitempty" yaml:"new_name,omitempty" json:"new_name,omitempty" toml:"new_name,omitempty"` // New name for the group (for modify action)
	Action   string   `mapstructure:"action" yaml:"action,omitempty" json:"action,omitempty" toml:"action,omitempty"`                   // Action to perform with the group
}

type User struct {
	Name       string   `mapstructure:"name" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`                                       // Name of the user
	Profiles   []string `mapstructure:"profiles,omitempty" yaml:"profiles,omitempty" json:"profiles,omitempty" toml:"profiles,omitempty"`           // Profiles this user belongs to
	NewName    string   `mapstructure:"new_name,omitempty" yaml:"new_name,omitempty" json:"new_name,omitempty" toml:"new_name,omitempty"`             // New name for the user (for modify action)
	Action     string   `mapstructure:"action" yaml:"action,omitempty" json:"action,omitempty" toml:"action,omitempty"`                               // Action to perform with the user
	Password   string   `mapstructure:"password,omitempty" yaml:"password,omitempty" json:"password,omitempty" toml:"password,omitempty"`             // Password of the user
	Groups     []string `mapstructure:"groups,omitempty" yaml:"groups,omitempty" json:"groups,omitempty" toml:"groups,omitempty"`                     // Groups of the user
	AddGroups  []string `mapstructure:"add_groups,omitempty" yaml:"add_groups,omitempty" json:"add_groups,omitempty" toml:"add_groups,omitempty"`     // Groups to add the user to (for modify action)
	RemoveHome bool     `mapstructure:"remove_home,omitempty" yaml:"remove_home,omitempty" json:"remove_home,omitempty" toml:"remove_home,omitempty"` // Flag to remove the user's home directory (for remove action)
	Shell      string   `mapstructure:"shell,omitempty" yaml:"shell,omitempty" json:"shell,omitempty" toml:"shell,omitempty"`                         // Shell of the user
	NewShell   string   `mapstructure:"new_shell,omitempty" yaml:"new_shell,omitempty" json:"new_shell,omitempty" toml:"new_shell,omitempty"`         // New shell for the user (for modify action)
	Home       string   `mapstructure:"home,omitempty" yaml:"home,omitempty" json:"home,omitempty" toml:"home,omitempty"`                             // Home directory of the user
	NewHome    string   `mapstructure:"new_home,omitempty" yaml:"new_home,omitempty" json:"new_home,omitempty" toml:"new_home,omitempty"`             // New home directory for the user (for modify action)
}

type UsersData struct {
	Groups []Group `mapstructure:"groups,omitempty" yaml:"groups,omitempty" json:"groups,omitempty" toml:"groups,omitempty"` // Groups data
	Users  []User  `mapstructure:"users,omitempty" yaml:"users,omitempty" json:"users,omitempty" toml:"users,omitempty"`     // Users data
}

// GetProfiles returns the profiles for this group
func (g Group) GetProfiles() []string {
	return g.Profiles
}

// GetProfiles returns the profiles for this user
func (u User) GetProfiles() []string {
	return u.Profiles
}
