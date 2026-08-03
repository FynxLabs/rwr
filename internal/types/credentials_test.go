package types

import (
	"strings"
	"testing"
)

// resetCredentialState restores the package-level registries a test mutates.
func resetCredentialState(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		RegisterCredentials(nil)
		SetExposedCredentials(nil)
	})
}

func TestDecodeCredentialSpecs(t *testing.T) {
	tests := []struct {
		name    string
		raw     interface{}
		wantErr string
	}{
		{name: "absent section", raw: nil},
		{
			name: "well-formed entry",
			raw: []interface{}{map[string]interface{}{
				"name":        "cachix_token",
				"description": "Cachix auth token",
				"sources":     []interface{}{"env:CACHIX_AUTH_TOKEN", "keyring", "prompt"},
				"scope":       []interface{}{"scripts"},
			}},
		},
		{
			// Viper's own Unmarshal ignores unknown keys; the strict re-decode is
			// what turns a typo into an error instead of a silently different
			// source order.
			name: "unknown key errors",
			raw: []interface{}{map[string]interface{}{
				"name":    "cachix_token",
				"soruces": []interface{}{"keyring"},
			}},
			wantErr: "soruces",
		},
		{
			name:    "invalid name",
			raw:     []interface{}{map[string]interface{}{"name": "Cachix-Token"}},
			wantErr: "lowercase identifiers",
		},
		{
			name: "duplicate declaration",
			raw: []interface{}{
				map[string]interface{}{"name": "cachix_token"},
				map[string]interface{}{"name": "cachix_token"},
			},
			wantErr: "declared twice",
		},
		{
			name:    "built-ins cannot be redeclared",
			raw:     []interface{}{map[string]interface{}{"name": "gh_api_token"}},
			wantErr: "built in",
		},
		{
			name: "unknown source",
			raw: []interface{}{map[string]interface{}{
				"name":    "cachix_token",
				"sources": []interface{}{"vault"},
			}},
			wantErr: `unknown source "vault"`,
		},
		{
			name: "env source without a variable",
			raw: []interface{}{map[string]interface{}{
				"name":    "cachix_token",
				"sources": []interface{}{"env:"},
			}},
			wantErr: "names no environment variable",
		},
		{
			name: "unknown scope",
			raw: []interface{}{map[string]interface{}{
				"name":  "cachix_token",
				"scope": []interface{}{"everything"},
			}},
			wantErr: `unknown scope "everything"`,
		},
		{
			name: "processor-name scope",
			raw: []interface{}{map[string]interface{}{
				"name":  "cachix_token",
				"scope": []interface{}{"ssh_keys"},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeCredentialSpecs(tt.raw)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("DecodeCredentialSpecs = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("DecodeCredentialSpecs = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

// A declared credential is managed exactly like the built-ins: the registry is
// what generalizes the fixed two-key list.
func TestIsManagedCredential(t *testing.T) {
	resetCredentialState(t)
	RegisterCredentials([]CredentialSpec{{Name: "cachix_token"}})

	for _, name := range []string{"gh_api_token", "ssh_private_key", "cachix_token"} {
		if !IsManagedCredential(name) {
			t.Errorf("IsManagedCredential(%q) = false, want true", name)
		}
	}
	if IsManagedCredential("editor_theme") {
		t.Error("IsManagedCredential(\"editor_theme\") = true, want false")
	}
}

// A resolved value leaves the registry only through the exposure gates: absent
// from template scope and the env export without the exposeCredentials opt-in,
// present with it.
func TestCredentialExposureGates(t *testing.T) {
	resetCredentialState(t)
	RegisterCredentials([]CredentialSpec{{Name: "cachix_token"}})
	SetCredentialValue("cachix_token", "cachix-secret-value")

	if _, ok := TemplateCredentials()["cachix_token"]; ok {
		t.Error("unexposed credential reached template scope")
	}
	if len(ExportedCredentialEnv()) != 0 {
		t.Errorf("unexposed credential reached the env export: %v", ExportedCredentialEnv())
	}

	SetExposedCredentials([]string{"cachix_token"})

	if got := TemplateCredentials()["cachix_token"]; got != "cachix-secret-value" {
		t.Errorf("exposed credential missing from template scope: %v", got)
	}
	if got := ExportedCredentialEnv()["RWR_CRED_CACHIX_TOKEN"]; got != "cachix-secret-value" {
		t.Errorf("exposed credential missing from the env export: %v", ExportedCredentialEnv())
	}
}

// Scope narrows exposure per surface: a scripts-only credential stays out of
// template scope even when exposed, and vice versa. A scope naming only a
// processor restricts when the credential resolves, not where it may appear.
func TestCredentialScopeGates(t *testing.T) {
	resetCredentialState(t)
	RegisterCredentials([]CredentialSpec{
		{Name: "script_token", Scope: []string{CredentialSurfaceScripts}},
		{Name: "template_token", Scope: []string{CredentialSurfaceTemplates}},
		{Name: "ssh_token", Scope: []string{BlueprintTypeSSHKeys}},
	})
	SetExposedCredentials([]string{"script_token", "template_token", "ssh_token"})
	SetCredentialValue("script_token", "s")
	SetCredentialValue("template_token", "t")
	SetCredentialValue("ssh_token", "k")

	templates := TemplateCredentials()
	env := ExportedCredentialEnv()

	if _, ok := templates["script_token"]; ok {
		t.Error("scripts-scoped credential reached template scope")
	}
	if _, ok := env["RWR_CRED_SCRIPT_TOKEN"]; !ok {
		t.Error("scripts-scoped credential missing from the env export")
	}
	if _, ok := templates["template_token"]; !ok {
		t.Error("templates-scoped credential missing from template scope")
	}
	if _, ok := env["RWR_CRED_TEMPLATE_TOKEN"]; ok {
		t.Error("templates-scoped credential reached the env export")
	}
	if _, ok := templates["ssh_token"]; !ok {
		t.Error("processor-scoped credential missing from template scope")
	}
	if _, ok := env["RWR_CRED_SSH_TOKEN"]; !ok {
		t.Error("processor-scoped credential missing from the env export")
	}
}

// The debug dump of template variables may log credential names, never values.
func TestTemplateCredentialNames(t *testing.T) {
	resetCredentialState(t)
	RegisterCredentials([]CredentialSpec{{Name: "b_token"}, {Name: "a_token"}})
	SetExposedCredentials([]string{"a_token", "b_token"})
	SetCredentialValue("a_token", "va")
	SetCredentialValue("b_token", "vb")

	names := TemplateCredentialNames()
	if len(names) != 2 || names[0] != "a_token" || names[1] != "b_token" {
		t.Errorf("TemplateCredentialNames = %v, want sorted [a_token b_token]", names)
	}
	for _, name := range names {
		if strings.Contains(name, "va") || strings.Contains(name, "vb") {
			t.Errorf("TemplateCredentialNames leaked a value: %v", names)
		}
	}
}
