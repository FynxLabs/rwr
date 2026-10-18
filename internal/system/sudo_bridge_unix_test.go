//go:build !windows

package system

import (
	"bytes"
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
	bridge, err := installSudoBridge(command, types.Command{Escalates: true})
	if err != nil {
		t.Fatal(err)
	}
	defer bridge.Close()
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
	var captured bytes.Buffer
	command.Stdout = &captured
	command.Stderr = &captured
	err = bridge.Run(command)
	output := captured.Bytes()
	if err != nil {
		t.Fatalf("nested sudo: %v: %s", err, output)
	}
	if string(output) != "authenticated\nauthenticated\n" || prompts != 1 {
		t.Fatalf("output=%q prompts=%d", output, prompts)
	}
	for _, entry := range command.Env {
		if strings.HasPrefix(entry, "RWR_INTERNAL_ASKPASS_TOKEN=") {
			t.Fatal("capability token in environment")
		}
		if strings.Contains(entry, "fixture-secret") {
			t.Fatal("password leaked into environment")
		}
	}
	bridge.Close()
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
	command := exec.Command("sh", "-c", "exec \"$SUDO_ASKPASS\"")
	command.Env = os.Environ()
	bridge, err := installSudoBridge(command, types.Command{Escalates: true})
	if err != nil {
		t.Fatal(err)
	}
	defer bridge.Close()
	done := make(chan error, 1)
	go func() { done <- bridge.Run(command) }()
	select {
	case <-r.requests:
	case <-time.After(5 * time.Second):
		t.Fatal("no managed prompt")
	}
	cleaned := make(chan struct{})
	go func() { bridge.Close(); close(cleaned) }()
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

func TestExplicitInteractiveCommandInstallsManagedAskpass(t *testing.T) {
	defer reporting.Set(bridgeReporter{})()
	command := exec.Command("sh")
	command.Env = os.Environ()
	bridge, err := installSudoBridge(command, types.Command{Interactive: true, Escalates: true})
	if err != nil {
		t.Fatal(err)
	}
	defer bridge.Close()
	if environmentValue(command.Env, askpassSocketEnv) == "" {
		t.Fatal("interactive popup needs askpass when its new TTY cannot reuse ambient sudo authorization")
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

// Knowing the socket location and helper path must not authorize a sibling
// process belonging to RWR's own session (or any other command session).
func TestSudoBridgeRejectsHelperOutsideCommandSession(t *testing.T) {
	defer reporting.Set(bridgeReporter{})()
	prompts := 0
	defer SetSudoPasswordForTest(func() ([]byte, error) { prompts++; return []byte("fixture-secret"), nil })()
	defer SetSudoValidateForTest(func(context.Context, []byte) error { return nil })()
	clearSudoPassword()
	t.Cleanup(clearSudoPassword)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sudo"), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	command := exec.Command("sleep", "30")
	command.Env = os.Environ()
	bridge, err := installSudoBridge(command, types.Command{Escalates: true})
	if err != nil {
		t.Fatal(err)
	}
	defer bridge.Close()
	done := make(chan error, 1)
	go func() { done <- bridge.Run(command) }()
	select {
	case <-bridge.ready:
	case err := <-done:
		t.Fatalf("child exited before ready: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("child never started")
	}
	defer func() { _ = command.Process.Kill(); <-done }()
	outsider := exec.Command(environmentValue(command.Env, "SUDO_ASKPASS"))
	outsider.Env = command.Env
	output, err := outsider.Output()
	if err == nil || len(output) != 0 {
		t.Fatalf("foreign helper accepted: error=%v output bytes=%d", err, len(output))
	}
	if prompts != 0 {
		t.Fatalf("unauthorized helper triggered %d password prompts", prompts)
	}
}

func TestSudoPasswordContextCancellation(t *testing.T) {
	for _, cancelRun := range []bool{false, true} {
		t.Run(map[bool]string{false: "command scope", true: "whole run"}[cancelRun], func(t *testing.T) {
			defer BeginRun()()
			ctx, cancel := context.WithCancel(RunContext())
			defer cancel()
			defer SetSudoPasswordForTest(func() ([]byte, error) {
				if cancelRun {
					Cancel()
				} else {
					cancel()
				}
				return nil, context.Canceled
			})()
			_, err := validatedSudoPasswordContext(ctx)
			want := error(context.Canceled)
			if cancelRun {
				want = ErrCancelled
			}
			if !errors.Is(err, want) {
				t.Fatalf("error=%v want %v", err, want)
			}
		})
	}
}
