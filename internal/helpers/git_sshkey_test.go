package helpers

import (
	"encoding/base64"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

// generateKey writes a real ed25519 keypair and returns its path and PEM.
// A hand-rolled fixture would not prove anything here: the point is that what
// ssh-keygen produces, and what rwr stores about it, round-trips through
// go-git.
func generateKey(t *testing.T) (path string, pem []byte) {
	t.Helper()
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available")
	}
	path = filepath.Join(t.TempDir(), "id_ed25519")
	if out, err := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-C", "rwr-test", "-f", path).CombinedOutput(); err != nil {
		t.Skipf("ssh-keygen failed: %v: %s", err, out)
	}
	pem, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading generated key: %v", err)
	}
	return path, pem
}

func sshAuthFor(value string) (err error) {
	cfg := &types.InitConfig{}
	cfg.Variables.Flags.SSHKey = value
	_, err = getSSHAuthMethod(cfg)
	return err
}

// The regression this file exists for.
//
// setAsRWRSSHKey stores base64 of the private key under
// repository.ssh_private_key, and --ssh-key documents base64 as an accepted
// input. Nothing decoded it: base64 has no newlines, so it fell through to the
// file-path branch and rwr tried to open the blob as a filename, failing with
// "file name too long". set_as_rwr_ssh_key produced a value that could never
// authenticate anything.
func TestSSHAuthAcceptsBase64EncodedKey(t *testing.T) {
	t.Parallel()

	_, pem := generateKey(t)
	encoded := base64.StdEncoding.EncodeToString(pem)

	if err := sshAuthFor(encoded); err != nil {
		t.Fatalf("base64-encoded key rejected: %v", err)
	}
}

// The two forms that already worked must keep working.
func TestSSHAuthAcceptsRawPEMAndPath(t *testing.T) {
	t.Parallel()

	path, pem := generateKey(t)

	if err := sshAuthFor(string(pem)); err != nil {
		t.Errorf("raw PEM rejected: %v", err)
	}
	if err := sshAuthFor(path); err != nil {
		t.Errorf("key path rejected: %v", err)
	}
}

// A key file whose name begins with "ssh-" is a path, not key material.
//
// The classifier used to shortcut on that prefix, so "ssh-key" and
// "ssh-private.pem" were parsed as key data and failed with "ssh: no key
// found" while the same file's absolute path worked. The prefix only ever
// matches a public key, which this flow cannot authenticate with, so the
// shortcut could not enable any working case - it could only swallow paths.
func TestSSHAuthTreatsSSHPrefixedNameAsPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, pem := generateKey(t)

	for _, name := range []string{"ssh-key", "ssh-private.pem", "ssh-rsa"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, pem, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := sshAuthFor(path); err != nil {
			t.Errorf("key file named %q rejected: %v", name, err)
		}
	}
}

// A public key is not a credential this flow can use. It must not be mistaken
// for one, whether supplied directly or base64-encoded.
func TestSSHAuthRejectsAPublicKey(t *testing.T) {
	t.Parallel()

	const pub = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHhkAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA user@host"

	if err := sshAuthFor(pub); err == nil {
		t.Error("a public key was accepted as an authentication credential")
	}
	if err := sshAuthFor(base64.StdEncoding.EncodeToString([]byte(pub))); err == nil {
		t.Error("a base64-encoded public key was accepted as an authentication credential")
	}
}

// A path that happens to be valid base64 is still a path. "config" decodes
// cleanly under StdEncoding, so the decode alone cannot be the test - the
// decoded bytes have to look like a key.
func TestSSHAuthTreatsBase64LookingPathAsPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// "config" is 6 chars, a multiple of 3, so it round-trips as base64.
	name := base64.StdEncoding.EncodeToString([]byte("config"))
	path := filepath.Join(dir, name)

	_, pem := generateKey(t)
	if err := os.WriteFile(path, pem, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := sshAuthFor(path); err != nil {
		t.Fatalf("base64-looking path rejected: %v", err)
	}
}

// A value that is neither says so, rather than reporting a stat error about a
// filename the operator never wrote. The message must not carry the whole
// value, which may be the key itself.
func TestSSHAuthRejectsUnusableValueClearly(t *testing.T) {
	t.Parallel()

	junk := strings.Repeat("z", 400)

	err := sshAuthFor(junk)
	if err == nil {
		t.Fatal("expected an error for a value that is neither a key nor a path")
	}
	// The sentinel, not the message text: the condition is part of the
	// contract, the wording is not.
	if !errors.Is(err, ErrUnusableSSHKey) {
		t.Errorf("error is not ErrUnusableSSHKey: %v", err)
	}
	if strings.Contains(err.Error(), junk) {
		t.Error("the error repeats the whole value; it may be a private key")
	}
}

// wrapAt breaks a string into fixed-width lines, the way a hand-written or
// tool-generated base64 blob usually arrives.
func wrapAt(s string, width int, eol string) string {
	var b strings.Builder
	for i := 0; i < len(s); i += width {
		end := i + width
		if end > len(s) {
			end = len(s)
		}
		b.WriteString(s[i:end])
		b.WriteString(eol)
	}
	return b.String()
}

