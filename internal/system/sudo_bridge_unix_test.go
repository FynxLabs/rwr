//go:build !windows

package system

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/freehold-digital/rwr/internal/reporting"
	"github.com/freehold-digital/rwr/internal/types"
)

type bridgeReporter struct{ requests chan reporting.SecretReq }

func (bridgeReporter) SupportsInlinePrompts() bool { return true }
func (r bridgeReporter) Emit(e reporting.Event) {
	if req, ok := e.(reporting.SecretReq); ok {
		r.requests <- req
	}
}

func TestNestedSudoBridgeReusesPasswordAndCleansUp(t *testing.T) {
	defer reporting.Set(bridgeReporter{})()
	clearSudoPassword()
	t.Cleanup(clearSudoPassword)
	prompts := 0
	defer SetSudoPasswordForTest(func() ([]byte, error) { prompts++; return []byte("fixture-secret"), nil })()
	defer SetSudoValidateForTest(func(context.Context, []byte) error { return nil })()
	dir := t.TempDir()
	sudo := filepath.Join(dir, "sudo")
	// Each invocation has its own sudo PID. The private helper must work when
	// re-entering the test executable, just as it does with the installed CLI.
	if err := os.WriteFile(sudo, []byte("#!/bin/sh\n[ \"$1\" = -A ] || exit 21\nvalue=$(\"$SUDO_ASKPASS\") || exit 22\n[ \"$value\" = fixture-secret ] || exit 23\nprintf 'authenticated\\n'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	command := exec.Command("sh", "-c", "sudo; sudo")
	command.Env = os.Environ()
	stop, err := installSudoBridge(command, types.Command{Escalates: true})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	endpoint := environmentValue(command.Env, askpassSocketEnv)
	for _, path := range []string{filepath.Dir(endpoint), endpoint} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		want := os.FileMode(0600)
		if info.IsDir() {
			want = 0700
		}
		if info.Mode().Perm() != want {
			t.Fatalf("permissions %s: %v", path, info.Mode())
		}
	}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("nested sudo: %v: %s", err, output)
	}
	if string(output) != "authenticated\nauthenticated\n" || prompts != 1 {
		t.Fatalf("output=%q prompts=%d", output, prompts)
	}
	for _, entry := range command.Env {
		if strings.Contains(entry, "fixture-secret") {
			t.Fatal("password leaked into environment")
		}
	}
	stop()
	if _, err := os.Stat(filepath.Dir(endpoint)); !os.IsNotExist(err) {
		t.Fatalf("bridge directory survived: %v", err)
	}
}

func TestSudoBridgeCleanupCancelsPendingPrompt(t *testing.T) {
	r := bridgeReporter{requests: make(chan reporting.SecretReq, 1)}
	defer reporting.Set(r)()
	clearSudoPassword()
	t.Cleanup(clearSudoPassword)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sudo"), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	command := exec.Command("sh")
	command.Env = os.Environ()
	stop, err := installSudoBridge(command, types.Command{Escalates: true})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	helper := exec.Command(environmentValue(command.Env, "SUDO_ASKPASS"))
	helper.Env = command.Env
	done := make(chan error, 1)
	go func() { done <- helper.Run() }()
	select {
	case <-r.requests:
	case <-time.After(5 * time.Second):
		t.Fatal("no managed prompt")
	}
	cleaned := make(chan struct{})
	go func() { stop(); close(cleaned) }()
	select {
	case <-cleaned:
	case <-time.After(5 * time.Second):
		t.Fatal("cleanup blocked on pending prompt")
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled helper succeeded")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("helper survived cleanup")
	}
}

func TestSudoAuthenticationRetriesAreBounded(t *testing.T) {
	clearSudoPassword()
	t.Cleanup(clearSudoPassword)
	attempts := 0
	defer SetSudoPasswordForTest(func() ([]byte, error) { attempts++; return []byte("wrong"), nil })()
	denied := errors.New("authentication failed")
	defer SetSudoValidateForTest(func(context.Context, []byte) error { return denied })()
	if _, err := validatedSudoPassword(); !errors.Is(err, denied) {
		t.Fatalf("error=%v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts=%d", attempts)
	}
}

func TestExplicitInteractiveCommandDoesNotInstallAskpass(t *testing.T) {
	defer reporting.Set(bridgeReporter{})()
	command := exec.Command("sh")
	command.Env = os.Environ()
	stop, err := installSudoBridge(command, types.Command{Interactive: true, Escalates: true})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	if environmentValue(command.Env, askpassSocketEnv) != "" {
		t.Fatal("interactive terminal handoff must not wait for a suspended inline display")
	}
}

func TestSudoAuthenticationRecoversAfterRejectedPassword(t *testing.T) {
	clearSudoPassword()
	t.Cleanup(clearSudoPassword)
	prompts := 0
	defer SetSudoPasswordForTest(func() ([]byte, error) {
		prompts++
		if prompts == 1 {
			return []byte("wrong"), nil
		}
		return []byte("correct"), nil
	})()
	defer SetSudoValidateForTest(func(_ context.Context, input []byte) error {
		if string(input) != "correct\n" {
			return errors.New("denied")
		}
		return nil
	})()
	password, err := validatedSudoPassword()
	defer zeroBytes(password)
	if err != nil || string(password) != "correct" || prompts != 2 {
		t.Fatalf("error=%v attempts=%d", err, prompts)
	}
}
