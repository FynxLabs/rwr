package types

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

type CredentialConnection struct {
	Name       string `mapstructure:"name,omitempty" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`
	Provider   string `mapstructure:"provider,omitempty" yaml:"provider,omitempty" json:"provider,omitempty" toml:"provider,omitempty"`
	Server     string `mapstructure:"server,omitempty" yaml:"server,omitempty" json:"server,omitempty" toml:"server,omitempty"`
	Account    string `mapstructure:"account,omitempty" yaml:"account,omitempty" json:"account,omitempty" toml:"account,omitempty"`
	SessionEnv string `mapstructure:"sessionEnv,omitempty" yaml:"sessionEnv,omitempty" json:"sessionEnv,omitempty" toml:"sessionEnv,omitempty"`
}

type CredentialReference struct {
	Connection string `mapstructure:"connection,omitempty" yaml:"connection,omitempty" json:"connection,omitempty" toml:"connection,omitempty"`
	Item       string `mapstructure:"item,omitempty" yaml:"item,omitempty" json:"item,omitempty" toml:"item,omitempty"`
	Field      string `mapstructure:"field,omitempty" yaml:"field,omitempty" json:"field,omitempty" toml:"field,omitempty"`
}

type CredentialAttachment struct {
	Name       string `mapstructure:"name,omitempty" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`
	Connection string `mapstructure:"connection,omitempty" yaml:"connection,omitempty" json:"connection,omitempty" toml:"connection,omitempty"`
	Item       string `mapstructure:"item,omitempty" yaml:"item,omitempty" json:"item,omitempty" toml:"item,omitempty"`
	Filename   string `mapstructure:"filename,omitempty" yaml:"filename,omitempty" json:"filename,omitempty" toml:"filename,omitempty"`
	Write      bool   `mapstructure:"write,omitempty" yaml:"write,omitempty" json:"write,omitempty" toml:"write,omitempty"`
}

type CredentialSetup struct {
	Name       string           `mapstructure:"name,omitempty" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`
	Profiles   []string         `mapstructure:"profiles,omitempty" yaml:"profiles,omitempty" json:"profiles,omitempty" toml:"profiles,omitempty"`
	Import     string           `mapstructure:"import,omitempty" yaml:"import,omitempty" json:"import,omitempty" toml:"import,omitempty"`
	Connection string           `mapstructure:"connection,omitempty" yaml:"connection,omitempty" json:"connection,omitempty" toml:"connection,omitempty"`
	Install    string           `mapstructure:"install,omitempty" yaml:"install,omitempty" json:"install,omitempty" toml:"install,omitempty"`
	Session    string           `mapstructure:"session,omitempty" yaml:"session,omitempty" json:"session,omitempty" toml:"session,omitempty"`
	Tasks      []CredentialTask `mapstructure:"tasks,omitempty" yaml:"tasks,omitempty" json:"tasks,omitempty" toml:"tasks,omitempty"`
}

type CredentialTask struct {
	Name                string `mapstructure:"name,omitempty" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`
	Kind                string `mapstructure:"kind,omitempty" yaml:"kind,omitempty" json:"kind,omitempty" toml:"kind,omitempty"`
	Source              string `mapstructure:"source,omitempty" yaml:"source,omitempty" json:"source,omitempty" toml:"source,omitempty"`
	PublicSource        string `mapstructure:"publicSource,omitempty" yaml:"publicSource,omitempty" json:"publicSource,omitempty" toml:"publicSource,omitempty"`
	RevocationSource    string `mapstructure:"revocationSource,omitempty" yaml:"revocationSource,omitempty" json:"revocationSource,omitempty" toml:"revocationSource,omitempty"`
	Credential          string `mapstructure:"credential,omitempty" yaml:"credential,omitempty" json:"credential,omitempty" toml:"credential,omitempty"`
	Fingerprint         string `mapstructure:"fingerprint,omitempty" yaml:"fingerprint,omitempty" json:"fingerprint,omitempty" toml:"fingerprint,omitempty"`
	Passphrase          string `mapstructure:"passphrase,omitempty" yaml:"passphrase,omitempty" json:"passphrase,omitempty" toml:"passphrase,omitempty"`
	OwnerTrust          int    `mapstructure:"ownerTrust,omitempty" yaml:"ownerTrust,omitempty" json:"ownerTrust,omitempty" toml:"ownerTrust,omitempty"`
	ConfigureGitSigning bool   `mapstructure:"configureGitSigning,omitempty" yaml:"configureGitSigning,omitempty" json:"configureGitSigning,omitempty" toml:"configureGitSigning,omitempty"`
	WriteProfile        string `mapstructure:"writeProfile,omitempty" yaml:"writeProfile,omitempty" json:"writeProfile,omitempty" toml:"writeProfile,omitempty"`
}

