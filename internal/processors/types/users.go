package types

type Group struct {
	Name   string `yaml:"name" json:"name" toml:"name"`
	Action string `yaml:"action" json:"action" toml:"action"`
}

type User struct {
	Name     string   `yaml:"name" json:"name" toml:"name"`
	Action   string   `yaml:"action" json:"action" toml:"action"`
	Password string   `yaml:"password" json:"password" toml:"password"`
	Groups   []string `yaml:"groups" json:"groups" toml:"groups"`
	Shell    string   `yaml:"shell" json:"shell" toml:"shell"`
	Home     string   `yaml:"home" json:"home" toml:"home"`
}
