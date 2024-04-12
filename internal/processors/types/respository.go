package types

type Repository struct {
	Name           string `mapstructure:"name" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`                                             // Name of the repository
	PackageManager string `mapstructure:"package_manager" yaml:"package_manager,omitempty" json:"package_manager,omitempty" toml:"package_manager,omitempty"` // Package manager to use
	Action         string `mapstructure:"action" yaml:"action,omitempty" json:"action,omitempty" toml:"action,omitempty"`                                     // Action to perform with the repository
	URL            string `mapstructure:"url" yaml:"url,omitempty" json:"url,omitempty" toml:"url,omitempty"`                                                 // URL of the repository
	Arch           string `mapstructure:"arch,omitempty" yaml:"arch,omitempty" json:"arch,omitempty" toml:"arch,omitempty"`                                   // Architecture of the repository
	KeyURL         string `mapstructure:"key_url,omitempty" yaml:"key_url,omitempty" json:"key_url,omitempty" toml:"key_url,omitempty"`                       // Key URL of the repository
	Channel        string `mapstructure:"channel,omitempty" yaml:"channel,omitempty" json:"channel,omitempty" toml:"channel,omitempty"`                       // Channel of the repository
	Component      string `mapstructure:"component,omitempty" yaml:"component,omitempty" json:"component,omitempty" toml:"component,omitempty"`               // Component of the repository
	Repository     string `mapstructure:"repository,omitempty" yaml:"repository,omitempty" json:"repository,omitempty" toml:"repository,omitempty"`           // Repository name
}
