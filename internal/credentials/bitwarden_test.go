package credentials

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

// withBWFake swaps the CLI seam for a fake and restores it. The fake keys off
// the full `bw:<item>[/<key>]` source so a test can express both the values it
// returns and the failures it simulates.
func withBWFake(t *testing.T, responses map[string]string, errs map[string]error) {
	t.Helper()
	orig := bwFetch
	bwFetch = func(spec bwSource) (string, error) {
		key := "bw:" + spec.Item
		if spec.Key != "password" {
			key += "/" + spec.Key
			if spec.Field != "" {
				key += ":" + spec.Field
			}
		}
		if err, ok := errs[key]; ok {
			return "", err
		}
		if value, ok := responses[key]; ok {
			return value, nil
		}
		return "", errors.New("Not found. ") // bw's item-not-found text
	}
	t.Cleanup(func() { bwFetch = orig })
}

func TestFromBitwardenKeys(t *testing.T) {
	withBWFake(t, map[string]string{
		"bw:github":               "hunter2",
		"bw:github/username":      "octocat",
		"bw:signing/notes":        "line one\nline two\n",
		"bw:signing/totp":         "123456",
		"bw:github/field:api-key": "tok_123",
	}, nil)

	for source, want := range map[string]string{
		"bw:github":               "hunter2",
		"bw:github/username":      "octocat",
		"bw:signing/notes":        "line one\nline two\n",
		"bw:signing/totp":         "123456",
		"bw:github/field:api-key": "tok_123",
	} {
		got, ok := FromBitwarden(source)
		if !ok {
			t.Errorf("FromBitwarden(%q) = not found, want %q", source, want)
			continue
		}
		if got != want {
			t.Errorf("FromBitwarden(%q) = %q, want %q", source, got, want)
		}
	}
}

