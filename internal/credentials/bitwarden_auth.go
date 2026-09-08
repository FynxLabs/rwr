package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
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
	// An empty server is the CLI default (the hosted HTTPS vault).
	if status.ServerURL != "" {
		if err := validateBitwardenServer(status.ServerURL); err != nil {
			return err
		}
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
		if err := validateBitwardenServer(server); err != nil {
			return err
		}
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
			huh.NewInput().Title("Bitwarden server").Description("Use vault.bitwarden.eu for EU accounts, or your self-hosted URL.").Value(&server).Validate(validateBitwardenServer),
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
	cmd := exec.CommandContext(ctx, "bw", args...) // #nosec G204 -- fixed bw executable; internal login/unlock arguments are passed separately, never through a shell
	for _, entry := range bitwardenEnv() {
		if !strings.HasPrefix(entry, "RWR_BW_PASSWORD=") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env, "RWR_BW_PASSWORD="+password)
	var out, stderr strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	// Login owns the terminal for MFA prompts; --quiet suppresses the session.
	run := cmd.Run
	if interactive {
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		run = func() error { return reporting.WithTerminal(cmd.Run) }
	}
	if err := run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		for _, secret := range []string{password, bitwardenSession, os.Getenv("BW_SESSION")} {
			if secret != "" {
				detail = strings.ReplaceAll(detail, secret, "[redacted]")
			}
		}
		return "", &BitwardenAuthenticationError{Action: args[0], Diagnostic: detail, Err: err}
	}
	return out.String(), nil
}

func validateBitwardenServer(server string) error {
	parsed, err := url.Parse(server)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" {
		return fmt.Errorf("bitwarden server must be an HTTPS URL with a host and no embedded credentials or fragment")
	}
	return nil
}

// BitwardenAuthenticationError retains the command failure and its redacted diagnostic.
type BitwardenAuthenticationError struct {
	Action     string
	Diagnostic string
	Err        error
}

func (e *BitwardenAuthenticationError) Error() string {
	return fmt.Sprintf("bitwarden %s failed: %v: %s", e.Action, e.Err, e.Diagnostic)
}

func (e *BitwardenAuthenticationError) Unwrap() error { return e.Err }
