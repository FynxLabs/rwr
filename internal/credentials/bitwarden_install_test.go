package credentials

import (
	"archive/zip"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

func TestManagedBitwardenDiscoveredOnLaterRun(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("PATH", "")
	dir, err := bitwardenBinDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	name := "bw"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("fixture"), 0o700); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := addBitwardenPath(); err != nil {
			t.Fatal(err)
		}
		if os.Getenv("PATH") != dir {
			t.Fatal("managed PATH duplicates entries or adds current directory")
		}
		if got, err := exec.LookPath("bw"); err != nil || got != path {
			t.Fatalf("got %q, %v", got, err)
		}
	}
}

func TestMissingBitwardenHasTypedCause(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := fetchWithCLI(bwSource{Item: "fixture", Key: "password"})
	if !errors.Is(err, ErrBitwardenNotInstalled) {
		t.Fatalf("got %v", err)
	}
}

func TestMissingBitwardenInstallOrSkip(t *testing.T) {
	for _, tc := range []struct {
		name                                                    string
		interactive, tty, dryRun, accept, installFails, session bool
		wantOffers, wantInstalls                                int
	}{
		{name: "headless"},
		{name: "no terminal", interactive: true},
		{name: "noninteractive terminal", tty: true},
		{name: "dry run", interactive: true, tty: true, dryRun: true},
		{name: "skip", interactive: true, tty: true, wantOffers: 1},
		{name: "install fails", interactive: true, tty: true, accept: true, installFails: true, wantOffers: 1, wantInstalls: 1},
		{name: "install without vault session", interactive: true, tty: true, accept: true, wantOffers: 1, wantInstalls: 1},
		{name: "install with vault session", interactive: true, tty: true, accept: true, session: true, wantOffers: 1, wantInstalls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			withFakes(t, &fakeKeyring{}, tc.tty, func(_, _ string) (string, error) {
				t.Fatal("missing Bitwarden must not force a credential prompt")
				return "", nil
			})
			t.Setenv("BW_SESSION", "")
			if tc.session {
				t.Setenv("BW_SESSION", "test-session")
			}
			wasDryRun := system.IsDryRun()
			system.SetDryRun(tc.dryRun)
			t.Cleanup(func() { system.SetDryRun(wasDryRun) })
			origFetch, origOffer, origInstall := bwFetch, offerBitwardenInstall, installBitwarden
			t.Cleanup(func() { bwFetch, offerBitwardenInstall, installBitwarden = origFetch, origOffer, origInstall })
			offers, installs := 0, 0
			installed := false
			bwFetch = func(bwSource) (string, error) {
				if installed && tc.session {
					return "vault-value", nil
				}
				return "", ErrBitwardenNotInstalled
			}
			offerBitwardenInstall = func() bool { offers++; return tc.accept }
			installBitwarden = func() error {
				installs++
				if tc.installFails {
					return errors.New("download failed")
				}
				installed = true
				return nil
			}
			specs := []types.CredentialSpec{
				{Name: "one", Sources: []string{"bw:one", "keyring", "prompt"}},
				{Name: "two", Sources: []string{"bw:two", "prompt"}},
				{Name: "unrelated", Sources: []string{"env:TEST_UNRELATED"}},
			}
			t.Setenv("TEST_UNRELATED", "still-resolved")
			types.RegisterCredentials(specs)
			if err := Resolve(specs, Options{Interactive: tc.interactive}); err != nil {
				t.Fatal(err)
			}
			if offers != tc.wantOffers || installs != tc.wantInstalls {
				t.Fatalf("offers=%d installs=%d", offers, installs)
			}
			for _, name := range []string{"one", "two"} {
				value, ok := types.CredentialValue(name)
				if tc.session {
					if !ok || value != "vault-value" {
						t.Fatalf("%s was not resolved after installation", name)
					}
				} else if ok {
					t.Fatalf("skipped credential %s was registered with value %q", name, value)
				}
			}
			if value, _ := types.CredentialValue("unrelated"); value != "still-resolved" {
				t.Fatal("unrelated resolution stopped")
			}
		})
	}
}

func TestMissingBitwardenStillUsesFallback(t *testing.T) {
	withFakes(t, &fakeKeyring{entries: map[string]string{"one": "fallback"}}, false, nil)
	withBWFake(t, nil, map[string]error{"bw:one": ErrBitwardenNotInstalled})
	if err := Resolve([]types.CredentialSpec{{Name: "one", Sources: []string{"bw:one", "keyring"}}}, Options{}); err != nil {
		t.Fatal(err)
	}
	if value, _ := types.CredentialValue("one"); value != "fallback" {
		t.Fatalf("got %q", value)
	}
}

func TestInstalledBitwardenErrorsRemainDistinct(t *testing.T) {
	withFakes(t, &fakeKeyring{}, false, nil)
	withBWFake(t, nil, map[string]error{"bw:one": errors.New("vault is locked")})
	if err := Resolve([]types.CredentialSpec{{Name: "one", Sources: []string{"bw:one"}}}, Options{}); err == nil {
		t.Fatal("installed vault failures must retain existing behavior")
	}
}

func TestExtractBitwarden(t *testing.T) {
	for _, entry := range []string{"bw", "../bw", "unexpected"} {
		t.Run(entry, func(t *testing.T) {
			dir := t.TempDir()
			archive := filepath.Join(dir, "bw.zip")
			f, err := os.Create(archive)
			if err != nil {
				t.Fatal(err)
			}
			z := zip.NewWriter(f)
			w, err := z.Create(entry)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write([]byte("test executable")); err != nil {
				t.Fatal(err)
			}
			if err := z.Close(); err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
			dest := filepath.Join(dir, "installed-bw")
			err = extractBitwarden(archive, dest, "bw")
			if entry != "bw" {
				if err == nil {
					t.Fatal("accepted an unexpected archive entry")
				}
				if _, err := os.Stat(dest); !os.IsNotExist(err) {
					t.Fatal("created a binary from invalid archive")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(dest)
			if err != nil || string(data) != "test executable" {
				t.Fatalf("data=%q err=%v", data, err)
			}
		})
	}
}

func TestBitwardenAssetPlatforms(t *testing.T) {
	for _, tc := range []struct{ os, arch, want string }{
		{"linux", "amd64", "bw-linux-1.0.zip"},
		{"linux", "arm64", "bw-linux-arm64-1.0.zip"},
		{"darwin", "amd64", "bw-macos-1.0.zip"},
		{"darwin", "arm64", "bw-macos-arm64-1.0.zip"},
		{"windows", "amd64", "bw-windows-1.0.zip"},
	} {
		got, err := bitwardenAssetName(tc.os, tc.arch, "1.0")
		if err != nil || got != tc.want {
			t.Fatalf("%s/%s: %q %v", tc.os, tc.arch, got, err)
		}
	}
	if _, err := bitwardenAssetName("linux", "riscv64", "1.0"); err == nil {
		t.Fatal("unsupported platform accepted")
	}
}
