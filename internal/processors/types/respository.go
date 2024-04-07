package types

type Repository struct {
	Name           string `yaml:"name"`
	PackageManager string `yaml:"package_manager"`
	Action         string `yaml:"action"`
	URL            string `yaml:"url"`
	Arch           string `yaml:"arch"`
	KeyURL         string `yaml:"key_url"`
	Channel        string `yaml:"channel"`
	Component      string `yaml:"component"`
	Repository     string `yaml:"repository"`
}
