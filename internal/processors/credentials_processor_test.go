package processors

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/freehold-digital/rwr/internal/credentials"
	"github.com/freehold-digital/rwr/internal/helpers"
	"github.com/freehold-digital/rwr/internal/system"
	"github.com/freehold-digital/rwr/internal/types"
)

type testCredentialProvider struct {
	opens *int
	fail  error
}

func (p testCredentialProvider) Open(_ context.Context, _ types.CredentialConnection, _ credentials.SetupOptions) (credentials.Session, error) {
	*p.opens++
	if p.fail != nil {
		return nil, p.fail
	}
	return &testCredentialSession{}, nil
}

type testCredentialSession struct{}

func (*testCredentialSession) ReadSecret(context.Context, types.CredentialReference) (string, error) {
	return "test-secret", nil
}
func (*testCredentialSession) Close() error { return nil }

func TestCredentialsRunSelectionEndToEnd(t *testing.T) {
	for _, tt := range []struct {
		name     string
		selected []string
		except   []string
		profiles []string
		want     int
		fail     error
		wantErr  bool
	}{
		{name: "default full run", want: 0},
		{name: "excluded", except: []string{"credentials"}, want: 0},
		{name: "standalone", selected: []string{"credentials"}, profiles: []string{"bitwarden"}, want: 1},
		{name: "explicit failure", selected: []string{"credentials"}, profiles: []string{"bitwarden"}, want: 1, fail: errors.New("authentication failed"), wantErr: true},
		{name: "explicit skip", selected: []string{"credentials"}, profiles: []string{"bitwarden"}, want: 1, fail: &credentials.Unavailable{State: credentials.Skipped}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer system.BeginRun()()
			count := 0
			defer credentials.RegisterProvider("fixture", func() credentials.Provider { return testCredentialProvider{&count, tt.fail} })()
			tree := writeBlueprintTree(t, map[string]string{"credentials/main.yaml": "credential_setup:\n - name: vault\n   profiles: [bitwarden]\n   connection: personal\n"})
			c := treeConfig(tree)
			c.Init.Except = []string{"credentials"}
			c.Variables.Flags.Except = tt.except
			c.Variables.Flags.Profiles = tt.profiles
			c.CredentialProviders = []types.CredentialConnection{{Name: "personal", Provider: "fixture"}}
			if err := All(c, &types.OSInfo{}, tt.selected); (err != nil) != tt.wantErr {
				t.Fatalf("run error=%v", err)
			}
			if count != tt.want {
				t.Fatalf("provider calls=%d want=%d", count, tt.want)
			}
		})
	}
}

func TestCredentialDependenciesSkipOnlyConsumer(t *testing.T) {
	for _, strict := range []bool{false, true} {
		t.Run(fmt.Sprint(strict), func(t *testing.T) {
			defer system.BeginRun()()
			resetFailures()
			count := 0
			defer credentials.RegisterProvider("fixture", func() credentials.Provider { count++; t.Fatal("excluded provider called"); return nil })()
			c := treeConfig(t.TempDir())
			c.Variables.Flags.Selection = &types.RunSelection{DenyProviders: true}
			c.CredentialProviders = []types.CredentialConnection{{Name: "personal", Provider: "fixture"}}
			c.Credentials = []types.CredentialSpec{{Name: "password", Scope: []string{"scripts"}, References: []types.CredentialReference{{Connection: "personal", Item: "one"}}}}
			if strict {
				c.CredentialPolicy.OnUnavailable = "fail"
			}
			types.RegisterCredentials(c.Credentials)
			defer types.RegisterCredentials(nil)
			marker := filepath.Join(t.TempDir(), "independent")
			data := []byte(fmt.Sprintf("scripts:\n - name: consumer\n   action: run\n   exec: self\n   requiresCredentials: [password]\n   content: invalid-executable\n - name: independent\n   action: run\n   exec: self\n   content: |\n     #!/bin/sh\n     echo done > '%s'\n", marker))
			if err := ProcessScripts(data, c.Init.Location, "yaml", &types.OSInfo{}, c); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(marker); err != nil {
				t.Fatal("independent script did not run")
			}
			if (failureCount() > 0) != strict {
				t.Fatalf("failure count=%d strict=%v", failureCount(), strict)
			}
			if count != 0 {
				t.Fatal("provider invoked")
			}
		})
	}
}

