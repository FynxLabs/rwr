package types

type Blueprints struct {
	Format   string   `mapstructure:"format"`
	Location string   `mapstructure:"location"`
	Order    []string `mapstructure:"order"`
}
