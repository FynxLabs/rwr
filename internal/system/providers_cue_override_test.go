package system

import (
	"os"
	"path/filepath"
	"testing"
)

// A CUE override is one provider document validated against the embedded
// schema: a good one loads, a bad field fails naming the problem.
func TestLoadProviderDefinition_CUE(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "mypm.cue")
	if err := os.WriteFile(good, []byte(`
name: "mypm"
elevated: false
detection: {
	binary: "mypm"
	files: ["/usr/bin/mypm"]
	distributions: []
}
commands: {
	install: "install"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	provider, err := LoadProviderDefinition(good)
	if err != nil {
		t.Fatalf("good CUE override rejected: %v", err)
	}
	if provider.Name != "mypm" || provider.Commands.Install != "install" {
		t.Fatalf("decoded = %+v", provider)
	}

	bad := filepath.Join(dir, "bad.cue")
	if err := os.WriteFile(bad, []byte(`
name: "bad"
detection: { binary: "bad", files: [], distributions: [] }
commands: { instal: "typo" }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadProviderDefinition(bad); err == nil {
		t.Fatal("schema-violating CUE override accepted")
	}
}
