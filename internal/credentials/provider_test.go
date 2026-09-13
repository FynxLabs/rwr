package credentials

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestProviderSetupOwnsSessionAndReusesReadiness(t *testing.T) {
	for _, status := range []string{"locked", "unauthenticated", "unlocked"} {
		t.Run(status, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			t.Setenv("LOCALAPPDATA", t.TempDir())
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			t.Setenv("BW_SESSION", "")
			bin := t.TempDir()
			if err := os.WriteFile(filepath.Join(bin, "bw"), []byte("#!/bin/sh\nexit 90\n"), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin)
			oldCommand, oldPrompt, oldTTY := providerCommand, providerPrompt, stdinIsTerminal
			oldLogin := providerLogin
			t.Cleanup(func() { providerLogin = oldLogin })
			t.Cleanup(func() { providerCommand, providerPrompt, stdinIsTerminal = oldCommand, oldPrompt, oldTTY })
			stdinIsTerminal = func() bool { return true }
			prompts, unlocks, logins := 0, 0, 0
			providerPrompt = func(string) (string, string, error) { prompts++; return "a@example.invalid", "master-secret", nil }
			providerCommand = func(_ context.Context, _ string, args []string, env map[string]string, _ io.Reader) ([]byte, error) {
				for _, arg := range args {
					if strings.Contains(arg, "master-secret") || strings.Contains(arg, "new-session") {
						t.Fatal("secret in argv")
					}
				}
				if args[0] != "login" && args[len(args)-1] != "--nointeraction" {
					t.Fatal("lookup can prompt")
				}
				switch args[0] {
				case "status":
					return []byte(`{"status":"` + status + `","userEmail":"a@example.invalid","serverUrl":"https://vault.bitwarden.com"}`), nil
				case "config":
					return nil, nil
				case "login":
					logins++
					if env["RWR_BW_PASSWORD"] != "master-secret" {
						t.Fatal("password absent")
					}
					return nil, nil
				case "unlock":
					unlocks++
					return []byte("new-session\n"), nil
				case "get":
					if status != "unlocked" && env["BW_SESSION"] != "new-session" {
						t.Fatal("session absent")
					}
					return []byte("secret-value\n"), nil
				}
				t.Fatalf("unexpected call %v", args)
				return nil, nil
			}
			providerLogin = providerCommand
			c := types.CredentialConnection{Name: "personal", Provider: "bitwarden", Account: "a@example.invalid"}
			s, err := OpenProvider(context.Background(), c, SetupOptions{Authenticate: true, Interactive: true})
			if err != nil {
				t.Fatal(err)
			}
			value, err := s.ReadSecret(context.Background(), types.CredentialReference{Item: "one"})
			if err != nil || value != "secret-value" {
				t.Fatalf("read: %q %v", value, err)
			}
			if os.Getenv("BW_SESSION") != "" || os.Getenv("RWR_BW_PASSWORD") != "" {
				t.Fatal("parent environment changed")
			}
			if status == "unlocked" {
				if prompts != 0 || unlocks != 0 || logins != 0 {
					t.Fatal("ready provider reauthenticated")
				}
			} else if prompts != 1 || unlocks != 1 {
				t.Fatal("authentication did not run once")
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			if len(s.(*bitwardenHandle).env) != 0 {
				t.Fatal("session retained secrets after close")
			}
		})
	}
}

func TestExcludedResolverDoesNotEvenConstructProvider(t *testing.T) {
	defer RegisterProvider("fixture", func() Provider { t.Fatal("excluded provider constructed"); return nil })()
	c := &types.InitConfig{CredentialProviders: []types.CredentialConnection{{Name: "one", Provider: "fixture"}}}
	c.Variables.Flags.Selection = &types.RunSelection{DenyProviders: true}
	r := NewResolver(c)
	defer r.Close()
	if _, err := r.Session(context.Background(), "one"); !IsUnavailable(err) {
		t.Fatalf("got %v", err)
	}
}

func TestProtectedOutputDoesNotLeakOnFailure(t *testing.T) {
	_, err := ProtectedCommand(context.Background(), "/bin/sh", []string{"-c", "printf secret-output; printf secret-diagnostic >&2; exit 1"}, map[string]string{"RWR_BW_PASSWORD": "master-secret"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, secret := range []string{"secret-output", "secret-diagnostic", "master-secret"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatal("secret leaked")
		}
	}
}

func TestProviderAccountConflictAndSkip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "bw"), []byte("#!/bin/sh\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	oldCommand, oldPrompt, oldTTY := providerCommand, providerPrompt, stdinIsTerminal
	defer func() { providerCommand, providerPrompt, stdinIsTerminal = oldCommand, oldPrompt, oldTTY }()
	stdinIsTerminal = func() bool { return true }
	providerCommand = func(context.Context, string, []string, map[string]string, io.Reader) ([]byte, error) {
		return []byte(`{"status":"locked","userEmail":"other@example.invalid","serverUrl":"https://vault.bitwarden.com"}`), nil
	}
	providerPrompt = func(string) (string, string, error) { t.Fatal("conflict prompted"); return "", "", nil }
	_, err := OpenProvider(context.Background(), types.CredentialConnection{Name: "one", Provider: "bitwarden", Account: "a@example.invalid"}, SetupOptions{Authenticate: true, Interactive: true})
	if err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("got %v", err)
	}
	providerPrompt = func(string) (string, string, error) { return "", "", &Unavailable{Skipped} }
	_, err = OpenProvider(context.Background(), types.CredentialConnection{Name: "one", Provider: "bitwarden"}, SetupOptions{Authenticate: true, Interactive: true})
	var unavailable *Unavailable
	if !errors.As(err, &unavailable) || unavailable.State != Skipped {
		t.Fatalf("skip lost: %v", err)
	}
}

func TestTerminalRedactorHandlesChunkBoundaries(t *testing.T) {
	for size := 1; size < 20; size++ {
		var output bytes.Buffer
		w := types.NewSecretWriter(&output, []string{"master-secret", "session"})
		text := "prefix master-secret and session suffix"
		for i := 0; i < len(text); i += size {
			end := min(i+size, len(text))
			if _, err := w.Write([]byte(text[i:end])); err != nil {
				t.Fatal(err)
			}
		}
		w.Flush()
		if output.String() != "prefix [redacted] and [redacted] suffix" {
			t.Fatalf("size=%d output=%q", size, output.String())
		}
	}
}
