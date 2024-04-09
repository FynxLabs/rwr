package types

type Blueprints struct {
	Format           string        `yaml:"format" json:"format" toml:"format"`
	Location         string        `yaml:"location" json:"location" toml:"location"`
	Order            []interface{} `yaml:"order" json:"order" toml:"order"`
	Git              *GitOptions   `yaml:"git" json:"git" toml:"git"`
	RunOnlyListed    bool          `yaml:"runOnlyListed" json:"runOnlyListed" toml:"runOnlyListed"`
	TemplatesEnabled bool          `yaml:"templatesEnabled" json:"templatesEnabled" toml:"templatesEnabled"`
}

type BlueprintOrder struct {
	Source string   `yaml:"source" json:"source" toml:"source"`
	Files  []string `yaml:"files" json:"files" toml:"files"`
}
