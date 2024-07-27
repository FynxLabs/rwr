package types

type BaseFile struct {
	Name     string   `mapstructure:"name" yaml:"name" json:"name" toml:"name"`
	Names    []string `mapstructure:"names,omitempty" yaml:"names,omitempty" json:"names,omitempty" toml:"names,omitempty"`
	Action   string   `mapstructure:"action" yaml:"action" json:"action" toml:"action"`
	Content  string   `mapstructure:"content,omitempty" yaml:"content,omitempty" json:"content,omitempty" toml:"content,omitempty"`
	Source   string   `mapstructure:"source,omitempty" yaml:"source,omitempty" json:"source,omitempty" toml:"source,omitempty"`
	Target   string   `mapstructure:"target" yaml:"target" json:"target" toml:"target"`
	Owner    string   `mapstructure:"owner,omitempty" yaml:"owner,omitempty" json:"owner,omitempty" toml:"owner,omitempty"`
	Group    string   `mapstructure:"group,omitempty" yaml:"group,omitempty" json:"group,omitempty" toml:"group,omitempty"`
	Mode     int      `mapstructure:"mode,omitempty" yaml:"mode,omitempty" json:"mode,omitempty" toml:"mode,omitempty"`
	Elevated bool     `mapstructure:"elevated" yaml:"elevated" json:"elevated" toml:"elevated"`
}

type File BaseFile

type Template struct {
	BaseFile
	Variables map[string]interface{} `mapstructure:"variables,omitempty" yaml:"variables,omitempty" json:"variables,omitempty" toml:"variables,omitempty"`
}

type Directory struct {
	Name     string   `mapstructure:"name,omitempty" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`
	Names    []string `mapstructure:"names,omitempty" yaml:"names,omitempty" json:"names,omitempty" toml:"names,omitempty"`
	Action   string   `mapstructure:"action" yaml:"action" json:"action" toml:"action"`
	Source   string   `mapstructure:"source,omitempty" yaml:"source,omitempty" json:"source,omitempty" toml:"source,omitempty"`
	Target   string   `mapstructure:"target" yaml:"target" json:"target" toml:"target"`
	Owner    string   `mapstructure:"owner,omitempty" yaml:"owner,omitempty" json:"owner,omitempty" toml:"owner,omitempty"`
	Group    string   `mapstructure:"group,omitempty" yaml:"group,omitempty" json:"group,omitempty" toml:"group,omitempty"`
	Mode     int      `mapstructure:"mode,omitempty" yaml:"mode,omitempty" json:"mode,omitempty" toml:"mode,omitempty"`
	Elevated bool     `mapstructure:"elevated" yaml:"elevated" json:"elevated" toml:"elevated"`
}

type FileData struct {
	Files       []File      `yaml:"files" json:"files" toml:"files"`
	Directories []Directory `yaml:"directories" json:"directories" toml:"directories"`
	Templates   []Template  `yaml:"templates" json:"templates" toml:"templates"`
}
