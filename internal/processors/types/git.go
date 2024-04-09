package types

type GitOptions struct {
	URL     string `yaml:"url" json:"url" toml:"url"`
	Private bool   `yaml:"private" json:"private" toml:"private"`
	Target  string `yaml:"target" json:"target" toml:"target"`
	Update  bool   `yaml:"update" json:"update" toml:"update"`
	Branch  string `yaml:"branch" json:"branch" toml:"branch"`
}

type Git struct {
	Name    string `yaml:"name" json:"name" toml:"name"`
	Action  string `yaml:"action" json:"action" toml:"action"`
	Path    string `yaml:"path" json:"path" toml:"path"`
	URL     string `yaml:"url" json:"url" toml:"url"`
	Branch  string `yaml:"branch" json:"branch" toml:"branch"`
	Private bool   `yaml:"private" json:"private" toml:"private"`
}
