package processors

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
)

func TestBootstrapInstallsCredentialToolBeforeResolution(t *testing.T) {
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
	if value, _ := types.CredentialValue("bootstrap_secret"); value != "vault-value" {
		t.Fatal("did not use bootstrapped CLI")
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
	if err == nil || !strings.Contains(err.Error(), "no profile named") {
		t.Fatalf("expected profile rejection before bootstrap, got %v", err)
	}
}
