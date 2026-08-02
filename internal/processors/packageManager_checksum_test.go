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

// An install step's sha256 pins what the download must be. It was rendered by
// renderActionStep like every other field and then silently discarded, so a
// provider author's digest bought nothing on this path (repository download
// steps honored theirs).
func TestProcessPackageManagers_InstallStepVerifiesSha256(t *testing.T) {
	content := []byte("installer script bytes")
	sum := sha256.Sum256(content)
	good := hex.EncodeToString(sum[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(content)
	}))
	defer server.Close()

	// Detection.Binary is deliberately absent from PATH: install steps only run
	// when the manager's binary is not yet there.
	newProvider := func(dest, digest string) *types.Provider {
		return &types.Provider{
			Name: "fake",
			Detection: types.DetectionConfig{
				Binary:        "rwr-test-binary-that-does-not-exist",
				Distributions: []string{runtime.GOOS},
			},
			Install: types.InstallConfig{
				Steps: []types.ActionStep{{
					Action: "download",
					Source: server.URL,
					Dest:   dest,
					Sha256: digest,
				}},
			},
		}
	}

	run := func(t *testing.T, dest, digest string) error {
		t.Helper()
		defer system.SetExecutor(exectest.New())()
		defer system.SetProvidersForTest(map[string]*types.Provider{"fake": newProvider(dest, digest)})()

		return ProcessPackageManagers(
			[]types.PackageManagerInfo{{Name: "fake", Action: "install"}},
			&types.OSInfo{},
			&types.InitConfig{},
		)
	}

	t.Run("declared digest matches", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "installer.sh")
		if err := run(t, dest, good); err != nil {
			t.Fatalf("ProcessPackageManagers: %v", err)
		}
		if got, readErr := os.ReadFile(dest); readErr != nil || string(got) != string(content) {
			t.Fatalf("dest = %q (%v), want the downloaded content", got, readErr)
		}
	})

	t.Run("declared digest mismatch refuses", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "installer.sh")
		err := run(t, dest, strings.Repeat("0", 64))
		if err == nil || !strings.Contains(err.Error(), "sha256 mismatch") {
			t.Fatalf("err = %v, want a sha256 mismatch refusal", err)
		}
		if _, statErr := os.Stat(dest); statErr == nil {
			t.Fatal("dest written despite the mismatch")
		}
	})
}
