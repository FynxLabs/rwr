package processors

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/exectest"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// A download step's sha256 is a template like every other field, so a provider
// can pin the fetched content to the digest the blueprint declares.
func TestProcessRepository_DownloadStepVerifiesSha256(t *testing.T) {
	content := []byte("signing key bytes")
	sum := sha256.Sum256(content)
	good := hex.EncodeToString(sum[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(content)
	}))
	defer server.Close()

	newProvider := func(dest string) *types.Provider {
		return &types.Provider{
			Name: "fake",
			Detection: types.DetectionConfig{
				Binary:        "go",
				Distributions: []string{runtime.GOOS},
			},
			Repository: types.RepositoryConfig{
				Add: types.RepositoryAction{
					Steps: []types.ActionStep{{
						Action: "download",
						Source: "{{ .URL }}",
						Dest:   dest,
						Sha256: "{{ .SHA256 }}",
					}},
				},
			},
		}
	}

	t.Run("declared digest matches", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "key.gpg")
		defer system.SetExecutor(exectest.New())()
		defer system.SetProvidersForTest(map[string]*types.Provider{"fake": newProvider(dest)})()

		err := processRepository(types.Repository{
			Name:           "example",
			PackageManager: "fake",
			Action:         "add",
			URL:            server.URL,
			SHA256:         good,
		}, &types.OSInfo{}, &types.InitConfig{})
		if err != nil {
			t.Fatalf("processRepository: %v", err)
		}
		if got, readErr := os.ReadFile(dest); readErr != nil || string(got) != string(content) {
			t.Fatalf("dest = %q (%v), want the downloaded content", got, readErr)
		}
	})

	t.Run("declared digest mismatch refuses", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "key.gpg")
		defer system.SetExecutor(exectest.New())()
		defer system.SetProvidersForTest(map[string]*types.Provider{"fake": newProvider(dest)})()

		err := processRepository(types.Repository{
			Name:           "example",
			PackageManager: "fake",
			Action:         "add",
			URL:            server.URL,
			SHA256:         strings.Repeat("0", 64),
		}, &types.OSInfo{}, &types.InitConfig{})
		if err == nil || !strings.Contains(err.Error(), "sha256 mismatch") {
			t.Fatalf("err = %v, want a sha256 mismatch refusal", err)
		}
		if _, statErr := os.Stat(dest); statErr == nil {
			t.Fatal("dest written despite the mismatch")
		}
	})
}
