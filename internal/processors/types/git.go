package types

type GitOptions struct {
	URL     string `yaml:"url" json:"url" toml:"url"`
	Private bool   `yaml:"private" json:"private" toml:"private"`
	Target  string `yaml:"target" json:"target" toml:"target"`
	Update  bool   `yaml:"update" json:"update" toml:"update"`
}
