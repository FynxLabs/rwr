package types

type Template struct {
	Name    string `mapstructure:"name" yaml:"name" json:"name" toml:"name"`                                                     // Name of the template 	// Names of the templates
	Action  string `mapstructure:"action" yaml:"action" json:"action" toml:"action"`                                             // Action to perform with the template
	Content string `mapstructure:"content,omitempty" yaml:"content,omitempty" json:"content,omitempty" toml:"content,omitempty"` // Content of the template
	Source  string `mapstructure:"source,omitempty" yaml:"source,omitempty" json:"source,omitempty" toml:"source,omitempty"`     // Source of the template
	Target  string `mapstructure:"target" yaml:"target" json:"target" toml:"target"`                                             // Target of the template
	Owner   int    `mapstructure:"owner,omitempty" yaml:"owner,omitempty" json:"owner,omitempty" toml:"owner,omitempty"`         // Owner of the template
	Group   int    `mapstructure:"group,omitempty" yaml:"group,omitempty" json:"group,omitempty" toml:"group,omitempty"`         // Group of the template
	Mode    int    `mapstructure:"mode,omitempty" yaml:"mode,omitempty" json:"mode,omitempty" toml:"mode,omitempty"`             // Mode of the template
	Create  bool   `mapstructure:"create,omitempty" yaml:"create,omitempty" json:"create,omitempty" toml:"create,omitempty"`     // Whether to create the template
}
