package types

type File struct {
	Name    string   `mapstructure:"name,omitempty"`    // Name of the file
	Names   []string `mapstructure:"names,omitempty"`   // Names of the files
	Action  string   `mapstructure:"action"`            // Action to perform with the file
	Content string   `mapstructure:"content,omitempty"` // Content of the file
	Source  string   `mapstructure:"source,omitempty"`  // Source of the file
	Target  string   `mapstructure:"target,omitempty"`  // Target of the file
	Owner   int      `mapstructure:"owner,omitempty"`   // Owner of the file
	Group   int      `mapstructure:"group,omitempty"`   // Group of the file
	Mode    int      `mapstructure:"mode,omitempty"`    // Mode of the file
	Create  bool     `mapstructure:"create,omitempty"`  // Whether to create the file
}

type Directory struct {
	Name   string   `mapstructure:"name,omitempty"`   // Name of the directory
	Names  []string `mapstructure:"names,omitempty"`  // Names of the directories
	Action string   `mapstructure:"action"`           // Action to perform with the directory
	Source string   `mapstructure:"source,omitempty"` // Source of the directory
	Target string   `mapstructure:"target,omitempty"` // Target of the directory
	Owner  int      `mapstructure:"owner,omitempty"`  // Owner of the directory
	Group  int      `mapstructure:"group,omitempty"`  // Group of the directory
	Mode   int      `mapstructure:"mode,omitempty"`   // Mode of the directory
	Create bool     `mapstructure:"create,omitempty"` // Whether to create the directory
}
