package provider

import (
	"fmt"
	"os/exec"
)

type Provider struct {
	Name       string           `yaml:"name"`
	Elevated   bool             `yaml:"elevated"`
	Detection  DetectionConfig  `yaml:"detection"`
	Commands   CommandConfig    `yaml:"commands"`
	Repository RepositoryConfig `yaml:"repository"`
}

type DetectionConfig struct {
	Binary        string   `yaml:"binary"`
	Files         []string `yaml:"files"`
	Distributions []string `yaml:"distributions"`
}

type CommandConfig struct {
	Install string `yaml:"install"`
	Update  string `yaml:"update"`
	Remove  string `yaml:"remove"`
	List    string `yaml:"list"`
	Search  string `yaml:"search"`
	Clean   string `yaml:"clean"`
}

type RepositoryConfig struct {
	Paths  RepositoryPaths  `yaml:"paths"`
	Add    RepositoryAction `yaml:"add"`
	Remove RepositoryAction `yaml:"remove"`
}

type RepositoryPaths struct {
	Sources string `yaml:"sources"`
	Keys    string `yaml:"keys"`
	Config  string `yaml:"config"`
}

type RepositoryAction struct {
	Steps []ActionStep `yaml:"steps"`
}

type ActionStep struct {
	Action  string   `yaml:"action"`
	Source  string   `yaml:"source,omitempty"`
	Dest    string   `yaml:"dest,omitempty"`
	Exec    string   `yaml:"exec,omitempty"`
	Args    []string `yaml:"args,omitempty"`
	Content string   `yaml:"content,omitempty"`
}

// repository/loader.go
