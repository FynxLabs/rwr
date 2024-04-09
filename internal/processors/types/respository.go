package types

type Repository struct {
	Name           string `yaml:"name" json:"name" toml:"name"`
	PackageManager string `yaml:"package_manager" json:"package_manager" toml:"package_manager"`
	Action         string `yaml:"action" json:"action" toml:"action"`
	URL            string `yaml:"url" json:"url" toml:"url"`
	Arch           string `yaml:"arch" json:"arch" toml:"arch"`
	KeyURL         string `yaml:"key_url" json:"key_url" toml:"key_url"`
	Channel        string `yaml:"channel" json:"channel" toml:"channel"`
	Component      string `yaml:"component" json:"component" toml:"component"`
	Repository     string `yaml:"repository" json:"repository" toml:"repository"`
}
