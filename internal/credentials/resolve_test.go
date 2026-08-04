package credentials

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/types"
)

// fakeKeyring stands in for the OS keyring: CI has no Secret Service and
// desktop keyring behavior is not something to assert against.
type fakeKeyring struct {
	entries map[string]string
	// unavailable simulates a headless machine with no keyring backend.
	unavailable bool
}

func (f *fakeKeyring) Get(name string) (string, error) {
	if f.unavailable {
		return "", errors.New("no keyring backend")
	}
	value, ok := f.entries[name]
	if !ok {
		return "", ErrKeyringNotFound
	}
	return value, nil
}

func (f *fakeKeyring) Set(name, value string) error {
	if f.unavailable {
		return errors.New("no keyring backend")
	}
	if f.entries == nil {
		f.entries = map[string]string{}
	}
	f.entries[name] = value
	return nil
}

// withFakes swaps the keyring and prompt seams for a test and restores them.
func withFakes(t *testing.T, ring Keyring, tty bool, prompt func(name, description string) (string, error)) {
	t.Helper()
	origRing, origTTY, origPrompt, origOffer := Ring, stdinIsTerminal, promptForCredential, offerKeyringSave
	Ring = ring
	stdinIsTerminal = func() bool { return tty }
	if prompt != nil {
		promptForCredential = prompt
	}
	offerKeyringSave = func(name, value string) {}
	t.Cleanup(func() {
		Ring, stdinIsTerminal, promptForCredential, offerKeyringSave = origRing, origTTY, origPrompt, origOffer
		types.RegisterCredentials(nil)
	})
}

func TestResolveSourcePrecedence(t *testing.T) {
	tests := []struct {
		name    string
		sources []string
		env     map[string]string
		keyring map[string]string
		want    string
	}{
		{
			name:    "env wins over keyring",
			sources: []string{"env:CACHIX_AUTH_TOKEN", "keyring"},
			env:     map[string]string{"CACHIX_AUTH_TOKEN": "from-env"},
			keyring: map[string]string{"cachix_token": "from-keyring"},
			want:    "from-env",
		},
		{
			name:    "keyring when env is empty",
			sources: []string{"env:CACHIX_AUTH_TOKEN", "keyring"},
			keyring: map[string]string{"cachix_token": "from-keyring"},
			want:    "from-keyring",
		},
		{
			name:    "declaration order is the precedence",
			sources: []string{"keyring", "env:CACHIX_AUTH_TOKEN"},
			env:     map[string]string{"CACHIX_AUTH_TOKEN": "from-env"},
			keyring: map[string]string{"cachix_token": "from-keyring"},
			want:    "from-keyring",
		},
		{
			name:    "default order reads env:RWR_CRED_<NAME> first",
			sources: nil,
			env:     map[string]string{"RWR_CRED_CACHIX_TOKEN": "from-default-env"},
			keyring: map[string]string{"cachix_token": "from-keyring"},
			want:    "from-default-env",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withFakes(t, &fakeKeyring{entries: tt.keyring}, false, nil)
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			spec := types.CredentialSpec{Name: "cachix_token", Sources: tt.sources}
			if err := Resolve([]types.CredentialSpec{spec}, Options{}); err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if got, _ := types.CredentialValue("cachix_token"); got != tt.want {
				t.Errorf("resolved %q, want %q", got, tt.want)
			}
		})
	}
}

// A declared credential that resolves nowhere fails the run up front, naming
// the credential and the sources tried - and in a non-TTY run the error says
// prompt was skipped rather than silently hanging on an invisible form.
func TestResolveUnresolvableNamesSourcesTried(t *testing.T) {
	withFakes(t, &fakeKeyring{}, false, nil)

	spec := types.CredentialSpec{
		Name:    "cachix_token",
		Sources: []string{"env:CACHIX_AUTH_TOKEN", "keyring", "prompt"},
	}
	err := Resolve([]types.CredentialSpec{spec}, Options{Interactive: true})
	if err == nil {
		t.Fatal("Resolve = nil, want an up-front error")
	}
	for _, want := range []string{`"cachix_token"`, "env:CACHIX_AUTH_TOKEN", "keyring", "prompt skipped: non-interactive run"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %q", err, want)
		}
	}
}

