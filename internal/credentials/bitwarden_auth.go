package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"charm.land/huh/v2"
	"github.com/fynxlabs/rwr/internal/reporting"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

var bwAuthStatus = func() (string, error) { return runBW([]string{"status"}) }

// Authenticate in RWR's terminal lease. Passwords are passed through a child-only
// environment variable, never argv, logs, a shell, or the credential registry.
var authenticateBitwarden = func() error {
	out, err := bwAuthStatus()
	if err != nil {
		return fmt.Errorf("checking Bitwarden status: %w", err)
	}
	var status struct {
		Status    string `json:"status"`
		ServerURL string `json:"serverUrl"`
	}
	if err := json.Unmarshal([]byte(out), &status); err != nil {
		return fmt.Errorf("invalid Bitwarden status: %w", err)
	}
	if status.Status == "unlocked" {
		return nil
	}
	if status.Status != "locked" && status.Status != "unauthenticated" {
		return fmt.Errorf("unknown Bitwarden status %q", status.Status)
	}
	login := status.Status == "unauthenticated"
	email, password, server, err := promptBitwardenAuth(login, status.ServerURL)
	if err != nil {
		return err
	}
	if login {
		if server != "" && server != status.ServerURL {
			if _, err := runBW([]string{"config", "server", server}); err != nil {
				return fmt.Errorf("configuring Bitwarden server: %w", err)
			}
		}
		if _, err := bwAuthenticate([]string{"login", email, "--passwordenv", "RWR_BW_PASSWORD", "--quiet"}, password, true); err != nil {
			return err
		}
	}
	session, err := bwAuthenticate([]string{"unlock", "--passwordenv", "RWR_BW_PASSWORD", "--raw"}, password, false)
	if err != nil {
		return err
	}
	session = strings.TrimSpace(session)
	if session == "" || strings.ContainsAny(session, " \t\r\n") {
		return fmt.Errorf("bitwarden did not return a valid session")
	}
	bitwardenSession = session
	types.SetCredentialValue("bw_session", session)
	return nil
}

var promptBitwardenAuth = func(login bool, currentServer string) (email, password, server string, err error) {
	server = currentServer
	if server == "" {
		server = "https://vault.bitwarden.com"
	}
	fields := []huh.Field{}
	required := func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("value cannot be empty")
		}
		return nil
	}
	if login {
		fields = append(fields,
			huh.NewInput().Title("Bitwarden server").Description("Use vault.bitwarden.eu for EU accounts, or your self-hosted URL.").Value(&server).Validate(required),
			huh.NewInput().Title("Bitwarden email").Value(&email).Validate(required))
	}
	fields = append(fields, huh.NewInput().Title("Unlock Bitwarden").Description("Enter your master password. RWR uses it only for this login and unlock.").EchoMode(huh.EchoModePassword).Value(&password).Validate(required))
	err = reporting.WithTerminal(huh.NewForm(huh.NewGroup(fields...)).Run)
	return
}

var bwAuthenticate = func(args []string, password string, interactive bool) (string, error) {
	ctx := system.RunContext()
	var cancel context.CancelFunc
	if !interactive {
		ctx, cancel = context.WithTimeout(ctx, bwTimeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, "bw", args...)
	for _, entry := range bitwardenEnv() {
		if !strings.HasPrefix(entry, "RWR_BW_PASSWORD=") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env, "RWR_BW_PASSWORD="+password)
	var out strings.Builder
	cmd.Stdout = &out
	// Login owns the terminal for MFA prompts; --quiet suppresses the session.
	run := cmd.Run
	if interactive {
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		run = func() error { return reporting.WithTerminal(cmd.Run) }
	}
	if err := run(); err != nil {
		return "", fmt.Errorf("bitwarden %s failed: %w", args[0], err)
	}
	return out.String(), nil
}
