package types

type Repository struct {
	Name           string   `mapstructure:"name" yaml:"name" json:"name" toml:"name"`                                                                 // Name of the repository
	Profiles       []string `mapstructure:"profiles,omitempty" yaml:"profiles,omitempty" json:"profiles,omitempty" toml:"profiles,omitempty"`         // Profiles this repository belongs to
	PackageManager string   `mapstructure:"package_manager" yaml:"package_manager" json:"package_manager" toml:"package_manager"`                     // Package manager to use
	Action         string   `mapstructure:"action" yaml:"action" json:"action" toml:"action"`                                                         // Action to perform with the repository
	URL            string   `mapstructure:"url" yaml:"url" json:"url" toml:"url"`                                                                     // URL of the repository
	Arch           string   `mapstructure:"arch,omitempty" yaml:"arch,omitempty" json:"arch,omitempty" toml:"arch,omitempty"`                         // Architecture of the repository
	KeyURL         string   `mapstructure:"key_url,omitempty" yaml:"key_url,omitempty" json:"key_url,omitempty" toml:"key_url,omitempty"`             // Key URL of the repository
	Channel        string   `mapstructure:"channel,omitempty" yaml:"channel,omitempty" json:"channel,omitempty" toml:"channel,omitempty"`             // Channel of the repository
	Component      string   `mapstructure:"component,omitempty" yaml:"component,omitempty" json:"component,omitempty" toml:"component,omitempty"`     // Component of the repository
	Repository     string   `mapstructure:"repository,omitempty" yaml:"repository,omitempty" json:"repository,omitempty" toml:"repository,omitempty"` // Repository name
	Import         string   `mapstructure:"import,omitempty" yaml:"import,omitempty" json:"import,omitempty" toml:"import,omitempty"`                 // Import path for external repository definitions
}

type RepositoriesData struct {
	Repositories []Repository `mapstructure:"repositories" yaml:"repositories" json:"repositories" toml:"repositories"`
}

// GetProfiles returns the profiles for this repository
func (r Repository) GetProfiles() []string {
	return r.Profiles
}
