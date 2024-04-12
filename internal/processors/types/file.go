package types

type File struct {
	Name    string   `mapstructure:"name,omitempty" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`             // Name of the file
	Names   []string `mapstructure:"names,omitempty" yaml:"names,omitempty" json:"names,omitempty" toml:"names,omitempty"`         // Names of the files
	Action  string   `mapstructure:"action" yaml:"action,omitempty" json:"action,omitempty" toml:"action,omitempty"`               // Action to perform with the file
	Content string   `mapstructure:"content,omitempty" yaml:"content,omitempty" json:"content,omitempty" toml:"content,omitempty"` // Content of the file
	Source  string   `mapstructure:"source,omitempty" yaml:"source,omitempty" json:"source,omitempty" toml:"source,omitempty"`     // Source of the file
	Target  string   `mapstructure:"target,omitempty" yaml:"target,omitempty" json:"target,omitempty" toml:"target,omitempty"`     // Target of the file
	Owner   int      `mapstructure:"owner,omitempty" yaml:"owner,omitempty" json:"owner,omitempty" toml:"owner,omitempty"`         // Owner of the file
	Group   int      `mapstructure:"group,omitempty" yaml:"group,omitempty" json:"group,omitempty" toml:"group,omitempty"`         // Group of the file
	Mode    int      `mapstructure:"mode,omitempty" yaml:"mode,omitempty" json:"mode,omitempty" toml:"mode,omitempty"`             // Mode of the file
	Create  bool     `mapstructure:"create,omitempty" yaml:"create,omitempty" json:"create,omitempty" toml:"create,omitempty"`     // Whether to create the file
}

type Directory struct {
	Name   string   `mapstructure:"name,omitempty" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`         // Name of the directory
	Names  []string `mapstructure:"names,omitempty" yaml:"names,omitempty" json:"names,omitempty" toml:"names,omitempty"`     // Names of the directories
	Action string   `mapstructure:"action" yaml:"action,omitempty" json:"action,omitempty" toml:"action,omitempty"`           // Action to perform with the directory
	Source string   `mapstructure:"source,omitempty" yaml:"source,omitempty" json:"source,omitempty" toml:"source,omitempty"` // Source of the directory
	Target string   `mapstructure:"target,omitempty" yaml:"target,omitempty" json:"target,omitempty" toml:"target,omitempty"` // Target of the directory
	Owner  int      `mapstructure:"owner,omitempty" yaml:"owner,omitempty" json:"owner,omitempty" toml:"owner,omitempty"`     // Owner of the directory
	Group  int      `mapstructure:"group,omitempty" yaml:"group,omitempty" json:"group,omitempty" toml:"group,omitempty"`     // Group of the directory
	Mode   int      `mapstructure:"mode,omitempty" yaml:"mode,omitempty" json:"mode,omitempty" toml:"mode,omitempty"`         // Mode of the directory
	Create bool     `mapstructure:"create,omitempty" yaml:"create,omitempty" json:"create,omitempty" toml:"create,omitempty"` // Whether to create the directory
}