// --interactive=false skips prompt even on a TTY.
func TestResolveInteractiveFalseSkipsPrompt(t *testing.T) {
	prompted := false
	withFakes(t, &fakeKeyring{}, true, func(name, description string) (string, error) {
		prompted = true
		return "never", nil
	})

	spec := types.CredentialSpec{Name: "cachix_token", Sources: []string{"prompt"}}
	err := Resolve([]types.CredentialSpec{spec}, Options{Interactive: false})
	if prompted {
		t.Error("prompt ran despite --interactive=false")
	}
	if err == nil || !strings.Contains(err.Error(), "prompt skipped") {
		t.Errorf("error = %v, want it to say prompt was skipped", err)
	}
}

// On a TTY the prompt is the last resort, and its answer resolves the credential.
func TestResolvePromptsOnTTY(t *testing.T) {
	withFakes(t, &fakeKeyring{}, true, func(name, description string) (string, error) {
		if name != "cachix_token" || description != "Cachix auth token" {
			t.Errorf("prompt got (%q, %q)", name, description)
		}
		return "typed-in", nil
	})

	spec := types.CredentialSpec{Name: "cachix_token", Description: "Cachix auth token", Sources: []string{"keyring", "prompt"}}
	if err := Resolve([]types.CredentialSpec{spec}, Options{Interactive: true}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got, _ := types.CredentialValue("cachix_token"); got != "typed-in" {
		t.Errorf("resolved %q, want the prompted value", got)
	}
}

// The keyring is never load-bearing on read: an unavailable backend moves
// precedence along instead of erroring. An explicit save is the opposite.
func TestKeyringUnavailable(t *testing.T) {
	withFakes(t, &fakeKeyring{unavailable: true}, false, nil)
	t.Setenv("CACHIX_AUTH_TOKEN", "from-env")

	spec := types.CredentialSpec{Name: "cachix_token", Sources: []string{"keyring", "env:CACHIX_AUTH_TOKEN"}}
	if err := Resolve([]types.CredentialSpec{spec}, Options{}); err != nil {
		t.Fatalf("Resolve with an unavailable keyring: %v", err)
	}
	if got, _ := types.CredentialValue("cachix_token"); got != "from-env" {
		t.Errorf("resolved %q, want the env value", got)
	}

	if err := SaveToKeyring("cachix_token", "value"); err == nil {
		t.Error("SaveToKeyring with no backend = nil, want a loud error")
	}
}

// A value saved to the keyring is what the next resolution reads back.
func TestKeyringRoundtrip(t *testing.T) {
	withFakes(t, &fakeKeyring{}, false, nil)

	if err := SaveToKeyring("cachix_token", "stored-once"); err != nil {
		t.Fatalf("SaveToKeyring: %v", err)
	}
	spec := types.CredentialSpec{Name: "cachix_token", Sources: []string{"keyring"}}
	if err := Resolve([]types.CredentialSpec{spec}, Options{}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got, _ := types.CredentialValue("cachix_token"); got != "stored-once" {
		t.Errorf("resolved %q, want the saved value", got)
	}
}

// A credential scoped to processors outside the run is not resolved - no point
// failing (or prompting) for a value nothing will read.
func TestResolveSkipsOutOfScopeCredentials(t *testing.T) {
	withFakes(t, &fakeKeyring{}, false, nil)

	specs := []types.CredentialSpec{
		{Name: "ssh_token", Sources: []string{"env:NOWHERE"}, Scope: []string{types.BlueprintTypeSSHKeys}},
	}
	if err := Resolve(specs, Options{Selected: []string{types.BlueprintTypePackages}}); err != nil {
		t.Fatalf("out-of-scope credential was resolved (and failed): %v", err)
	}
	if _, ok := types.CredentialValue("ssh_token"); ok {
		t.Error("out-of-scope credential has a value")
	}

	// The same credential resolves (here: fails loudly) once its processor is in
	// the run, and the templates surface maps to the files processor.
	if err := Resolve(specs, Options{Selected: []string{types.BlueprintTypeSSHKeys}}); err == nil {
		t.Error("in-scope credential was skipped")
	}
	templScoped := []types.CredentialSpec{
		{Name: "tmpl_token", Sources: []string{"env:NOWHERE"}, Scope: []string{types.CredentialSurfaceTemplates}},
	}
	if err := Resolve(templScoped, Options{Selected: []string{types.BlueprintTypeFiles}}); err == nil {
		t.Error("templates-scoped credential was skipped in a files run")
	}
}

// Resolution logs values behind the redaction placeholder; the value itself
// must never reach the log.
func TestResolveRedactsInDebugLogs(t *testing.T) {
	withFakes(t, &fakeKeyring{}, false, nil)
	t.Setenv("CACHIX_AUTH_TOKEN", "cachix-super-secret")

	var buf bytes.Buffer
	log.SetOutput(&buf)
	origLevel := log.GetLevel()
	log.SetLevel(log.DebugLevel)
	t.Cleanup(func() {
		log.SetOutput(nil)
		log.SetLevel(origLevel)
	})

	spec := types.CredentialSpec{Name: "cachix_token", Sources: []string{"env:CACHIX_AUTH_TOKEN"}}
	if err := Resolve([]types.CredentialSpec{spec}, Options{}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "cachix-super-secret") {
		t.Errorf("debug log leaked the credential value: %s", out)
	}
	if !strings.Contains(out, types.RedactedPlaceholder) {
		t.Errorf("debug log shows no redaction placeholder: %s", out)
	}
}

// The built-ins resolve through their pre-existing sources, keyring last, and a
// keyring token flows into the flag field existing consumers read. They are
// optional: nothing configured is not an error.
func TestResolveBuiltins(t *testing.T) {
	t.Run("flag wins", func(t *testing.T) {
		withFakes(t, &fakeKeyring{entries: map[string]string{"gh_api_token": "from-keyring"}}, false, nil)
		flags := types.Flags{GHAPIToken: "from-flag"}
		ResolveBuiltins(&flags)
		if got, _ := types.CredentialValue("gh_api_token"); got != "from-flag" {
			t.Errorf("resolved %q, want the flag/config value", got)
		}
	})

	t.Run("GITHUB_TOKEN beats keyring", func(t *testing.T) {
		withFakes(t, &fakeKeyring{entries: map[string]string{"gh_api_token": "from-keyring"}}, false, nil)
		t.Setenv("GITHUB_TOKEN", "from-env")
		flags := types.Flags{}
		ResolveBuiltins(&flags)
		if got, _ := types.CredentialValue("gh_api_token"); got != "from-env" {
			t.Errorf("resolved %q, want the env value", got)
		}
	})

	t.Run("keyring fills the flag field", func(t *testing.T) {
		withFakes(t, &fakeKeyring{entries: map[string]string{"gh_api_token": "from-keyring"}}, false, nil)
		t.Setenv("GITHUB_TOKEN", "")
		flags := types.Flags{}
		ResolveBuiltins(&flags)
		if flags.GHAPIToken != "from-keyring" {
			t.Errorf("flags.GHAPIToken = %q, want the keyring token", flags.GHAPIToken)
		}
	})

	t.Run("nothing configured is not an error", func(t *testing.T) {
		withFakes(t, &fakeKeyring{}, false, nil)
		t.Setenv("GITHUB_TOKEN", "")
		flags := types.Flags{}
		ResolveBuiltins(&flags)
		if _, ok := types.CredentialValue("gh_api_token"); ok {
			t.Error("gh_api_token has a value from nowhere")
		}
	})
}
