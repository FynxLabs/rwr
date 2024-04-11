package types

type Repository struct {
	Name           string `mapstructure:"name"`                 // Name of the repository
	PackageManager string `mapstructure:"package_manager"`      // Package manager to use
	Action         string `mapstructure:"action"`               // Action to perform with the repository
	URL            string `mapstructure:"url"`                  // URL of the repository
	Arch           string `mapstructure:"arch,omitempty"`       // Architecture of the repository
	KeyURL         string `mapstructure:"key_url,omitempty"`    // Key URL of the repository
	Channel        string `mapstructure:"channel,omitempty"`    // Channel of the repository
	Component      string `mapstructure:"component,omitempty"`  // Component of the repository
	Repository     string `mapstructure:"repository,omitempty"` // Repository name
}