func TestProfilesPreventCredentialLookup(t *testing.T) {
	defer system.BeginRun()()
	resetFailures()
	count := 0
	defer credentials.RegisterProvider("fixture", func() credentials.Provider { count++; t.Fatal("filtered resource resolved credential"); return nil })()
	c := treeConfig(t.TempDir())
	c.Variables.Flags.Profiles = []string{"desktop"}
	c.Credentials = []types.CredentialSpec{{Name: "password", References: []types.CredentialReference{{Connection: "personal", Item: "one"}}}}
	c.CredentialProviders = []types.CredentialConnection{{Name: "personal", Provider: "fixture"}}
	types.RegisterCredentials(c.Credentials)
	defer types.RegisterCredentials(nil)
	raw := []byte("scripts:\n - name: filtered\n   profiles: [restore]\n   requiresCredentials: [password]\n   content: '{{ .Credentials.password }}'\n   action: run\n")
	data, err := helpers.ResolveStaticTemplate(raw, c.Variables)
	if err != nil {
		t.Fatal(err)
	}
	if err := ProcessScripts(data, c.Init.Location, "yaml", &types.OSInfo{}, c); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("provider called")
	}
}

func TestCredentialScopeRestoredAndNoParentExport(t *testing.T) {
	defer system.BeginRun()()
	resetFailures()
	t.Setenv("TEST_RWR_SECRET", "correct-value")
	c := treeConfig(t.TempDir())
	c.Credentials = []types.CredentialSpec{{Name: "password", Sources: []string{"env:TEST_RWR_SECRET"}, Scope: []string{"scripts"}}}
	types.RegisterCredentials(c.Credentials)
	types.SetExposedCredentials([]string{"password"})
	defer types.RegisterCredentials(nil)
	defer types.SetExposedCredentials(nil)
	data := []byte("scripts:\n - name: consumer\n   action: run\n   exec: self\n   requiresCredentials: [password]\n   content: |\n     #!/bin/sh\n     test \"$RWR_CRED_PASSWORD\" = correct-value\n - name: independent\n   action: run\n   exec: self\n   content: |\n     #!/bin/sh\n     test -z \"$RWR_CRED_PASSWORD\"\n")
	if err := ProcessScripts(data, c.Init.Location, "yaml", &types.OSInfo{}, c); err != nil {
		t.Fatal(err)
	}
	if value, _ := types.CredentialValue("password"); value != "" {
		t.Fatal("value leaked beyond resource")
	}
	if os.Getenv("RWR_CRED_PASSWORD") != "" {
		t.Fatal("parent environment changed")
	}
}

func TestCredentialDryRunNeverOpensProvider(t *testing.T) {
	system.SetDryRun(true)
	defer system.SetDryRun(false)
	defer credentials.RegisterProvider("fixture", func() credentials.Provider { t.Fatal("dry-run opened provider"); return nil })()
	c := treeConfig(t.TempDir())
	c.CredentialProviders = []types.CredentialConnection{{Name: "personal", Provider: "fixture"}}
	if err := ProcessCredentials([]byte("credential_setup:\n - name: setup\n   connection: personal\n"), c.Init.Location, "yaml", &types.OSInfo{}, c); err != nil {
		t.Fatal(err)
	}
}

func TestCredentialsOnlyDoesNotBootstrap(t *testing.T) {
	defer system.BeginRun()()
	tree := writeBlueprintTree(t, map[string]string{"bootstrap.yaml": "scripts:\n - name: must-not-run\n   action: run\n   content: invalid-executable\n   exec: self\n", "credentials/main.yaml": "credential_setup: []\n"})
	if err := All(treeConfig(tree), &types.OSInfo{}, []string{"credentials"}); err != nil {
		t.Fatal(err)
	}
}