type CredentialDependencies struct {
	RequiresCredentials     []string `mapstructure:"requiresCredentials,omitempty" yaml:"requiresCredentials,omitempty" json:"requiresCredentials,omitempty" toml:"requiresCredentials,omitempty"`
	OnCredentialUnavailable string   `mapstructure:"onCredentialUnavailable,omitempty" yaml:"onCredentialUnavailable,omitempty" json:"onCredentialUnavailable,omitempty" toml:"onCredentialUnavailable,omitempty"`
}

type CredentialSetupData struct {
	SchemaVersion `mapstructure:",squash" yaml:",inline" json:",inline" toml:",inline"`
	Entries       []CredentialSetup `mapstructure:"credential_setup" yaml:"credential_setup" json:"credential_setup" toml:"credential_setup"`
}

func (c CredentialSetup) GetProfiles() []string { return c.Profiles }

var fingerprintPattern = regexp.MustCompile(`^[A-Fa-f0-9]{40}$|^[A-Fa-f0-9]{64}$`)

func (c CredentialSetupData) Validate() error {
	seen := map[string]bool{}
	for _, e := range c.Entries {
		if e.Import != "" {
			if e.Name != "" || e.Connection != "" || len(e.Tasks) > 0 {
				return fmt.Errorf("credential import cannot also declare setup")
			}
			continue
		}
		if e.Name == "" || e.Connection == "" {
			return fmt.Errorf("credential setup requires name and connection")
		}
		if seen[e.Name] {
			return fmt.Errorf("duplicate credential setup %q", e.Name)
		}
		seen[e.Name] = true
		if e.Install != "" && e.Install != "never" && e.Install != "if-missing" {
			return fmt.Errorf("install must be never or if-missing")
		}
		if e.Session != "" && e.Session != "ensure-ready" && e.Session != "existing" {
			return fmt.Errorf("session must be ensure-ready or existing")
		}
		for _, t := range e.Tasks {
			if t.Name == "" {
				return fmt.Errorf("credential task requires name")
			}
			switch t.Kind {
			case "keyring":
				if t.Credential == "" {
					return fmt.Errorf("keyring task requires credential")
				}
			case "gpg-restore", "gpg-backup":
				if t.Source == "" || !fingerprintPattern.MatchString(t.Fingerprint) || t.Passphrase == "" {
					return fmt.Errorf("GPG task requires source, full fingerprint and passphrase reference")
				}
				if t.OwnerTrust < 0 || t.OwnerTrust > 6 || t.OwnerTrust == 1 {
					return fmt.Errorf("ownerTrust must be omitted or 2..6")
				}
				if t.Kind == "gpg-backup" && t.WriteProfile == "" {
					return fmt.Errorf("GPG backup requires writeProfile")
				}
			default:
				return fmt.Errorf("unsupported credential task kind %q", t.Kind)
			}
		}
	}
	return nil
}
func ValidateCredentialConnections(c *InitConfig) error {
	if err := c.CredentialPolicy.Validate(); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, p := range c.CredentialProviders {
		if p.Name == "" || p.Provider == "" || seen[p.Name] {
			return fmt.Errorf("credential providers require unique names and provider IDs")
		}
		seen[p.Name] = true
	}
	bindings := map[string]bool{}
	for _, a := range c.CredentialAttachments {
		if a.Name == "" || bindings[a.Name] || !seen[a.Connection] || a.Item == "" || a.Filename == "" || a.Filename == "." || a.Filename == ".." || filepath.Base(a.Filename) != a.Filename || strings.ContainsAny(a.Filename, "/\\") {
			return fmt.Errorf("invalid credential attachment binding %q", a.Name)
		}
		bindings[a.Name] = true
	}
	for _, s := range c.Credentials {
		for _, r := range s.References {
			if !seen[r.Connection] || r.Item == "" || strings.HasPrefix(r.Item, "-") {
				return fmt.Errorf("credential %q has an invalid connection/item reference", s.Name)
			}
		}
	}
	return nil
}
