package types

type GitOptions struct {
	URL     string `mapstructure:"url"`               // URL of the git repository
	Private bool   `mapstructure:"private,omitempty"` // Whether the repository is private
	Target  string `mapstructure:"target"`            // Target directory for the repository
	Update  bool   `mapstructure:"update,omitempty"`  // Whether to update the repository
	Branch  string `mapstructure:"branch,omitempty"`  // Branch of the repository
}

type Git struct {
	Name    string `mapstructure:"name"`              // Name of the git operation
	Action  string `mapstructure:"action"`            // Action to perform with git
	Path    string `mapstructure:"path"`              // Path for the git operation
	URL     string `mapstructure:"url"`               // URL of the git repository
	Branch  string `mapstructure:"branch,omitempty"`  // Branch of the repository
	Private bool   `mapstructure:"private,omitempty"` // Whether the repository is private
}
