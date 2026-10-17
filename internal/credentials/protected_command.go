package credentials

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/freehold-digital/rwr/internal/reporting"
	"github.com/freehold-digital/rwr/internal/system"
	"github.com/freehold-digital/rwr/internal/types"
)

// ProtectedCommand captures credential-bearing output outside ordinary logging.
// Errors identify the operation, never vendor stdout/stderr. Env is child-only.
// Input must contain no trailing protocol data beyond what the executable needs.
func ProtectedCommand(ctx context.Context, executable string, args []string, env map[string]string, input io.Reader) ([]byte, error) {
	return protectedCommand(ctx, executable, args, env, input, nil)
}

// Diagnostics are opt-in for commands whose credential inputs are known. Vault
// reads can print new, unknown secrets on stderr and must keep it suppressed.
func protectedCommand(ctx context.Context, executable string, args []string, env map[string]string, input io.Reader, diagnosticSecrets []string) ([]byte, error) {
	if system.IsDryRun() {
		return nil, fmt.Errorf("credential command is prohibited during dry-run")
	}
	cmd := exec.CommandContext(ctx, executable, args...) // #nosec G204 -- fixed adapter executables, discrete argv, never shell orchestration
	cmd.Env = protectedEnv(env)
	cmd.Stdin = input
	var output bytes.Buffer
	cmd.Stdout = &output
	var diagnostic bytes.Buffer
	cmd.Stderr = io.Discard
	if diagnosticSecrets != nil {
		cmd.Stderr = &diagnostic
	}
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if diagnosticSecrets != nil {
			secrets := append(types.ResolvedSecrets(), diagnosticSecrets...)
			for key, value := range env {
				if types.IsCredentialEnvironmentKey(key) {
					secrets = append(secrets, value)
				}
			}
			// Failed commands can emit credential-bearing stdout as well.
			secrets = append(secrets, strings.Split(output.String(), "\n")...)
			var redacted bytes.Buffer
			writer := types.NewSecretWriter(&redacted, secrets)
			_, _ = fmt.Fprintf(writer, "%v: %s", err, diagnostic.String()) //nolint:errcheck // bytes.Buffer cannot fail
			writer.Flush()
			return nil, fmt.Errorf("%s operation failed: %s", executable, strings.TrimSpace(redacted.String()))
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

// protectedLogin routes vendor MFA through the shared terminal popup.
// Login must use --quiet; unlock's token-bearing output uses ProtectedCommand.
func protectedLogin(ctx context.Context, executable string, args []string, env map[string]string, _ io.Reader) ([]byte, error) {
	if system.IsDryRun() {
		return nil, fmt.Errorf("login prohibited during dry-run")
	}
	cmd := exec.CommandContext(ctx, executable, args...) // #nosec G204 -- fixed provider CLI and structured login arguments
	cmd.Env = protectedEnv(env)
	secrets := []string{env["RWR_BW_PASSWORD"], env["BW_SESSION"]}
	err := reporting.RunInteractiveCommand(ctx, cmd, secrets, true)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, fmt.Errorf("credential login failed")
	}
	return nil, nil
}
