package types

// Manifest is the root file of a multi-configuration blueprint repo: one
// entry per machine shape, each pointing at that configuration's init file.
// Manifests are untrusted input like blueprints: init paths must resolve
// inside the repo, and matchers are data only.
type Manifest struct {
	Configurations []ManifestEntry `mapstructure:"configurations" yaml:"configurations" json:"configurations" toml:"configurations"`
}

// ManifestEntry is one named configuration and the machines it matches.
type ManifestEntry struct {
	Name    string `mapstructure:"name" yaml:"name" json:"name" toml:"name"`
	Init    string `mapstructure:"init" yaml:"init" json:"init" toml:"init"`
	OS      string `mapstructure:"os,omitempty" yaml:"os,omitempty" json:"os,omitempty" toml:"os,omitempty"`
	Distro  string `mapstructure:"distro,omitempty" yaml:"distro,omitempty" json:"distro,omitempty" toml:"distro,omitempty"`
	Family  string `mapstructure:"family,omitempty" yaml:"family,omitempty" json:"family,omitempty" toml:"family,omitempty"`
	Arch    string `mapstructure:"arch,omitempty" yaml:"arch,omitempty" json:"arch,omitempty" toml:"arch,omitempty"`
	Default bool   `mapstructure:"default,omitempty" yaml:"default,omitempty" json:"default,omitempty" toml:"default,omitempty"`
}