func TestCredentialSchemaAcrossFormats(t *testing.T) {
	for format, data := range map[string]string{
		"yaml": "credential_setup:\n - name: setup\n   connection: personal\n",
		"json": `{"credential_setup":[{"name":"setup","connection":"personal"}]}`,
		"toml": "[[credential_setup]]\nname='setup'\nconnection='personal'\n",
		"cue":  "credential_setup: [{name: \"setup\", connection: \"personal\"}]",
	} {
		t.Run(format, func(t *testing.T) {
			var d types.CredentialSetupData
			if err := helpers.DecodeBlueprintInto([]byte(data), format, "credentials", 0, &d); err != nil {
				t.Fatal(err)
			}
			if err := d.Validate(); err != nil {
				t.Fatal(err)
			}
			bad := strings.Replace(data, "connection", "unknown_field", 1)
			if err := helpers.DecodeBlueprintInto([]byte(bad), format, "credentials", 0, &d); err == nil {
				t.Fatal("unknown field accepted")
			}
		})
	}
}

func TestCredentialValuesRedactedFromChildLogs(t *testing.T) {
	defer system.BeginRun()()
	resetFailures()
	t.Setenv("RWR_TEST_TOKEN", "sensitive-test-value")
	c := treeConfig(t.TempDir())
	c.Credentials = []types.CredentialSpec{{Name: "password", Sources: []string{"env:RWR_TEST_TOKEN"}, Scope: []string{"scripts"}}}
	types.RegisterCredentials(c.Credentials)
	types.SetExposedCredentials([]string{"password"})
	defer types.RegisterCredentials(nil)
	defer types.SetExposedCredentials(nil)
	path := filepath.Join(t.TempDir(), "output.log")
	data := []byte(fmt.Sprintf("scripts:\n - name: consumer\n   exec: self\n   action: run\n   log: '%s'\n   requiresCredentials: [password]\n   content: |\n     #!/bin/sh\n     printf 'value=%%s' \"$RWR_CRED_PASSWORD\"\n", path))
	if err := ProcessScripts(data, c.Init.Location, "yaml", &types.OSInfo{}, c); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "sensitive-test-value") || !strings.Contains(string(raw), "[redacted]") {
		t.Fatalf("unsafe log: %q", raw)
	}
}

func TestCredentialTemplateLateRenderingAndPermissions(t *testing.T) {
	defer system.BeginRun()()
	resetFailures()
	t.Setenv("RWR_TEST_TOKEN", "quotes\"and\nnewlines")
	c := treeConfig(t.TempDir())
	c.Credentials = []types.CredentialSpec{{Name: "password", Sources: []string{"env:RWR_TEST_TOKEN"}, Scope: []string{"templates"}}}
	types.RegisterCredentials(c.Credentials)
	types.SetExposedCredentials([]string{"password"})
	defer types.RegisterCredentials(nil)
	defer types.SetExposedCredentials(nil)
	target := t.TempDir()
	data, err := helpers.ResolveStaticTemplate([]byte(fmt.Sprintf("files:\n - name: secret\n   action: create\n   target: '%s/'\n   content: '{{ .Credentials.password }}'\n", target)), c.Variables)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "quotes") {
		t.Fatal("structural rendering acquired a secret")
	}
	if err := ProcessFiles(data, c.Init.Location, "yaml", &types.OSInfo{}, c); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(target, "secret")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "quotes\"and\nnewlines" {
		t.Fatalf("rendered=%q", raw)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("mode=%v err=%v", info, err)
	}
}

func TestSourceEnvironmentDoesNotBypassDependencyExposure(t *testing.T) {
	defer system.BeginRun()()
	resetFailures()
	t.Setenv("SOURCE_SECRET", "secret-value")
	t.Setenv("BW_SESSION", "session-value")
	c := treeConfig(t.TempDir())
	c.Credentials = []types.CredentialSpec{{Name: "password", Sources: []string{"env:SOURCE_SECRET"}, Scope: []string{"scripts"}}}
	types.RegisterCredentials(c.Credentials)
	defer types.RegisterCredentials(nil)
	data := []byte("scripts:\n - name: independent\n   action: run\n   exec: self\n   content: |\n     #!/bin/sh\n     test -z \"$SOURCE_SECRET$BW_SESSION$RWR_CRED_PASSWORD\"\n")
	if err := ProcessScripts(data, c.Init.Location, "yaml", &types.OSInfo{}, c); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("SOURCE_SECRET") != "secret-value" {
		t.Fatal("parent source changed")
	}
}
