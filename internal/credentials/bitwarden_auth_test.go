package credentials

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestBitwardenAuthenticatesOnceAndResumesCredentials(t *testing.T) {
	for _, status := range []string{"locked", "unauthenticated"} {
		t.Run(status, func(t *testing.T) {
			withFakes(t, &fakeKeyring{}, true, func(_, _ string) (string, error) { t.Fatal("unexpected per-secret fallback"); return "", nil })
			oldFetch, oldStatus, oldPrompt, oldAuth := bwFetch, bwAuthStatus, promptBitwardenAuth, bwAuthenticate
			t.Cleanup(func() {
				bwFetch, bwAuthStatus, promptBitwardenAuth, bwAuthenticate = oldFetch, oldStatus, oldPrompt, oldAuth
			})
			bwAuthStatus = func() (string, error) {
				return `{"status":"` + status + `","serverUrl":"https://vault.bitwarden.com"}`, nil
			}
			prompts := 0
			promptBitwardenAuth = func(login bool, server string) (string, string, string, error) {
				prompts++
				if login != (status == "unauthenticated") {
					t.Fatal("wrong auth flow")
				}
				return "test@example.invalid", "master-secret", server, nil
			}
			var calls []string
			bwAuthenticate = func(args []string, password string, interactive bool) (string, error) {
				if password != "master-secret" {
					t.Fatal("password missing")
				}
				for _, arg := range args {
					if arg == password {
						t.Fatal("password leaked to argv")
					}
				}
				calls = append(calls, args[0])
				if args[0] == "unlock" {
					return "session-value\n", nil
				}
				return "", nil
			}
			bwFetch = func(bwSource) (string, error) {
				if bitwardenSession == "session-value" {
					return "vault-value", nil
				}
				return "", errors.New("vault is locked")
			}
			specs := []types.CredentialSpec{{Name: "one", Sources: []string{"bw:one"}}, {Name: "two", Sources: []string{"bw:two"}}}
			types.RegisterCredentials(specs)
			if err := Resolve(specs, Options{Interactive: true}); err != nil {
				t.Fatal(err)
			}
			if prompts != 1 {
				t.Fatalf("prompted %d times", prompts)
			}
			want := []string{"unlock"}
			if status == "unauthenticated" {
				want = []string{"login", "unlock"}
			}
			if !reflect.DeepEqual(calls, want) {
				t.Fatalf("commands: %v", calls)
			}
			if bitwardenSession != "" {
				t.Fatal("session survived resolution")
			}
			for _, name := range []string{"one", "two"} {
				if value, _ := types.CredentialValue(name); value != "vault-value" {
					t.Fatal("credential unresolved")
				}
			}
		})
	}
}

func TestBitwardenAuthPasswordIsChildOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixture")
	}
	t.Setenv("RWR_BW_PASSWORD", "parent-value")
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	script := "#!/bin/sh\n[ \"$RWR_BW_PASSWORD\" = 'master-secret' ] || exit 1\nfor arg do [ \"$arg\" != 'master-secret' ] || exit 2; done\nprintf session-value\n"
	if err := os.WriteFile(filepath.Join(dir, "bw"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	value, err := bwAuthenticate([]string{"unlock", "--passwordenv", "RWR_BW_PASSWORD", "--raw"}, "master-secret", false)
	if err != nil || value != "session-value" {
		t.Fatalf("unlock: %q %v", value, err)
	}
	if os.Getenv("RWR_BW_PASSWORD") != "parent-value" {
		t.Fatal("password escaped child environment")
	}
}

func TestBitwardenSessionRequiresExplicitExposure(t *testing.T) {
	types.RegisterCredentials(nil)
	types.SetExposedCredentials(nil)
	t.Cleanup(func() { types.RegisterCredentials(nil); types.SetExposedCredentials(nil) })
	types.SetCredentialValue("bw_session", "session-value")
	if len(types.ExportedCredentialEnv()) != 0 {
		t.Fatal("vault session exposed by default")
	}
	types.SetExposedCredentials([]string{"bw_session"})
	if types.ExportedCredentialEnv()["RWR_CRED_BW_SESSION"] != "session-value" {
		t.Fatal("explicit session exposure failed")
	}
}