// A source that cannot yield a value is a miss, not a failure: precedence
// moves to the next declared source. That is what lets a locked-vault laptop
// still resolve the credential from the keyring or a prompt.
func TestFromBitwardenMissFallsThrough(t *testing.T) {
	withBWFake(t, nil, map[string]error{
		"bw:locked-item": errors.New("Vault is locked. Unlock your vault."),
	})

	if value, ok := FromBitwarden("bw:locked-item"); ok {
		t.Errorf("FromBitwarden returned %q, want no value", value)
	}

	withFakes(t, &fakeKeyring{entries: map[string]string{"signing_passphrase": "from-keyring"}}, false, nil)
	spec := types.CredentialSpec{Name: "signing_passphrase", Sources: []string{"bw:locked-item", "keyring"}}
	if err := Resolve([]types.CredentialSpec{spec}, Options{}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got, _ := types.CredentialValue("signing_passphrase"); got != "from-keyring" {
		t.Errorf("resolved %q, want the keyring value", got)
	}
}

func TestFromBitwardenMalformedSources(t *testing.T) {
	withBWFake(t, nil, nil)
	for _, source := range []string{"bw:", "bw:/password"} {
		if _, ok := FromBitwarden(source); ok {
			t.Errorf("FromBitwarden(%q) resolved, want a miss", source)
		}
	}
}

// fakeBW installs a `bw` stand-in on PATH: it appends its argv to a log and
// answers from canned response files. `<key>.err` files go to stderr with exit
// 1, which is how a test simulates a failure. This exercises the real exec
// path - argv building, stdout/stderr handling, the exit path - rather than
// the seam.
func fakeBW(t *testing.T, dir string, answers map[string]string) {
	t.Helper()
	script := `#!/bin/sh
echo "$@" >> ` + filepath.Join(dir, "argv.log") + `
RESP=` + filepath.Join(dir, "responses") + `
key=$(echo "$@" | tr ' /' '__')
if [ -f "$RESP/$key.err" ]; then
  cat "$RESP/$key.err" >&2
  exit 1
fi
if [ -f "$RESP/$key" ]; then
  cat "$RESP/$key"
  exit 0
fi
echo "Not found." >&2
exit 1
`
	if err := os.WriteFile(filepath.Join(dir, "bw"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	responses := filepath.Join(dir, "responses")
	if err := os.MkdirAll(responses, 0o700); err != nil {
		t.Fatal(err)
	}
	for key, value := range answers {
		name := strings.NewReplacer(" ", "_", "/", "_").Replace(key)
		if err := os.WriteFile(filepath.Join(responses, name), []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// The seam-free test runs the real exec path against a scripted `bw`, so argv
// construction and newline handling are what the tests assert on. It needs a
// POSIX shell, so it is skipped on Windows.
func TestFetchWithCLIFakeBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs a POSIX shell to fake the bw binary")
	}
	dir := t.TempDir()
	fakeBW(t, dir, map[string]string{
		"get password github": "hunter2\n",
		"get username github": "octocat\n",
		"get item signing":    `{"fields":[{"name":"api-key","type":1,"value":"tok_123"},{"name":"other","type":0,"value":"nope"}]}`,
		"get password owner":  "token\n\n",
		"get item ownfield":   `{"fields":[{"name":"api-key","value":"tok_123\n"}]}`,
	})
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if got, err := fetchWithCLI(bwSource{Item: "github", Key: "password"}); err != nil || got != "hunter2" {
		t.Errorf("password = %q, %v; want hunter2 with no error", got, err)
	}
	if got, err := fetchWithCLI(bwSource{Item: "github", Key: "username"}); err != nil || got != "octocat" {
		t.Errorf("username = %q, %v; want octocat with no error", got, err)
	}
	got, err := fetchWithCLI(bwSource{Item: "signing", Key: "field", Field: "api-key"})
	if err != nil || got != "tok_123" {
		t.Errorf("field = %q, %v; want tok_123 with no error", got, err)
	}
	if _, err := fetchWithCLI(bwSource{Item: "signing", Key: "field", Field: "missing"}); err == nil || !strings.Contains(err.Error(), `no custom field named "missing"`) {
		t.Errorf("missing field error = %v, want an error naming the field", err)
	}

	// Newlines the value itself owns survive: bw appends exactly one
	// formatting newline, and a JSON field value has none to remove. A
	// TrimRight here once turned "token\n" into "token".
	if got, err := fetchWithCLI(bwSource{Item: "owner", Key: "password"}); err != nil || got != "token\n" {
		t.Errorf("password owning a trailing newline = %q, %v; want %q with no error", got, err, "token\n")
	}
	if got, err := fetchWithCLI(bwSource{Item: "ownfield", Key: "field", Field: "api-key"}); err != nil || got != "tok_123\n" {
		t.Errorf("field value owning a trailing newline = %q, %v; want %q with no error", got, err, "tok_123\n")
	}

	argv, readErr := os.ReadFile(filepath.Join(dir, "argv.log"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	for _, want := range []string{"get password github", "get item signing"} {
		if !strings.Contains(string(argv), want) {
			t.Errorf("bw was invoked as [%s], want a call containing %q", argv, want)
		}
	}
}

// A locked vault is the failure class an operator can fix without rwr's help,
// so the error carries the BW_SESSION action.
func TestFetchWithCLILockedVaultHint(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs a POSIX shell to fake the bw binary")
	}
	dir := t.TempDir()
	fakeBW(t, dir, nil)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	responses := filepath.Join(dir, "responses")
	if err := os.WriteFile(filepath.Join(responses, "get_password_github.err"), []byte("Vault is locked. Unlock your vault.\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := fetchWithCLI(bwSource{Item: "github", Key: "password"})
	if err == nil || !strings.Contains(err.Error(), "BW_SESSION") {
		t.Errorf("locked-vault error = %v, want the BW_SESSION hint", err)
	}
}

// parse and validate must agree on the source vocabulary, or a source the init
// file accepts fails at resolve time (or the reverse: a valid source rejected
// at decode). Table entries are sources validateCredentialSource passes, with
// the item/key each must parse into.
func TestParseMatchesValidation(t *testing.T) {
	tests := []struct {
		source string
		want   bwSource
	}{
		{source: "bw:github", want: bwSource{Item: "github", Key: "password"}},
		{source: "bw:github/password", want: bwSource{Item: "github", Key: "password"}},
		{source: "bw:github/username", want: bwSource{Item: "github", Key: "username"}},
		{source: "bw:github/uri", want: bwSource{Item: "github", Key: "uri"}},
		{source: "bw:github/notes", want: bwSource{Item: "github", Key: "notes"}},
		{source: "bw:github/totp", want: bwSource{Item: "github", Key: "totp"}},
		{source: "bw:github/field:api-key", want: bwSource{Item: "github", Key: "field", Field: "api-key"}},
		// Item names may contain slashes; only a final key word is the key.
		{source: "bw:my/team", want: bwSource{Item: "my/team", Key: "password"}},
		{source: "bw:my/team/password", want: bwSource{Item: "my/team", Key: "password"}},
		{source: "bw:my/team/username", want: bwSource{Item: "my/team", Key: "username"}},
		// bw matches items by name; spaces are real.
		{source: "bw:my item/password", want: bwSource{Item: "my item", Key: "password"}},
	}
	for _, tt := range tests {
		if err := types.ValidateCredentialSpecs([]types.CredentialSpec{
			{Name: "probe", Sources: []string{tt.source}},
		}); err != nil {
			t.Errorf("ValidateCredentialSpecs(%q) = %v, want nil", tt.source, err)
			continue
		}
		got, err := parseBitwardenSource(tt.source)
		if err != nil {
			t.Errorf("parseBitwardenSource(%q) = %v, want nil", tt.source, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseBitwardenSource(%q) = %+v, want %+v", tt.source, got, tt.want)
		}
	}

	rejected := []string{
		"bw:",
		"bw:/password",
		"bw:github/field:",
	}
	for _, source := range rejected {
		if err := types.ValidateCredentialSpecs([]types.CredentialSpec{
			{Name: "probe", Sources: []string{source}},
		}); err == nil {
			t.Errorf("ValidateCredentialSpecs(%q) = nil, want an error", source)
		}
		if _, err := parseBitwardenSource(source); err == nil {
			t.Errorf("parseBitwardenSource(%q) = nil, want an error", source)
		}
	}
}
