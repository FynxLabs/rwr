package types

type Blueprints struct {
	Format           string        `mapstructure:"format"`
	Location         string        `mapstructure:"location"`
	Order            []interface{} `mapstructure:"order"`
	RunOnlyListed    bool          `mapstructure:"runOnlyListed"`
	TemplatesEnabled bool          `mapstructure:"templatesEnabled"`
}

type BlueprintOrder struct {
	Source string   `mapstructure:"source"`
	Files  []string `mapstructure:"files"`
}
