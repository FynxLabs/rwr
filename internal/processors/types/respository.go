package types

type Repository struct {
	Name           string `mapstructure:"name"`
	PackageManager string `mapstructure:"package_manager"`
	Action         string `mapstructure:"action"`
	URL            string `mapstructure:"url"`
	Arch           string `mapstructure:"arch"`
	KeyURL         string `mapstructure:"key_url"`
	Channel        string `mapstructure:"channel"`
	Component      string `mapstructure:"component"`
	Repository     string `mapstructure:"repository"`
}
