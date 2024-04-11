package types

type Template struct {
	Name      string                 `mapstructure:"name,omitempty"`      // Name of the template
	Names     []string               `mapstructure:"names,omitempty"`     // Names of the templates
	Action    string                 `mapstructure:"action"`              // Action to perform with the template
	Content   string                 `mapstructure:"content,omitempty"`   // Content of the template
	Source    string                 `mapstructure:"source,omitempty"`    // Source of the template
	Target    string                 `mapstructure:"target"`              // Target of the template
	Owner     int                    `mapstructure:"owner,omitempty"`     // Owner of the template
	Group     int                    `mapstructure:"group,omitempty"`     // Group of the template
	Mode      int                    `mapstructure:"mode,omitempty"`      // Mode of the template
	Create    bool                   `mapstructure:"create,omitempty"`    // Whether to create the template
	Variables map[string]interface{} `mapstructure:"variables,omitempty"` // Variables of the template
}
