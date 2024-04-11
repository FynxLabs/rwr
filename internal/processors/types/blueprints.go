package types

type Init struct {
	Format           string        `mapstructure:"format"`
	Location         string        `mapstructure:"location"`
	Order            []interface{} `mapstructure:"order,omitempty"`
	Git              *GitOptions   `mapstructure:"git,omitempty"`
	RunOnlyListed    bool          `mapstructure:"runOnlyListed,omitempty"`
	TemplatesEnabled bool          `mapstructure:"templatesEnabled,omitempty"`
}

type BlueprintOrder struct {
	Source string   `mapstructure:"source"`
	Files  []string `mapstructure:"files"`
}
