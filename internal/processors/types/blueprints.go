package types

type Blueprints struct {
	Format           string        `yaml:"format" json:"format" toml:"format"`
	Location         string        `yaml:"location" json:"location" toml:"location"`
	Order            []interface{} `mapstructure:"order"`
	Git              *GitOptions   `mapstructure:"git"`
	RunOnlyListed    bool          `yaml:"runOnlyListed" json:"runOnlyListed" toml:"runOnlyListed"`
	TemplatesEnabled bool          `yaml:"templatesEnabled" json:"templatesEnabled" toml:"templatesEnabled"`
}

type BlueprintOrder struct {
	Source string   `yaml:"source" json:"source" toml:"source"`
	Files  []string `mapstructure:"files"`
}
