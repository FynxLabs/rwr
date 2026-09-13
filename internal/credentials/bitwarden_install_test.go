package credentials

import (
	"archive/zip"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

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

func TestRuntimeNeverInstalls(t *testing.T) {
	for _, interactive := range []bool{false, true} {
		t.Run(fmt.Sprint(interactive), func(t *testing.T) {
			withFakes(t, &fakeKeyring{}, true, func(string, string) (string, error) { t.Fatal("runtime prompted"); return "", nil })
			old := installBitwarden
			installBitwarden = func() error { t.Fatal("runtime installed"); return nil }
			defer func() { installBitwarden = old }()
			withBWFake(t, nil, map[string]error{"bw:one": ErrBitwardenNotInstalled})
			if err := Resolve([]types.CredentialSpec{{Name: "one", Sources: []string{"bw:one", "prompt"}}}, Options{Interactive: interactive}); err == nil {
				t.Fatal("expected unavailable")
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
	for _, tc := range []struct {
		name, entry string
		mode        os.FileMode
		valid       bool
	}{
		{name: "regular binary", entry: "bw", mode: 0o700, valid: true},
		{name: "parent path", entry: "../bw", mode: 0o700},
		{name: "wrong name", entry: "unexpected", mode: 0o700},
		{name: "directory", entry: "bw", mode: os.ModeDir | 0o700},
		{name: "symlink", entry: "bw", mode: os.ModeSymlink | 0o700},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			archive := filepath.Join(dir, "bw.zip")
			f, err := os.Create(archive)
			if err != nil {
				t.Fatal(err)
			}
			z := zip.NewWriter(f)
			header := &zip.FileHeader{Name: tc.entry}
			header.SetMode(tc.mode)
			w, err := z.CreateHeader(header)
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
			if !tc.valid {
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
	t.Parallel()
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
