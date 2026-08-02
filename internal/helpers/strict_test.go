package helpers

import (
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

// A misspelled key used to produce an empty section and a run that reported
// success having done nothing. These assert it is now an error in every format.
func TestUnmarshalBlueprintStrict_RejectsUnknownKeys(t *testing.T) {
	cases := []struct {
		name   string
		format string
		data   string
		want   string
	}{
		{
			name:   "yaml misspelled section",
			format: types.FormatYAML,
			data:   "packagess:\n  - name: git\n    action: install\n",
			want:   "packagess",
		},
		{
			name:   "yaml misspelled entry key",
			format: types.FormatYAML,
			data:   "packages:\n  - name: git\n    action: install\n    profile: work\n",
			want:   "profile",
		},
		{
			name:   "json misspelled section",
			format: types.FormatJSON,
			data:   `{"packagess":[{"name":"git","action":"install"}]}`,
			want:   "packagess",
		},
		{
			name:   "toml misspelled section",
			format: types.FormatTOML,
			data:   "[[packagess]]\nname = \"git\"\naction = \"install\"\n",
			want:   "packagess",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var data types.PackagesData
			err := UnmarshalBlueprintStrict([]byte(tc.data), tc.format, &data)
			if err == nil {
				t.Fatalf("strict decode accepted unknown key %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not name the offending key %q", err, tc.want)
			}
		})
	}
}

func TestUnmarshalBlueprintStrict_AcceptsKnownKeys(t *testing.T) {
	cases := []struct {
		name   string
		format string
		data   string
	}{
		{"yaml", types.FormatYAML, "packages:\n  - name: git\n    action: install\n    profiles: [work]\n"},
		{"json", types.FormatJSON, `{"packages":[{"name":"git","action":"install"}]}`},
		{"toml", types.FormatTOML, "[[packages]]\nname = \"git\"\naction = \"install\"\n"},
		{"yaml empty document", types.FormatYAML, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var data types.PackagesData
			if err := UnmarshalBlueprintStrict([]byte(tc.data), tc.format, &data); err != nil {
				t.Fatalf("strict decode rejected a valid blueprint: %v", err)
			}
		})
	}
}

// The schema_version probe deliberately reads one key out of a full document, so
// it must stay lenient or every blueprint would fail before its version is known.
func TestUnmarshalBlueprint_LenientForVersionProbe(t *testing.T) {
	var probe types.SchemaVersion
	data := "schema_version: 1\npackages:\n  - name: git\n    action: install\n"
	if err := UnmarshalBlueprint([]byte(data), types.FormatYAML, &probe); err != nil {
		t.Fatalf("version probe rejected a full document: %v", err)
	}
	if probe.DeclaredVersion() != 1 {
		t.Errorf("probe read version %d, want 1", probe.DeclaredVersion())
	}
}
