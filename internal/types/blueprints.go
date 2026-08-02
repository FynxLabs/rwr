// Package types defines data structures and type definitions used throughout rwr.
// It provides type definitions for blueprints, configuration, initialization,
// packages, repositories, services, files, users, validation, and system information.
// These types form the core data model for blueprint processing and system management.
package types

type Init struct {
	Format string `mapstructure:"format" yaml:"format" json:"format" toml:"format"`
	// SchemaVersion is the blueprint schema version this tree is written in. It
	// applies to every blueprint type, and an individual blueprint file may
	// override it by declaring its own. Zero means undeclared, which resolves to
	// types.DefaultSchemaVersion.
	SchemaVersion int           `mapstructure:"schema_version,omitempty" yaml:"schema_version,omitempty" json:"schema_version,omitempty" toml:"schema_version,omitempty"`
	Location      string        `mapstructure:"location,omitempty" yaml:"location,omitempty" json:"location,omitempty" toml:"location,omitempty"`
	Order         []interface{} `mapstructure:"order,omitempty" yaml:"order,omitempty" json:"order,omitempty" toml:"order,omitempty"`
	Git           *GitOptions   `mapstructure:"git,omitempty" yaml:"git,omitempty" json:"git,omitempty" toml:"git,omitempty"`
	RunOnlyListed bool          `mapstructure:"runOnlyListed,omitempty" yaml:"runOnlyListed,omitempty" json:"runOnlyListed,omitempty" toml:"runOnlyListed,omitempty"`
}
