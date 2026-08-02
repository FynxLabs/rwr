package types

type Repository struct {
	Name           string   `mapstructure:"name" yaml:"name" json:"name" toml:"name"`                                                                     // Name of the repository
	Profiles       []string `mapstructure:"profiles,omitempty" yaml:"profiles,omitempty" json:"profiles,omitempty" toml:"profiles,omitempty"`             // Profiles this repository belongs to
	PackageManager string   `mapstructure:"package_manager" yaml:"package_manager" json:"package_manager" toml:"package_manager"`                         // Package manager to use
	Action         string   `mapstructure:"action" yaml:"action" json:"action" toml:"action"`                                                             // Action to perform with the repository
	URL            string   `mapstructure:"url" yaml:"url" json:"url" toml:"url"`                                                                         // URL of the repository
	Arch           string   `mapstructure:"arch,omitempty" yaml:"arch,omitempty" json:"arch,omitempty" toml:"arch,omitempty"`                             // Architecture of the repository
	KeyURL         string   `mapstructure:"key_url,omitempty" yaml:"key_url,omitempty" json:"key_url,omitempty" toml:"key_url,omitempty"`                 // Key URL of the repository
	Channel        string   `mapstructure:"channel,omitempty" yaml:"channel,omitempty" json:"channel,omitempty" toml:"channel,omitempty"`                 // Channel of the repository
	Component      string   `mapstructure:"component,omitempty" yaml:"component,omitempty" json:"component,omitempty" toml:"component,omitempty"`         // Component of the repository
	Repository     string   `mapstructure:"repository,omitempty" yaml:"repository,omitempty" json:"repository,omitempty" toml:"repository,omitempty"`     // Repository name
	KeyID          string   `mapstructure:"key_id,omitempty" yaml:"key_id,omitempty" json:"key_id,omitempty" toml:"key_id,omitempty"`                     // Key ID to import or delete (pacman-family, dnf, zypper)
	Description    string   `mapstructure:"description,omitempty" yaml:"description,omitempty" json:"description,omitempty" toml:"description,omitempty"` // Human-readable repository description (dnf .repo name=)
	OverlayPath    string   `mapstructure:"overlay_path,omitempty" yaml:"overlay_path,omitempty" json:"overlay_path,omitempty" toml:"overlay_path,omitempty"`
	SyncType       string   `mapstructure:"sync_type,omitempty" yaml:"sync_type,omitempty" json:"sync_type,omitempty" toml:"sync_type,omitempty"` // Portage sync-type, e.g. "git" or "rsync"
	SHA256         string   `mapstructure:"sha256,omitempty" yaml:"sha256,omitempty" json:"sha256,omitempty" toml:"sha256,omitempty"`             // Digest of a fetched nix overlay tarball
	ProxyURL       string   `mapstructure:"proxy_url,omitempty" yaml:"proxy_url,omitempty" json:"proxy_url,omitempty" toml:"proxy_url,omitempty"` // Proxy snapd should route through
	UUID           string   `mapstructure:"uuid,omitempty" yaml:"uuid,omitempty" json:"uuid,omitempty" toml:"uuid,omitempty"`                     // GNOME extension UUID
	ExtensionID    string   `mapstructure:"extension_id,omitempty" yaml:"extension_id,omitempty" json:"extension_id,omitempty" toml:"extension_id,omitempty"`
	Interactive    *bool    `mapstructure:"interactive,omitempty" yaml:"interactive,omitempty" json:"interactive,omitempty" toml:"interactive,omitempty"` // Override global interactive mode
	Import         string   `mapstructure:"import,omitempty" yaml:"import,omitempty" json:"import,omitempty" toml:"import,omitempty"`                     // Import path for external repository definitions

	// Credentials a provider needs to authenticate against a private repository:
	// chocolatey signs a source in with Username/Password, cargo with Token.
	//
	// They are treated like the credentials in secrets.go — never printed. Anything
	// rwr logs about a repository goes through Redact for these fields, because a
	// registry token in a debug log is as reusable as one in a blueprint, and rwr's
	// logs get pasted into issues.
	Username string `mapstructure:"username,omitempty" yaml:"username,omitempty" json:"username,omitempty" toml:"username,omitempty"`
	Password string `mapstructure:"password,omitempty" yaml:"password,omitempty" json:"password,omitempty" toml:"password,omitempty"`
	Token    string `mapstructure:"token,omitempty" yaml:"token,omitempty" json:"token,omitempty" toml:"token,omitempty"`
}

// LogString renders a repository for logging with its credentials redacted.
func (r Repository) LogString() string {
	out := r.Name + " (" + r.PackageManager + " " + r.Action + ")"
	if r.Username != "" {
		out += " username=" + Redact(r.Username)
	}
	if r.Password != "" {
		out += " password=" + Redact(r.Password)
	}
	if r.Token != "" {
		out += " token=" + Redact(r.Token)
	}
	return out
}

type RepositoriesData struct {
	// SchemaVersion, when set, overrides the tree-wide version from the init file.
	SchemaVersion `mapstructure:",squash" yaml:",inline" json:",inline" toml:",inline"`
	Repositories  []Repository `mapstructure:"repositories" yaml:"repositories" json:"repositories" toml:"repositories"`
}

// GetProfiles returns the profiles for this repository.
func (r Repository) GetProfiles() []string {
	return r.Profiles
}

// SecretValues returns the credential values this repository carries, for
// Command.Secrets.
//
// Some package managers accept a credential only as a command-line argument
// (chocolatey's --password, cargo's login token), so it cannot be moved to stdin
// the way a user password can. Listing the values here at least keeps them out of
// the debug and dry-run log lines, which print the whole argv. It does not hide
// them from `ps` — nothing can, short of the tool growing a stdin option.
//
// The username is not included: it is not a credential, and redacting it would
// only make the log harder to read.
func (r Repository) SecretValues() []string {
	var secrets []string
	for _, value := range []string{r.Password, r.Token} {
		if value != "" {
			secrets = append(secrets, value)
		}
	}
	return secrets
}
