package credentials

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

type TaskRunner struct {
	Resolver   *Resolver
	Connection string
}

func (t TaskRunner) binding(name string, write bool) (types.CredentialAttachment, error) {
	for _, a := range t.Resolver.Config.CredentialAttachments {
		if a.Name == name {
			if a.Connection != t.Connection {
				return a, fmt.Errorf("attachment belongs to another connection")
			}
			if write && !a.Write {
				return a, fmt.Errorf("attachment does not authorize writes")
			}
			return a, nil
		}
	}
	return types.CredentialAttachment{}, fmt.Errorf("undeclared attachment %q", name)
}
func (t TaskRunner) Validate(task types.CredentialTask) error {
	secretName := task.Passphrase
	if task.Kind == "keyring" {
		secretName = task.Credential
	}
	for _, spec := range t.Resolver.Config.Credentials {
		if spec.Name != secretName {
			continue
		}
		if len(spec.Scope) > 0 {
			allowed := false
			for _, scope := range spec.Scope {
				if scope == types.BlueprintTypeCredentials {
					allowed = true
				}
			}
			if !allowed {
				return fmt.Errorf("credential is not scoped to native credential tasks")
			}
		}
		for _, ref := range spec.References {
			if ref.Connection != t.Connection {
				return fmt.Errorf("task credential belongs to another connection")
			}
		}
	}

	if task.Kind == "keyring" {
		for _, s := range t.Resolver.Config.Credentials {
			if s.Name == task.Credential {
				if len(s.References) == 0 {
					return fmt.Errorf("keyring tasks require a provider-backed credential with references")
				}
				for _, ref := range s.References {
					if ref.Field == "totp" {
						return fmt.Errorf("TOTP values cannot be persisted")
					}
				}
				for _, source := range s.Sources {
					if strings.HasSuffix(source, "/totp") {
						return fmt.Errorf("TOTP values cannot be persisted")
					}
				}
				return nil
			}
		}
		return fmt.Errorf("undeclared credential %q", task.Credential)
	}
	if _, err := t.binding(task.Source, task.Kind == "gpg-backup"); err != nil {
		return err
	}
	for _, name := range []string{task.PublicSource, task.RevocationSource} {
		if name != "" {
			if _, err := t.binding(name, task.Kind == "gpg-backup"); err != nil {
				return err
			}
		}
	}
	for _, s := range t.Resolver.Config.Credentials {
		if s.Name == task.Passphrase {
			return nil
		}
	}
	return fmt.Errorf("undeclared passphrase credential")
}
func (t TaskRunner) Run(ctx context.Context, task types.CredentialTask) error {
	if err := t.Validate(task); err != nil {
		return err
	}
	if system.IsDryRun() {
		return nil
	}
	if task.Kind == "keyring" {
		value, err := t.Resolver.ReadSetup(ctx, task.Credential)
		if err != nil {
			return err
		}
		c, err := t.Resolver.Connection(t.Connection)
		if err != nil {
			return err
		}
		return SaveToKeyring("v1/"+connectionIdentity(c)+"/"+task.Credential, value)
	}
	if task.Kind == "gpg-restore" {
		return t.restore(ctx, task)
	}
	if task.Kind == "gpg-backup" {
		return t.backup(ctx, task)
	}
	return fmt.Errorf("unsupported credential task")
}
func gpg(ctx context.Context, home string, args []string, input []byte) ([]byte, error) {
	env := map[string]string{}
	if home != "" {
		env["GNUPGHOME"] = home
	}
	args = append([]string{"--batch", "--no-tty"}, args...)
	// GPG diagnostics describe local keyring failures. Redact every input line,
	// including passphrases and armored key material, before surfacing them.
	return protectedCommand(ctx, "gpg", args, env, bytes.NewReader(input), strings.Split(string(input), "\n"))
}
func keyFingerprints(raw []byte) []string {
	var result []string
	primary := false
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Split(line, ":")
		if len(fields) < 10 {
			continue
		}
		switch fields[0] {
		case "sec", "pub":
			primary = true
		case "ssb", "sub":
			primary = false
		}
		if fields[0] == "fpr" && primary {
			result = append(result, strings.ToUpper(fields[9]))
			primary = false
		}
	}
	return result
}
func hasGPGKey(ctx context.Context, fp string) (bool, error) {
	raw, err := gpg(ctx, "", []string{"--with-colons", "--list-secret-keys"}, nil)
	if err != nil {
		return false, err
	}
	for _, found := range keyFingerprints(raw) {
		if strings.EqualFold(fp, found) {
			return true, nil
		}
	}
	return false, nil
}
func privateGPGHome() (string, func(), error) {
	dir, err := os.MkdirTemp("", "rwr-gpg-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() {
		_, _ = ProtectedCommand(context.Background(), "gpgconf", []string{"--homedir", dir, "--kill", "gpg-agent"}, nil, nil) //nolint:errcheck // Best effort shutdown of the temporary agent.
		_ = os.RemoveAll(dir)                                                                                                 //nolint:errcheck // Best effort cleanup.
	}
	if err := os.WriteFile(filepath.Join(dir, "gpg-agent.conf"), []byte("allow-loopback-pinentry\ndefault-cache-ttl 0\nmax-cache-ttl 0\n"), 0600); err != nil {
		cleanup()
		return "", nil, err
	}
	return dir, cleanup, nil
}
func verifyGPG(ctx context.Context, material []byte, fp, passphrase string) (string, func(), error) {
	dir, cleanup, err := privateGPGHome()
	if err != nil {
		return "", nil, err
	}
	fail := func(err error) (string, func(), error) { cleanup(); return "", nil, err }
	if _, err := gpg(ctx, dir, []string{"--import"}, material); err != nil {
		return fail(fmt.Errorf("temporary key import failed: %w", err))
	}
	raw, err := gpg(ctx, dir, []string{"--with-colons", "--list-keys"}, nil)
	if err != nil {
		return fail(err)
	}
	keys := keyFingerprints(raw)
	if len(keys) != 1 || !strings.EqualFold(keys[0], fp) {
		return fail(fmt.Errorf("attachment contains unexpected key identities"))
	}
	raw, err = gpg(ctx, dir, []string{"--with-colons", "--list-secret-keys"}, nil)
	if err != nil {
		return fail(err)
	}
	keys = keyFingerprints(raw)
	if len(keys) != 1 || !strings.EqualFold(keys[0], fp) {
		return fail(fmt.Errorf("attachment does not contain the expected private key"))
	}
	challenge := filepath.Join(dir, "challenge")
	if err := os.WriteFile(challenge, []byte("RWR signing identity verification\n"), 0600); err != nil {
		return fail(err)
	}
	args := []string{"--yes", "--pinentry-mode", "loopback", "--passphrase-fd", "0", "--local-user", fp, "--output", filepath.Join(dir, "challenge.sig"), "--detach-sign", challenge}
	if _, err := gpg(ctx, dir, args, []byte(passphrase+"-rwr-invalid\n")); err == nil {
		return fail(fmt.Errorf("private key is not protected by its declared passphrase"))
	}
	if _, err := gpg(ctx, dir, args, []byte(passphrase+"\n")); err != nil {
		return fail(fmt.Errorf("private key passphrase or signing verification failed"))
	}
	return dir, cleanup, nil
}
func configureGPG(ctx context.Context, task types.CredentialTask) error {
	if task.OwnerTrust != 0 {
		if _, err := gpg(ctx, "", []string{"--import-ownertrust"}, []byte(fmt.Sprintf("%s:%d:\n", task.Fingerprint, task.OwnerTrust))); err != nil {
			return err
		}
	}
	if task.ConfigureGitSigning {
		for _, args := range [][]string{{"config", "--global", "user.signingkey", task.Fingerprint}, {"config", "--global", "commit.gpgsign", "true"}} {
			if _, err := ProtectedCommand(ctx, "git", args, nil, nil); err != nil {
				return err
			}
		}
	}
	return nil
}
func (t TaskRunner) AlreadyPresent(ctx context.Context, task types.CredentialTask) (bool, error) {
	if task.Kind != "gpg-restore" {
		return false, nil
	}
	return hasGPGKey(ctx, task.Fingerprint)
}
func (t TaskRunner) restore(ctx context.Context, task types.CredentialTask) error {
	exists, err := t.AlreadyPresent(ctx, task)
	if err != nil {
		return err
	}
	if exists {
		return configureGPG(ctx, task)
	}
	binding, err := t.binding(task.Source, false)
	if err != nil {
		return err
	}
	session, err := t.Resolver.Session(ctx, t.Connection)
	if err != nil {
		return err
	}
	attachments, ok := session.(AttachmentSession)
	if !ok {
		return fmt.Errorf("provider does not support attachments")
	}
	material, err := attachments.ReadAttachment(ctx, binding)
	if err != nil {
		return err
	}
	passphrase, err := t.Resolver.ReadSetup(ctx, task.Passphrase)
	if err != nil {
		return err
	}
	_, cleanup, err := verifyGPG(ctx, material, task.Fingerprint, passphrase)
	if err != nil {
		return err
	}
	defer cleanup()
	if _, err := gpg(ctx, "", []string{"--import"}, material); err != nil {
		return fmt.Errorf("verified key import failed: %w", err)
	}
	return configureGPG(ctx, task)
}
func (t TaskRunner) backup(ctx context.Context, task types.CredentialTask) error {
	selected := false
	for _, p := range t.Resolver.Config.Variables.Flags.Profiles {
		if p == task.WriteProfile {
			selected = true
		}
	}
	if !selected {
		return &Unavailable{Skipped}
	}
	passphrase, err := t.Resolver.ReadSetup(ctx, task.Passphrase)
	if err != nil {
		return err
	}
	material, err := gpg(ctx, "", []string{"--armor", "--pinentry-mode", "loopback", "--passphrase-fd", "0", "--export-secret-keys", task.Fingerprint}, []byte(passphrase+"\n"))
	if err != nil {
		return err
	}
	_, cleanup, err := verifyGPG(ctx, material, task.Fingerprint, passphrase)
	if err != nil {
		return err
	}
	defer cleanup()
	session, err := t.Resolver.Session(ctx, t.Connection)
	if err != nil {
		return err
	}
	attachments, ok := session.(AttachmentSession)
	if !ok {
		return fmt.Errorf("provider does not support attachments")
	}
	binding, err := t.binding(task.Source, true)
	if err != nil {
		return err
	}
	if err := attachments.ReplaceAttachment(ctx, binding, material); err != nil {
		return err
	}
	if task.PublicSource != "" {
		public, err := gpg(ctx, "", []string{"--armor", "--export", task.Fingerprint}, nil)
		if err != nil {
			return err
		}
		binding, err := t.binding(task.PublicSource, true)
		if err != nil {
			return err
		}
		if err := attachments.ReplaceAttachment(ctx, binding, public); err != nil {
			return err
		}
	}
	if task.RevocationSource != "" {
		home := os.Getenv("GNUPGHOME")
		if home == "" {
			user, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			home = filepath.Join(user, ".gnupg")
		}
		material, err := readGPGRevocation(home, task.Fingerprint)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("revocation certificate unavailable")
		}
		binding, err := t.binding(task.RevocationSource, true)
		if err != nil {
			return err
		}
		if err := attachments.ReplaceAttachment(ctx, binding, material); err != nil {
			return err
		}
	}
	return nil
}

// Keep certificate reads within the configured GPG home, including when a
// certificate or its parent directory is a symlink.
func readGPGRevocation(home, fingerprint string) ([]byte, error) {
	root, err := os.OpenRoot(home)
	if err != nil {
		return nil, err
	}
	material, readErr := root.ReadFile(filepath.Join("openpgp-revocs.d", strings.ToUpper(fingerprint)+".rev"))
	closeErr := root.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return material, nil
}