// Every base64 shape a hand-written --ssh-key value can arrive in.
//
// Column-wrapped base64 was the interesting one: it contains newlines, and the
// classifier used to treat a newline as proof of raw PEM, so it went to go-git
// as PEM and failed to parse. Go's decoder ignores embedded LF and CRLF, so it
// would have decoded perfectly well - the value was rejected by the
// classification, not by the encoding.
func TestSSHAuthAcceptsEveryBase64Shape(t *testing.T) {
	t.Parallel()

	_, pem := generateKey(t)
	std := base64.StdEncoding.EncodeToString(pem)

	cases := map[string]string{
		"unwrapped StdEncoding":   std,
		"wrapped at 64 with LF":   wrapAt(std, 64, "\n"),
		"wrapped at 64 with CRLF": wrapAt(std, 64, "\r\n"),
		"wrapped at 76 with LF":   wrapAt(std, 76, "\n"),
		"surrounded by space":     "  " + std + "\n",
	}

	for name, encoded := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := sshAuthFor(encoded); err != nil {
				t.Errorf("%s rejected: %v", name, err)
			}
		})
	}
}

// A PEM written on Windows has CRLF line endings, and is still the key.
func TestSSHAuthAcceptsCRLFPEM(t *testing.T) {
	t.Parallel()

	_, pem := generateKey(t)
	crlf := strings.ReplaceAll(string(pem), "\n", "\r\n")

	if err := sshAuthFor(crlf); err != nil {
		t.Errorf("CRLF PEM rejected: %v", err)
	}
}

// A newline is not proof of key material. A path that somehow carries one is
// still not a key, and must not be handed to the PEM parser.
func TestSSHAuthDoesNotTreatANewlineAsKeyMaterial(t *testing.T) {
	t.Parallel()

	err := sshAuthFor("not-a-key\nnot-a-key-either")
	if err == nil {
		t.Fatal("a newline-containing non-key was accepted")
	}
	if !errors.Is(err, ErrUnusableSSHKey) {
		t.Errorf("want ErrUnusableSSHKey, got: %v", err)
	}
}

// paddedKeyLikePayload is PEM-shaped bytes chosen so the four base64 alphabets
// actually disagree about it: its length is not a multiple of three (so the
// unpadded encodings differ from the padded ones) and it contains bytes that
// encode to "+" and "/" under the standard alphabet and to "-" and "_" under
// the URL-safe one.
//
// A real ed25519 key will not do: its PEM is 411 bytes, a multiple of three,
// with no bytes that diverge between the alphabets, so all four encodings
// produce the identical string and a table of them tests one case four times.
func paddedKeyLikePayload(t *testing.T) []byte {
	t.Helper()

	payload := append([]byte("-----BEGIN PRIVATE KEY-----\n"), 0xFF, 0xFE, 0xFD, 0xFB)
	payload = append(payload, []byte("\n-----END PRIVATE KEY-----")...)
	if len(payload)%3 == 0 {
		payload = append(payload, '\n')
	}

	// The precondition this fixture exists for. If it ever stops holding, the
	// table below silently stops testing anything.
	if base64.StdEncoding.EncodeToString(payload) == base64.RawStdEncoding.EncodeToString(payload) {
		t.Fatal("fixture no longer forces padding; the alphabets would not diverge")
	}
	if base64.StdEncoding.EncodeToString(payload) == base64.URLEncoding.EncodeToString(payload) {
		t.Fatal("fixture no longer contains bytes that differ between the standard and URL alphabets")
	}
	return payload
}

// The classifier accepts all four base64 alphabets, padded and unpadded,
// standard and URL-safe. Which one an operator's tooling emits is not
// something rwr gets to choose.
//
// This drives sshKeyMaterial directly rather than going through go-git,
// because the payload is PEM-shaped rather than a real key: the variation
// under test is in the classification, and a real key cannot express it (see
// paddedKeyLikePayload).
func TestSSHKeyMaterialAcceptsEveryBase64Alphabet(t *testing.T) {
	t.Parallel()

	payload := paddedKeyLikePayload(t)

	for name, encoding := range map[string]*base64.Encoding{
		"StdEncoding":    base64.StdEncoding,
		"RawStdEncoding": base64.RawStdEncoding,
		"URLEncoding":    base64.URLEncoding,
		"RawURLEncoding": base64.RawURLEncoding,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			decoded, ok := sshKeyMaterial(encoding.EncodeToString(payload))
			if !ok {
				t.Fatalf("%s-encoded key material was not recognised", name)
			}
			if string(decoded) != string(payload) {
				t.Errorf("%s round-trip changed the bytes", name)
			}
		})
	}
}

// Wrapping is orthogonal to the alphabet: a blob can arrive in either
// alphabet at any column width.
func TestSSHKeyMaterialAcceptsWrappedNonStandardAlphabets(t *testing.T) {
	t.Parallel()

	payload := paddedKeyLikePayload(t)
	encoded := base64.RawURLEncoding.EncodeToString(payload)

	for _, eol := range []string{"\n", "\r\n"} {
		decoded, ok := sshKeyMaterial(wrapAt(encoded, 8, eol))
		if !ok {
			t.Fatalf("wrapped RawURLEncoding not recognised (eol %q)", eol)
		}
		if string(decoded) != string(payload) {
			t.Errorf("wrapped RawURLEncoding round-trip changed the bytes (eol %q)", eol)
		}
	}
}
