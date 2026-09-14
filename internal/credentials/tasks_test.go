package credentials

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestReadGPGRevocationConfinesFileAccess(t *testing.T) {
	fingerprint := strings.Repeat("a", 40)
	for _, mode := range []string{"regular", "missing", "file symlink", "directory symlink"} {
		t.Run(mode, func(t *testing.T) {
			home := t.TempDir()
			revocations := filepath.Join(home, "openpgp-revocs.d")
			if err := os.Mkdir(revocations, 0700); err != nil {
				t.Fatal(err)
			}
			filename := strings.ToUpper(fingerprint) + ".rev"
			certificate := filepath.Join(revocations, filename)
			want := []byte("test revocation certificate")
			switch mode {
			case "regular":
				if err := os.WriteFile(certificate, want, 0600); err != nil {
					t.Fatal(err)
				}
			case "file symlink", "directory symlink":
				outside := t.TempDir()
				if err := os.WriteFile(filepath.Join(outside, filename), want, 0600); err != nil {
					t.Fatal(err)
				}
				target, link := filepath.Join(outside, filename), certificate
				if mode == "directory symlink" {
					if err := os.Remove(revocations); err != nil {
						t.Fatal(err)
					}
					target, link = outside, revocations
				}
				if err := os.Symlink(target, link); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			}
			got, err := readGPGRevocation(home, fingerprint)
			switch mode {
			case "regular":
				if err != nil || !bytes.Equal(got, want) {
					t.Fatalf("read certificate: %v", err)
				}
			case "missing":
				if !os.IsNotExist(err) {
					t.Fatalf("missing certificate must remain optional: %v", err)
				}
			default:
				if err == nil || len(got) != 0 {
					t.Fatal("read certificate outside GPG home")
				}
			}
		})
	}
}

