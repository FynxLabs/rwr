package types

type Template struct {
	Name      string                 `yaml:"name" json:"name" toml:"name"`
	Names     []string               `yaml:"names" json:"names" toml:"names"`
	Action    string                 `yaml:"action" json:"action" toml:"action"`
	Content   string                 `yaml:"content" json:"content" toml:"content"`
	Source    string                 `yaml:"source" json:"source" toml:"source"`
	Target    string                 `yaml:"target" json:"target" toml:"target"`
	Owner     int                    `yaml:"owner" json:"owner" toml:"owner"`
	Group     int                    `yaml:"group" json:"group" toml:"group"`
	Mode      int                    `yaml:"mode" json:"mode" toml:"mode"`
	Create    bool                   `yaml:"create" json:"create" toml:"create"`
	Variables map[string]interface{} `yaml:"variables" json:"variables" toml:"variables"`
}
