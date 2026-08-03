package system

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/fynxlabs/rwr/internal/types"
)

// The embedded JSON is exported from providers/cue/ and must decode into
// exactly the types.Provider values the retired TOML definitions produced —
// the gate that makes the CUE migration a refactor rather than a behavior
// change. The TOML originals live on as test data for this comparison.
func TestEmbeddedProviders_RoundTripEqualsTOMLOriginals(t *testing.T) {
	fromJSON, err := LoadEmbeddedProviders()
	if err != nil {
		t.Fatalf("LoadEmbeddedProviders: %v", err)
	}

	tomlDir := filepath.Join("..", "..", "providers", "cue", "testdata", "toml")
	entries, err := os.ReadDir(tomlDir)
	if err != nil {
		t.Fatalf("reading TOML originals: %v", err)
	}

	checked := 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".toml")
		data, err := os.ReadFile(filepath.Join(tomlDir, entry.Name())) // #nosec G304 -- repo's own test data
		if err != nil {
			t.Fatalf("reading %s: %v", entry.Name(), err)
		}

		var config struct {
			Provider types.Provider `toml:"provider"`
		}
		if _, err := toml.Decode(string(data), &config); err != nil {
			t.Fatalf("decoding %s: %v", entry.Name(), err)
		}

		got, ok := fromJSON[name]
		if !ok {
			t.Errorf("provider %s: in the TOML originals but not in the embedded JSON", name)
			continue
		}
		if !reflect.DeepEqual(*got, config.Provider) {
			t.Errorf("provider %s: JSON-derived value differs from the TOML original\n json: %+v\n toml: %+v", name, *got, config.Provider)
		}
		checked++
	}

	if checked != len(fromJSON) {
		t.Errorf("checked %d TOML originals but the embed carries %d providers", checked, len(fromJSON))
	}
	if checked == 0 {
		t.Fatal("no TOML originals found; the gate checked nothing")
	}
}
