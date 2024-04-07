package types

type File struct {
	Name    string   `yaml:"name"`
	Names   []string `yaml:"names"`
	Action  string   `yaml:"action"`
	Content string   `yaml:"content"`
	Source  string   `yaml:"source"`
	Target  string   `yaml:"target"`
	Owner   int      `yaml:"owner"`
	Group   int      `yaml:"group"`
	Mode    int      `yaml:"mode"`
	Create  bool     `yaml:"create"`
}

type Directory struct {
	Name   string   `yaml:"name"`
	Names  []string `yaml:"names"`
	Action string   `yaml:"action"`
	Source string   `yaml:"source"`
	Target string   `yaml:"target"`
	Owner  int      `yaml:"owner"`
	Group  int      `yaml:"group"`
	Mode   int      `yaml:"mode"`
	Create bool     `yaml:"create"`
}
