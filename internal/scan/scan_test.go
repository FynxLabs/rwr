package scan

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
)

func fakeProvider(t *testing.T, name, script string) *types.Provider {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "fakelist")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	return &types.Provider{Name: name, BinPath: bin}
}

// The explicit query wins over the full list, and its absence is marked.
func TestPackages_ExplicitPreferred(t *testing.T) {
	p := fakeProvider(t, "fake", `case "$1" in explicit) printf 'chosen\n';; all) printf 'chosen\ndep1\ndep2\n';; esac`)
	p.Commands.List = "all"
	p.Commands.ListExplicit = "explicit"

	results := Packages(map[string]*types.Provider{"fake": p})
	if len(results) != 1 || results[0].Unfiltered {
		t.Fatalf("results = %+v", results)
	}
	if len(results[0].Names) != 1 || results[0].Names[0] != "chosen" {
		t.Fatalf("names = %v, want the explicit set", results[0].Names)
	}

	p.Commands.ListExplicit = ""
	results = Packages(map[string]*types.Provider{"fake": p})
	if !results[0].Unfiltered || len(results[0].Names) != 3 {
		t.Fatalf("fallback = %+v, want the full set marked unfiltered", results[0])
	}
}

// A scan executes only list verbs - a provider whose other commands are
// traps proves it.
func TestPackages_OnlyListVerbsRun(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "mutated")
	trap := filepath.Join(dir, "trap")
	if err := os.WriteFile(trap, []byte("#!/bin/sh\ncase \"$1\" in list) printf 'x\\n';; *) touch "+marker+";; esac\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := &types.Provider{Name: "trap", BinPath: trap}
	p.Commands.List = "list"
	p.Commands.Install = "install"
	p.Commands.Remove = "remove"

	Packages(map[string]*types.Provider{"trap": p})
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("scan executed a non-list command")
	}
}

// Real-world list output: cargo indents each crate's binaries, snap prints
// a header row. Neither is a package.
func TestRunListCommand_SkipsDetailAndHeaders(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "fakelist")
	script := "#!/bin/sh\nprintf 'Name  Version  Rev\\nripgrep v14.1.0:\\n    rg\\nzoxide v0.9.4:\\n    zoxide\\n'\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	p := &types.Provider{Name: "fake", BinPath: bin}
	names := RunListCommand(p, "list")
	if strings.Join(names, ",") != "ripgrep,zoxide" {
		t.Fatalf("names = %v, want [ripgrep zoxide]", names)
	}
}

func TestConfigs_KnownNoiseAndSecrets(t *testing.T) {
	home := t.TempDir()
	for _, p := range []string{".bashrc", ".config/helix/config.toml", ".config/pulse/cookie", ".config/gh-credentials/token"} {
		full := filepath.Join(home, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	results := Configs(home, false)
	byRel := map[string]ConfigResult{}
	for _, r := range results {
		byRel[r.Rel] = r
	}
	if r, ok := byRel[".bashrc"]; !ok || !r.Known {
		t.Fatalf(".bashrc = %+v", r)
	}
	if _, ok := byRel[filepath.Join(".config", "helix")]; !ok {
		t.Fatal("app config missing")
	}
	if _, ok := byRel[filepath.Join(".config", "pulse")]; ok {
		t.Fatal("noise not excluded")
	}
	if _, ok := byRel[filepath.Join(".config", "gh-credentials")]; ok {
		t.Fatal("secret-shaped entry not excluded by default")
	}

	all := Configs(home, true)
	found := false
	for _, r := range all {
		if r.Rel == filepath.Join(".config", "pulse") {
			found = true
		}
	}
	if !found {
		t.Fatal("includeAll did not recover the excluded entry")
	}
}

func TestParseEnabledUnits_VendorPresetSkipped(t *testing.T) {
	out := "sshd.service enabled disabled\n" + // operator enabled (preset says off)
		"cups.service enabled enabled\n" + // vendor preset - not operator intent
		"tailscaled.service enabled disabled\n"
	units := parseEnabledUnits(out)
	if strings.Join(units, ",") != "sshd,tailscaled" {
		t.Fatalf("units = %v", units)
	}
}

func TestGitCheckouts(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	root := t.TempDir()
	repo := filepath.Join(root, "org", "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"remote", "add", "origin", "https://example.com/org/repo.git"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	checkouts := GitCheckouts([]string{root})
	if len(checkouts) != 1 || checkouts[0].URL != "https://example.com/org/repo.git" {
		t.Fatalf("checkouts = %+v", checkouts)
	}
}

// Emitted blocks strict-decode against the blueprint schema in every format.
func TestEmit_RoundTrips(t *testing.T) {
	packages := []PackageResult{{Provider: "pacman", Names: []string{"git", "vim"}}}
	for _, format := range []string{"yaml", "json", "toml", "cue"} {
		block, err := EmitPackages(packages, format)
		if err != nil {
			t.Fatalf("%s: %v", format, err)
		}
		decodeFormat := format
		if format == "cue" {
			decodeFormat = "json" // JSON-form CUE
		}
		var d types.PackagesData
		if err := helpers.DecodeBlueprintInto(block, decodeFormat, types.BlueprintTypePackages, 0, &d); err != nil {
			t.Fatalf("%s block does not strict-decode: %v\n%s", format, err, block)
		}
		if len(d.Packages) != 1 || d.Packages[0].PackageManager != "pacman" {
			t.Fatalf("%s round-trip = %+v", format, d.Packages)
		}
	}

	home := "/home/x"
	git, err := EmitGit([]GitCheckout{{Path: "/home/x/git/org/repo", URL: "u"}}, home, "yaml")
	if err != nil || !strings.Contains(string(git), "{{ .User.home }}/git/org/repo") {
		t.Fatalf("git emission not home-templated (%v):\n%s", err, git)
	}
	files, err := EmitConfigs([]ConfigResult{{Path: "/home/x/.bashrc", Rel: ".bashrc"}}, home, "yaml")
	if err != nil || !strings.Contains(string(files), "src/.bashrc") {
		t.Fatalf("config emission (%v):\n%s", err, files)
	}
}

func TestSecretShaped(t *testing.T) {
	for rel, want := range map[string]bool{
		".ssh/config": true, ".gnupg": true, ".netrc": true,
		".config/helix": false, ".bashrc": false,
	} {
		if SecretShaped(rel) != want {
			t.Errorf("SecretShaped(%q) != %v", rel, want)
		}
	}
}
