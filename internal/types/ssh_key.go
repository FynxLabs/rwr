package types

type SSHKey struct {
	Type         string `mapstructure:"type" yaml:"type" json:"type" toml:"type"`
	Path         string `mapstructure:"path" yaml:"path" json:"path" toml:"path"`
	Comment      string `mapstructure:"comment" yaml:"comment" json:"comment" toml:"comment"`
	NoPassphrase bool   `mapstructure:"no_passphrase" yaml:"no_passphrase" json:"no_passphrase" toml:"no_passphrase"`
	CopyToGitHub bool   `mapstructure:"copy_to_github" yaml:"copy_to_github" json:"copy_to_github" toml:"copy_to_github"`
}

type SSHKeyData struct {
	SSHKeys []SSHKey `mapstructure:"ssh_keys" yaml:"ssh_keys" json:"ssh_keys" toml:"ssh_keys"`
}
