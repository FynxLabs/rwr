package types

type Init struct {
	Format           string        `mapstructure:"format" yaml:"format,omitempty" json:"format,omitempty" toml:"format,omitempty"`
	Location         string        `mapstructure:"location" yaml:"location,omitempty" json:"location,omitempty" toml:"location,omitempty"`
	Order            []interface{} `mapstructure:"order,omitempty" yaml:"order,omitempty" json:"order,omitempty" toml:"order,omitempty"`
	Git              *GitOptions   `mapstructure:"git,omitempty" yaml:"git,omitempty" json:"git,omitempty" toml:"git,omitempty"`
	RunOnlyListed    bool          `mapstructure:"runOnlyListed,omitempty" yaml:"runOnlyListed,omitempty" json:"runOnlyListed,omitempty" toml:"runOnlyListed,omitempty"`
	TemplatesEnabled bool          `mapstructure:"templatesEnabled,omitempty" yaml:"templatesEnabled,omitempty" json:"templatesEnabled,omitempty" toml:"templatesEnabled,omitempty"`
}

type BlueprintOrder struct {
	Source string   `mapstructure:"source" yaml:"source,omitempty" json:"source,omitempty" toml:"source,omitempty"`
	Files  []string `mapstructure:"files" yaml:"files,omitempty" json:"files,omitempty" toml:"files,omitempty"`
}
