package credentials

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/fynxlabs/rwr/internal/reporting"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// ProtectedCommand captures credential-bearing output outside ordinary logging.
// Errors identify the operation, never vendor stdout/stderr. Env is child-only.
// Input must contain no trailing protocol data beyond what the executable needs.
func ProtectedCommand(ctx context.Context, executable string, args []string, env map[string]string, input io.Reader) ([]byte, error) {
	if system.IsDryRun() {
		return nil, fmt.Errorf("credential command is prohibited during dry-run")
	}
	cmd := exec.CommandContext(ctx, executable, args...) // #nosec G204 -- fixed adapter executables, discrete argv, never shell orchestration
	cmd.Env = protectedEnv(env)
	cmd.Stdin = input
	var output bytes.Buffer
	cmd.Stdout = &output
	// Never expose raw diagnostic text: it can contain fetched values and tokens.
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("%s operation failed", executable)
	}
	return output.Bytes(), nil
}
func protectedEnv(overrides map[string]string) []string {
	var env []string
	for _, e := range os.Environ() {
		key, _, _ := strings.Cut(e, "=")
		if _, set := overrides[key]; set {
			continue
		}
		if types.IsCredentialEnvironmentKey(key) {
			continue
		}
		env = append(env, e)
	}
	for k, v := range overrides {
		env = append(env, k+"="+v)
	}
	return env
}

// protectedLogin gives the explicit login command the terminal for vendor MFA.
// Login must use --quiet; unlock's token-bearing output uses ProtectedCommand.
func protectedLogin(ctx context.Context, executable string, args []string, env map[string]string, _ io.Reader) ([]byte, error) {
	if system.IsDryRun() {
		return nil, fmt.Errorf("login prohibited during dry-run")
	}
	cmd := exec.CommandContext(ctx, executable, args...) // #nosec G204 -- fixed provider CLI and structured login arguments
	cmd.Env = protectedEnv(env)
	cmd.Stdin = os.Stdin
	secrets := []string{env["RWR_BW_PASSWORD"], env["BW_SESSION"]}
	stdout := types.NewSecretWriter(os.Stdout, secrets)
	stderr := types.NewSecretWriter(os.Stderr, secrets)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := reporting.WithTerminal(func() error { defer stdout.Flush(); defer stderr.Flush(); return cmd.Run() })
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, fmt.Errorf("credential login failed")
	}
	return nil, nil
}
