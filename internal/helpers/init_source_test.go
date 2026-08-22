package helpers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveInitSource_LocalForms(t *testing.T) {
	dir := t.TempDir()
	initFile := filepath.Join(dir, "init.yaml")
	if err := os.WriteFile(initFile, []byte("blueprints: {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(dir, "custom.toml")
	if err := os.WriteFile(other, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("existing file wins as-is", func(t *testing.T) {
		got, err := ResolveInitSource(other)
		if err != nil || got != other {
			t.Fatalf("ResolveInitSource(%q) = %q, %v", other, got, err)
		}
	})

	t.Run("directory probes init names", func(t *testing.T) {
		got, err := ResolveInitSource(dir)
		if err != nil || got != initFile {
			t.Fatalf("ResolveInitSource(dir) = %q, %v; want %q", got, err, initFile)
		}
	})

	t.Run("empty ref probes the CWD", func(t *testing.T) {
		orig, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := os.Chdir(orig); err != nil {
				t.Fatal(err)
			}
		}()

		got, err := ResolveInitSource("")
		if err != nil || filepath.Base(got) != "init.yaml" {
			t.Fatalf("ResolveInitSource(\"\") = %q, %v", got, err)
		}
	})

	t.Run("directory without init file errors", func(t *testing.T) {
		if _, err := ResolveInitSource(t.TempDir()); err == nil {
			t.Fatal("want an error for a directory with no init file")
		}
	})

	// Existence on disk wins over shorthand interpretation.
	t.Run("local directory shaped like a shorthand", func(t *testing.T) {
		nested := filepath.Join(dir, "owner", "repo")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(nested, "init.yml"), []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
		orig, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := os.Chdir(orig); err != nil {
				t.Fatal(err)
			}
		}()

		got, err := ResolveInitSource("owner/repo")
		if err != nil || got != filepath.Join("owner", "repo", "init.yml") {
			t.Fatalf("ResolveInitSource(owner/repo with local dir) = %q, %v", got, err)
		}
	})
}

func TestResolveInitSource_URLs(t *testing.T) {
	origClone := cloneManifestRepo
	cloneManifestRepo = func(owner, repo string) (string, error) {
		return filepath.Join("cloned", owner, repo, "init.toml"), nil
	}
	defer func() { cloneManifestRepo = origClone }()

	for _, tt := range []struct {
		name    string
		ref     string
		want    string
		wantErr string
	}{
		{
			name: "github repository URL discovers its init",
			ref:  "https://github.com/FynxLabs/blueprints",
			want: filepath.Join("cloned", "FynxLabs", "blueprints", "init.toml"),
		},
		{
			name: "github clone URL discovers its init",
			ref:  "https://github.com/FynxLabs/blueprints.git/",
			want: filepath.Join("cloned", "FynxLabs", "blueprints", "init.toml"),
		},
		{
			name: "raw github init clones its repository",
			ref:  "https://raw.githubusercontent.com/FynxLabs/blueprints/refs/heads/main/macOS/init.cue",
			want: filepath.Join("cloned", "FynxLabs", "blueprints", "init.toml"),
		},
		{
			name: "raw https passes through",
			ref:  "https://example.com/machines/init.yaml",
			want: "https://example.com/machines/init.yaml",
		},
		{
			name: "github blob init clones its repository",
			ref:  "https://github.com/FynxLabs/blueprints/blob/main/init.yaml",
			want: filepath.Join("cloned", "FynxLabs", "blueprints", "init.toml"),
		},
		{
			name:    "malformed blob URL errors instead of panicking",
			ref:     "https://blob/x",
			wantErr: "cannot rewrite",
		},
		{
			name:    "plain http is refused",
			ref:     "http://example.com/init.yaml",
			wantErr: "http://",
		},
		{
			name:    "other schemes are refused",
			ref:     "ftp://example.com/init.yaml",
			wantErr: "unsupported init source scheme",
		},
		{
			name:    "nonsense is named",
			ref:     "definitely not a source",
			wantErr: "not an existing path",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveInitSource(tt.ref)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("ResolveInitSource(%q) = %q, %v; want %q", tt.ref, got, err, tt.want)
			}
		})
	}
}

func TestResolveInitSource_Shorthands(t *testing.T) {
	// Serves as raw.githubusercontent.com: init.toml exists at HEAD, init.yaml
	// only on the "dev" ref, and an explicit path exists wherever named.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/FynxLabs/blueprints/HEAD/init.toml",
			"/FynxLabs/blueprints/dev/init.yaml",
			"/FynxLabs/blueprints/v1.2/machines/laptop.yaml":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	origBase := rawGitHubBase
	rawGitHubBase = server.URL
	defer func() { rawGitHubBase = origBase }()

	for _, tt := range []struct {
		name    string
		ref     string
		want    string
		wantErr string
	}{
		{
			name: "owner/repo probes the default branch",
			ref:  "FynxLabs/blueprints",
			want: server.URL + "/FynxLabs/blueprints/HEAD/init.toml",
		},
		{
			name: "owner/repo@ref probes that ref",
			ref:  "FynxLabs/blueprints@dev",
			want: server.URL + "/FynxLabs/blueprints/dev/init.yaml",
		},
		{
			name: "explicit path needs no probing",
			ref:  "FynxLabs/blueprints/machines/laptop.yaml@v1.2",
			want: server.URL + "/FynxLabs/blueprints/v1.2/machines/laptop.yaml",
		},
		{
			name:    "repo without an init file names what it tried",
			ref:     "FynxLabs/empty",
			wantErr: "tried init.yaml",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveInitSource(tt.ref)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("ResolveInitSource(%q) = %q, %v; want %q", tt.ref, got, err, tt.want)
			}
		})
	}
}
