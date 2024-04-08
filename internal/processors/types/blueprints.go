package types

type Blueprints struct {
	Format        string        `mapstructure:"format"`
	Location      string        `mapstructure:"location"`
	Order         []interface{} `mapstructure:"order"`
	RunOnlyListed bool          `mapstructure:"runOnlyListed"`
}

type BlueprintOrder struct {
	Source string   `mapstructure:"source"`
	Files  []string `mapstructure:"files"`
}
