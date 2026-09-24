package processors

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/freehold-digital/rwr/internal/system"
	"github.com/freehold-digital/rwr/internal/types"
	"github.com/spf13/viper"
)

func TestBootstrapDoesNotTriggerCredentialResolution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixture")
	}
	defer system.BeginRun()()
	viper.Reset()
	defer viper.Reset()
	configDir := t.TempDir()
	viper.Set("rwr.configdir", configDir)
	bin := t.TempDir()
	t.Setenv("PATH", bin)
	tree := writeBlueprintTree(t, map[string]string{
		"init.yaml":      "blueprints:\n  location: .\ncredentials:\n - name: bootstrap_secret\n   sources: [bw:fixture]\n",
		"bootstrap.yaml": fmt.Sprintf("scripts:\n - name: prepare\n   action: run\n   exec: self\n   content: |\n     #!/bin/sh\n     printf '#!/bin/sh\\nprintf vault-value' > '%s/bw'\n     /bin/chmod 700 '%s/bw'\n", bin, bin),
	})
	config, err := LoadConfiguration(filepath.Join(tree, "init.yaml"), types.Flags{})
	if err != nil {
		t.Fatalf("configuration required a credential before bootstrap: %v", err)
	}
	if _, ok := types.CredentialValue("bootstrap_secret"); ok {
		t.Fatal("resolved before bootstrap")
	}
	if err := All(config, &types.OSInfo{}, nil); err != nil {
		t.Fatal(err)
	}
	if value, _ := types.CredentialValue("bootstrap_secret"); value != "" {
		t.Fatal("bootstrap unexpectedly resolved credentials")
	}
	if _, err := os.Stat(filepath.Join(configDir, "bootstrap")); err != nil {
		t.Fatal("bootstrap not marked successful")
	}
}

func TestInvalidProfileStopsBeforeBootstrap(t *testing.T) {
	defer system.BeginRun()()
	tree := writeBlueprintTree(t, map[string]string{
		"scripts/main.yaml": "scripts:\n - name: selected\n   action: run\n   profiles: [desktop]\n   content: exit 0\n",
		"bootstrap.yaml":    "scripts:\n - name: must-not-run\n   action: run\n   exec: self\n   content: invalid executable\n",
	})
	config := treeConfig(tree)
	config.Variables.Flags.Profiles = []string{"typo"}
	err := All(config, &types.OSInfo{}, nil)
	if !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("expected profile rejection before bootstrap, got %v", err)
	}
}

func TestInvalidBootstrapManagerStopsBeforePreparation(t *testing.T) {
	for _, standalone := range []bool{false, true} {
		t.Run(fmt.Sprint(standalone), func(t *testing.T) {
			defer system.BeginRun()()
			viper.Reset()
			defer viper.Reset()
			viper.Set("rwr.configdir", t.TempDir())
			tree := writeBlueprintTree(t, map[string]string{"bootstrap.yaml": "packageManagers:\n - name: brew\n   action: install\n - name: yay\n   action: invalid\nscripts:\n - name: must-not-run\n   action: run\n   exec: self\n   content: invalid executable\n"})
			config := treeConfig(tree)
			failuresBefore := failureCount()
			var err error
			if standalone {
				err = RunBootstrap(config, &types.OSInfo{})
			} else {
				err = All(config, &types.OSInfo{}, nil)
			}
			if !errors.Is(err, types.ErrInvalidPackageManager) {
				t.Fatalf("expected package-manager validation error: %v", err)
			}
			if failureCount() != failuresBefore {
				t.Fatal("invalid preparation script was executed")
			}
		})
	}
}

func TestBootstrapRetrySkipsCompletedPreparationScripts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixture")
	}
	defer system.BeginRun()()
	viper.Reset()
	defer viper.Reset()
	viper.Set("rwr.configdir", t.TempDir())
	dir := t.TempDir()
	counter := filepath.Join(dir, "count")
	gate := filepath.Join(dir, "ready")
	scripts := []types.Script{
		{Name: "first", Action: "run", Exec: "self", Content: "#!/bin/sh\nprintf x >> '" + counter + "'\n"},
		{Name: "later", Action: "run", Exec: "self", Content: "#!/bin/sh\ntest -f '" + gate + "'\n"},
	}
	config := treeConfig(dir)
	if err := processBootstrapScripts(scripts, &types.OSInfo{}, config, dir); err == nil {
		t.Fatal("later script should fail")
	}
	if err := os.WriteFile(gate, nil, 0600); err != nil {
		t.Fatal(err)
	}
	config.Variables.Flags.ForceBootstrap = true
	if err := processBootstrapScripts(scripts, &types.OSInfo{}, config, dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(counter)
	if err != nil || string(data) != "x" {
		t.Fatalf("successful preparation repeated: %q, %v", data, err)
	}
}

func TestBootstrapRejectsCredentialsBeforeMutation(t *testing.T) {
	for _, content := range []string{
		"files:\n - name: out.txt\n   action: create\n   target: .\n   content: '{{ .Credentials.testcred }}'\n",
		"directories:\n - name: '{{ .Credentials.testcred }}'\n   action: create\n   target: .\n",
		"scripts:\n - name: prepare\n   action: run\n   requiresCredentials: [testcred]\n   content: invalid executable\n",
	} {
		for _, standalone := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/standalone=%t", strings.Split(content, ":")[0], standalone), func(t *testing.T) {
				defer system.BeginRun()()
				viper.Reset()
				defer viper.Reset()
				configDir := t.TempDir()
				viper.Set("rwr.configdir", configDir)
				tree := writeBlueprintTree(t, map[string]string{"bootstrap.yaml": content})
				config := treeConfig(tree)
				config.Credentials = []types.CredentialSpec{{Name: "testcred", Sources: []string{"env:RWR_TEST_BOOTSTRAP_SECRET"}}}
				config.ExposeCredentials = []string{"testcred"}
				t.Setenv("RWR_TEST_BOOTSTRAP_SECRET", "secret-must-not-render")
				var err error
				if standalone {
					err = RunBootstrap(config, &types.OSInfo{})
				} else {
					err = All(config, &types.OSInfo{}, nil)
				}
				if err == nil || !strings.Contains(err.Error(), "bootstrap cannot") {
					t.Fatalf("expected bootstrap credential error, got %v", err)
				}
				entries, readErr := os.ReadDir(tree)
				if readErr != nil || len(entries) != 1 || entries[0].Name() != "bootstrap.yaml" {
					t.Fatalf("bootstrap changed tree: %v, %v", entries, readErr)
				}
				if _, err := os.Stat(filepath.Join(configDir, "bootstrap")); !os.IsNotExist(err) {
					t.Fatal("failed bootstrap marked complete")
				}
			})
		}
	}
}
