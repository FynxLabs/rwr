package types

type Service struct {
	Name     string `yaml:"name"`
	Action   string `yaml:"action"`
	Elevated bool   `yaml:"elevated"`
	Target   string `yaml:"target"`
	Content  string `yaml:"content"`
	Source   string `yaml:"source"`
	File     string `yaml:"file"`
}
