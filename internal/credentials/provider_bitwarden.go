package credentials

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"charm.land/huh/v2"
	"github.com/freehold-digital/rwr/internal/reporting"
	"github.com/freehold-digital/rwr/internal/system"
	"github.com/freehold-digital/rwr/internal/types"
)

type bitwardenProvider struct{}
type bitwardenHandle struct {
	binary string
	env    map[string]string
}

var providerCommand = ProtectedCommand
var providerLogin = protectedLogin
var providerPrompt = func(account string) (string, string, error) {
	password := ""
	choice := "continue"
	err := reporting.WithTerminal(func() error {
		if err := huh.NewSelect[string]().Title("Bitwarden credential setup").Options(huh.NewOption("Continue", "continue"), huh.NewOption("Skip setup", "skip")).Value(&choice).Run(); err != nil {
			return err
		}
		if choice == "skip" {
			return &Unavailable{Skipped}
		}
		return huh.NewForm(huh.NewGroup(huh.NewInput().Title("Bitwarden email").Value(&account), huh.NewInput().Title("Master password").EchoMode(huh.EchoModePassword).Value(&password))).Run()
	})
	if errors.Is(err, huh.ErrUserAborted) {
		err = system.ErrCancelled
	}
	return account, password, err
}

func connectionIdentity(c types.CredentialConnection) string {
	sum := sha256.Sum256([]byte(c.Provider + "\x00" + c.Name + "\x00" + c.Server + "\x00" + c.Account))
	return hex.EncodeToString(sum[:])
}
func (p *bitwardenProvider) Open(ctx context.Context, c types.CredentialConnection, o SetupOptions) (Session, error) {
	if system.IsDryRun() {
		return nil, fmt.Errorf("provider access is prohibited during dry-run")
	}
	server := c.Server
	if server == "" {
		server = "https://vault.bitwarden.com"
	}
	if err := validateBitwardenServer(server); err != nil {
		return nil, err
	}
	binary, err := exec.LookPath("bw")
	if err != nil {
		dir, e := bitwardenBinDir()
		if e != nil {
			return nil, e
		}
		name := "bw"
		if filepath.Separator == '\\' {
			name += ".exe"
		}
		candidate := filepath.Join(dir, name)
		if _, e = os.Stat(candidate); e == nil {
			binary = candidate
			err = nil
		}
	}
	if err != nil {
		if !o.Install {
			return nil, &Unavailable{Missing}
		}
		if o.Interactive && stdinIsTerminal() {
			accepted, err := offerBitwardenInstall()
			if err != nil {
				return nil, err
			}
			if !accepted {
				return nil, &Unavailable{Skipped}
			}
		}
		if e := installBitwarden(); e != nil {
			return nil, fmt.Errorf("bitwarden installation failed: %w", e)
		}
		dir, e := bitwardenBinDir()
		if e != nil {
			return nil, e
		}
		binary = filepath.Join(dir, "bw")
		if filepath.Separator == '\\' {
			binary += ".exe"
		}
		info, err := os.Stat(binary)
		if err != nil {
			return nil, fmt.Errorf("bitwarden installation failed: installed executable unavailable: %w", err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("bitwarden installation failed: installed executable is not a regular file")
		}
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	data := filepath.Join(base, "rwr", "credential-providers", connectionIdentity(c))
	if o.Authenticate || o.Install {
		if err := os.MkdirAll(data, 0700); err != nil {
			return nil, err
		}
	} else if _, err := os.Stat(data); err != nil {
		return nil, &Unavailable{Unauthenticated}
	}
	sessionEnv := c.SessionEnv
	if sessionEnv == "" {
		sessionEnv = "BW_SESSION"
	}
	h := &bitwardenHandle{binary: binary, env: map[string]string{"BITWARDENCLI_APPDATA_DIR": data, "BW_SESSION": os.Getenv(sessionEnv), "BITWARDENCLI_DEBUG": "false"}}
	status, err := h.status(ctx)
	if err != nil {
		return nil, err
	}
	if status.ServerURL == "" {
		status.ServerURL = "https://vault.bitwarden.com"
	}
	if err := validateBitwardenServer(status.ServerURL); err != nil {
		return nil, err
	}
	if status.Status != "unauthenticated" && (strings.TrimRight(status.ServerURL, "/") != strings.TrimRight(server, "/") || c.Account != "" && !strings.EqualFold(c.Account, status.UserEmail)) {
		return nil, fmt.Errorf("bitwarden account or endpoint conflicts with the configured connection")
	}
	if status.Status == "unlocked" {
		return h, nil
	}
	if status.Status != "locked" && status.Status != "unauthenticated" {
		return nil, fmt.Errorf("unexpected Bitwarden readiness response")
	}
	if !o.Authenticate || !o.Interactive || !stdinIsTerminal() {
		return nil, &Unavailable{Availability(status.Status)}
	}
	account, password, err := providerPrompt(c.Account)
	if err != nil {
		return nil, err
	}
	if password == "" || (c.Account != "" && !strings.EqualFold(c.Account, account)) {
		return nil, fmt.Errorf("credentials do not match configured Bitwarden account")
	}
	h.env["RWR_BW_PASSWORD"] = password
	defer delete(h.env, "RWR_BW_PASSWORD")
	if status.Status == "unauthenticated" {
		if _, err := h.call(ctx, []string{"config", "server", server}); err != nil {
			return nil, err
		}
		if _, err := providerLogin(ctx, h.binary, []string{"login", account, "--passwordenv", "RWR_BW_PASSWORD", "--quiet"}, h.env, nil); err != nil {
			return nil, err
		}
	}
	raw, err := h.call(ctx, []string{"unlock", "--passwordenv", "RWR_BW_PASSWORD", "--raw"})
	if err != nil {
		return nil, fmt.Errorf("bitwarden unlock failed")
	}
	token := strings.TrimSpace(string(raw))
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return nil, fmt.Errorf("bitwarden returned an invalid session")
	}
	h.env["BW_SESSION"] = token
	return h, nil
}

type bitwardenStatus struct {
	Status    string `json:"status"`
	ServerURL string `json:"serverUrl"`
	UserEmail string `json:"userEmail"`
}

func (h *bitwardenHandle) status(ctx context.Context) (bitwardenStatus, error) {
	out, err := h.call(ctx, []string{"status"})
	var s bitwardenStatus
	if err != nil {
		return s, err
	}
	if json.Unmarshal(out, &s) != nil {
		return s, fmt.Errorf("invalid Bitwarden status response")
	}
	return s, nil
}
func (h *bitwardenHandle) call(ctx context.Context, args []string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, bwTimeout)
	defer cancel()
	return providerCommand(ctx, h.binary, append(append([]string{}, args...), "--nointeraction"), h.env, nil)
}
func (h *bitwardenHandle) ReadSecret(ctx context.Context, r types.CredentialReference) (string, error) {
	key := r.Field
	if key == "" {
		key = "password"
	}
	spec := bwSource{Item: r.Item, Key: key}
	if strings.HasPrefix(key, "field:") {
		spec.Key = "field"
		spec.Field = strings.TrimPrefix(key, "field:")
	}
	args, err := bwArgs(spec)
	if err != nil {
		return "", err
	}
	raw, err := h.call(ctx, args)
	if err != nil {
		return "", err
	}
	if spec.Key == "field" {
		return bwFieldValue(string(raw), spec.Field, r.Item)
	}
	return strings.TrimSuffix(string(raw), "\n"), nil
}
func (h *bitwardenHandle) Close() error {
	for key := range h.env {
		delete(h.env, key)
	}
	return nil
}
