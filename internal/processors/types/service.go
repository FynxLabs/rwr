package types

type Service struct {
	Name     string `yaml:"name" json:"name" toml:"name"`
	Action   string `yaml:"action" json:"action" toml:"action"`
	Elevated bool   `yaml:"elevated" json:"elevated" toml:"elevated"`
	Target   string `yaml:"target" json:"target" toml:"target"`
	Content  string `yaml:"content" json:"content" toml:"content"`
	Source   string `yaml:"source" json:"source" toml:"source"`
	File     string `yaml:"file" json:"file" toml:"file"`
}
