package helpers

import (
	"strings"
	"testing"
)

func TestFormatForPath(t *testing.T) {
	for _, tt := range []struct {
		path    string
		want    string
		wantErr bool
	}{
		{"packages.yaml", "yaml", false},
		{"packages.yml", "yaml", false},
		{"dir/files.JSON", "json", false},
		{"a/b/repos.toml", "toml", false},
		// An extensionless file used to panic via filepath.Ext(file)[1:].
		{"Makefile", "", true},
		{"packages.xml", "", true},
	} {
		got, err := FormatForPath(tt.path)
		if tt.wantErr {
			if err == nil {
				t.Errorf("FormatForPath(%q) = %q, want an error", tt.path, got)
			} else if !strings.Contains(err.Error(), tt.path) {
				t.Errorf("FormatForPath(%q) error %q does not name the path", tt.path, err)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("FormatForPath(%q) = (%q, %v), want %q", tt.path, got, err, tt.want)
		}
	}
}

func TestIsBlueprintFile(t *testing.T) {
	for path, want := range map[string]bool{
		"x.yaml": true, "x.yml": true, "x.json": true, "x.toml": true,
		"x.md": false, "x": false, "x.sh": false,
	} {
		if got := IsBlueprintFile(path); got != want {
			t.Errorf("IsBlueprintFile(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestCandidateFilenames(t *testing.T) {
	got := CandidateFilenames("init")
	want := []string{"init.yaml", "init.yml", "init.json", "init.toml"}
	if len(got) != len(want) {
		t.Fatalf("CandidateFilenames = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("CandidateFilenames = %v, want %v", got, want)
		}
	}
}

func TestCanonicalFormat(t *testing.T) {
	for in, want := range map[string]string{
		"yaml": "yaml", "yml": "yaml", ".yaml": "yaml", ".yml": "yaml",
		"json": "json", ".json": "json", "toml": "toml", ".toml": "toml",
		"YAML": "yaml",
	} {
		if got, err := CanonicalFormat(in); err != nil || got != want {
			t.Errorf("CanonicalFormat(%q) = (%q, %v), want %q", in, got, err, want)
		}
	}
	if _, err := CanonicalFormat("xml"); err == nil {
		t.Error("CanonicalFormat(xml) succeeded, want an error")
	}
}
