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

	// The dest's directory is declared as the provider's keys path: download
	// destinations are contained inside [provider.repository.paths], the same
	// shape as a real provider writing a keyring.
	newProvider := func(dest string) *types.Provider {
		return &types.Provider{
			Name: "fake",
			Detection: types.DetectionConfig{
				Binary:        "go",
				Distributions: []string{runtime.GOOS},
			},
			Repository: types.RepositoryConfig{
				Paths: types.RepositoryPaths{Keys: filepath.Dir(dest)},
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

// key_sha256 pins the signing key the apt/dnf key-download step fetches. The
// shipped apt provider's own step template carries `sha256 = "{{ .KeySha256 }}"`,
// so the pin flows from the blueprint entry into the download verification.
func TestProcessRepository_KeySha256PinsTheAptKey(t *testing.T) {
	content := []byte("armored key bytes")
	sum := sha256.Sum256(content)
	good := hex.EncodeToString(sum[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(content)
	}))
	defer server.Close()

	run := func(t *testing.T, digest string) error {
		t.Helper()
		root := t.TempDir()
		sourcesDir := filepath.Join(root, "sources.list.d")
		keysDir := filepath.Join(root, "keyrings")
		for _, dir := range []string{sourcesDir, keysDir} {
			if err := os.MkdirAll(dir, 0o750); err != nil {
				t.Fatal(err)
			}
		}
		defer system.SetExecutor(exectest.New())()
		defer system.SetProvidersForTest(map[string]*types.Provider{
			"apt": providerForTest(t, "apt", sourcesDir, keysDir),
		})()

		return processRepository(types.Repository{
			Name:           "pinned",
			PackageManager: "apt",
			Action:         "add",
			URL:            "https://example.com/repo",
			KeyURL:         server.URL + "/gpg",
			KeySha256:      digest,
			Arch:           "amd64",
			Channel:        "stable",
			Component:      "main",
		}, &types.OSInfo{}, &types.InitConfig{})
	}

	t.Run("matching key digest proceeds", func(t *testing.T) {
		if err := run(t, good); err != nil {
			t.Fatalf("processRepository: %v", err)
		}
	})

	t.Run("mismatched key digest refuses", func(t *testing.T) {
		err := run(t, strings.Repeat("0", 64))
		if err == nil || !strings.Contains(err.Error(), "sha256 mismatch") {
			t.Fatalf("err = %v, want a sha256 mismatch refusal", err)
		}
	})
}
