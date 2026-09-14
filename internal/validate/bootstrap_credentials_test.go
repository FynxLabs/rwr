package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestBootstrapCredentialValidation(t *testing.T) {
	t.Parallel()
	for name, content := range map[string]string{
		"file":       "files:\n - name: out.txt\n   action: create\n   target: .\n   content: '{{ .Credentials.testcred }}'\n",
		"directory":  "directories:\n - name: '{{ .Credentials.testcred }}'\n   action: create\n   target: .\n",
		"dependency": "scripts:\n - name: prepare\n   action: run\n   requiresCredentials: [testcred]\n   content: echo ready\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "bootstrap.yaml")
			if err := os.WriteFile(path, []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			config := &types.InitConfig{Credentials: []types.CredentialSpec{{Name: "testcred"}}}
			err := validateBlueprintFile(path, config, &types.ValidationResults{})
			if err == nil || !strings.Contains(err.Error(), "bootstrap cannot") {
				t.Fatalf("validation accepted credential-dependent bootstrap: %v", err)
			}
		})
	}
}