func TestGPGVerifyBeforeImport(t *testing.T) {
	if _, err := exec.LookPath("gpg"); err != nil {
		t.Skip("gpg unavailable")
	}
	ctx := context.Background()
	source, cleanup, err := privateGPGHome()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	passphrase := "test-only-passphrase"
	if _, err := gpg(ctx, source, []string{"--pinentry-mode", "loopback", "--passphrase-fd", "0", "--quick-generate-key", "RWR Disposable <rwr@example.invalid>", "ed25519", "sign", "1d"}, []byte(passphrase+"\n")); err != nil {
		t.Fatal(err)
	}
	raw, err := gpg(ctx, source, []string{"--with-colons", "--list-secret-keys"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	fp := keyFingerprints(raw)[0]
	material, err := gpg(ctx, source, []string{"--armor", "--pinentry-mode", "loopback", "--passphrase-fd", "0", "--export-secret-keys", fp}, []byte(passphrase+"\n"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, fp, password string
		valid              bool
	}{{"valid", fp, passphrase, true}, {"wrong key", "0000000000000000000000000000000000000000", passphrase, false}, {"wrong password", fp, "wrong-passphrase", false}} {
		t.Run(tt.name, func(t *testing.T) {
			home, cleanup, err := verifyGPG(ctx, material, tt.fp, tt.password)
			if (err == nil) != tt.valid {
				t.Fatalf("validation error=%v", err)
			}
			if cleanup != nil {
				info, err := os.Stat(home)
				if err != nil || info.Mode().Perm() != 0700 {
					t.Fatal("temporary keyring permissions")
				}
				cleanup()
				if _, err := os.Stat(home); !os.IsNotExist(err) {
					t.Fatal("temporary material remains")
				}
			}
		})
	}
	// Use the same short, private home as verification. t.TempDir embeds the
	// test name beneath macOS's already long TMPDIR, exceeding the agent socket
	// path limit; it also need not have GPG's required owner-only permissions.
	target, cleanupTarget, err := privateGPGHome()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTarget()
	t.Setenv("GNUPGHOME", target)
	t.Setenv("RWR_TEST_PASSPHRASE", passphrase)
	c := &types.InitConfig{Credentials: []types.CredentialSpec{{Name: "passphrase", Sources: []string{"env:RWR_TEST_PASSPHRASE"}}}, CredentialAttachments: []types.CredentialAttachment{{Name: "key", Connection: "test", Filename: "private.asc", Item: "one"}}}
	r := NewResolver(c)
	defer r.Close()
	session := &taskSession{material: material}
	r.Attach("test", session)
	runner := TaskRunner{Resolver: r, Connection: "test"}
	task := types.CredentialTask{Kind: "gpg-restore", Name: "signing", Source: "key", Fingerprint: fp, Passphrase: "passphrase"}
	if err := runner.Run(ctx, task); err != nil {
		t.Fatal(err)
	}
	if session.reads != 1 {
		t.Fatal("attachment not fetched")
	}
	if err := runner.Run(ctx, task); err != nil {
		t.Fatal(err)
	}
	if session.reads != 1 {
		t.Fatal("already-present key touched vault")
	}
	if _, err := os.Stat(filepath.Join(target, "pubring.kbx")); err != nil {
		t.Fatal("key not imported")
	}
}

type taskSession struct {
	material []byte
	reads    int
}

func (s *taskSession) ReadSecret(context.Context, types.CredentialReference) (string, error) {
	return "", nil
}
func (s *taskSession) Close() error { return nil }
func (s *taskSession) ReadAttachment(context.Context, types.CredentialAttachment) ([]byte, error) {
	s.reads++
	return bytes.Clone(s.material), nil
}
func (s *taskSession) ReplaceAttachment(context.Context, types.CredentialAttachment, []byte) error {
	return nil
}

func TestTaskBindingCannotBroadenAccess(t *testing.T) {
	r := NewResolver(&types.InitConfig{CredentialAttachments: []types.CredentialAttachment{{Name: "one", Connection: "other", Item: "one", Filename: "private.asc"}}})
	defer r.Close()
	task := types.CredentialTask{Kind: "gpg-restore", Source: "one"}
	if err := (TaskRunner{Resolver: r, Connection: "test"}).Validate(task); err == nil {
		t.Fatal("cross-connection access allowed")
	}
	task.Source = "unknown"
	if err := (TaskRunner{Resolver: r, Connection: "test"}).Validate(task); err == nil {
		t.Fatal("undeclared attachment allowed")
	}
}

func TestKeyringMaterializationIsConnectionNamespaced(t *testing.T) {
	ring := &fakeKeyring{}
	withFakes(t, ring, false, nil)
	t.Setenv("RWR_TEST_VALUE", "one-value")
	c := &types.InitConfig{CredentialProviders: []types.CredentialConnection{{Name: "one", Provider: "fixture", Account: "one@example.invalid"}, {Name: "two", Provider: "fixture", Account: "two@example.invalid"}}, Credentials: []types.CredentialSpec{{Name: "password", Scope: []string{"credentials"}, Sources: []string{"env:RWR_TEST_VALUE"}}}}
	r := NewResolver(c)
	defer r.Close()
	task := types.CredentialTask{Name: "cache", Kind: "keyring", Credential: "password"}
	if err := (TaskRunner{Resolver: r, Connection: "one"}).Run(context.Background(), task); err == nil {
		t.Fatal("reference-less credential accepted for a namespaced keyring task")
	}
	if len(ring.entries) != 0 {
		t.Fatal("invalid task wrote the keyring")
	}
	c.Credentials[0].References = []types.CredentialReference{{Connection: "one", Item: "fixture", Field: "password"}}
	if err := (TaskRunner{Resolver: r, Connection: "one"}).Run(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RWR_TEST_VALUE", "two-value")
	c.Credentials[0].References[0].Connection = "two"
	if err := (TaskRunner{Resolver: r, Connection: "two"}).Run(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RWR_TEST_VALUE", "")
	c.Credentials[0].Sources = []string{"keyring"}
	if len(ring.entries) != 2 {
		t.Fatalf("accounts shared a keyring identity: %v", ring.entries)
	}
	for _, connection := range c.CredentialProviders {
		c.Credentials[0].References[0].Connection = connection.Name
		got, err := r.Read(context.Background(), "password")
		if err != nil || got != connection.Name+"-value" {
			t.Fatal("incorrect materialization")
		}
	}
	c.Credentials[0].References = []types.CredentialReference{{Connection: "one", Item: "otp", Field: "totp"}}
	if err := (TaskRunner{Resolver: r, Connection: "one"}).Validate(task); err == nil {
		t.Fatal("TOTP persistence accepted")
	}
}
