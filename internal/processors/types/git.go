package types

type GitOptions struct {
	URL     string `mapstructure:"url" yaml:"url,omitempty" json:"url,omitempty" toml:"url,omitempty"`                           // URL of the git repository
	Private bool   `mapstructure:"private,omitempty" yaml:"private,omitempty" json:"private,omitempty" toml:"private,omitempty"` // Whether the repository is private
	Target  string `mapstructure:"target" yaml:"target,omitempty" json:"target,omitempty" toml:"target,omitempty"`               // Target directory for the repository
	Update  bool   `mapstructure:"update,omitempty" yaml:"update,omitempty" json:"update,omitempty" toml:"update,omitempty"`     // Whether to update the repository
	Branch  string `mapstructure:"branch,omitempty" yaml:"branch,omitempty" json:"branch,omitempty" toml:"branch,omitempty"`     // Branch of the repository
}

type Git struct {
	Name    string `mapstructure:"name" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`                       // Name of the git operation
	Action  string `mapstructure:"action" yaml:"action,omitempty" json:"action,omitempty" toml:"action,omitempty"`               // Action to perform with git
	Path    string `mapstructure:"path" yaml:"path,omitempty" json:"path,omitempty" toml:"path,omitempty"`                       // Path for the git operation
	URL     string `mapstructure:"url" yaml:"url,omitempty" json:"url,omitempty" toml:"url,omitempty"`                           // URL of the git repository
	Branch  string `mapstructure:"branch,omitempty" yaml:"branch,omitempty" json:"branch,omitempty" toml:"branch,omitempty"`     // Branch of the repository
	Private bool   `mapstructure:"private,omitempty" yaml:"private,omitempty" json:"private,omitempty" toml:"private,omitempty"` // Whether the repository is private
}
