package helpers

import (
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

const v1Packages = `
packages:
  - name: git
    action: install
`

// A blueprint that declares nothing is v1, in every format. This is the
// compatibility guarantee: versioning cannot change how existing trees are read.
func TestDecodeBlueprint_UndeclaredIsReadAsV1(t *testing.T) {
	bodies := map[string]string{
		"yaml": v1Packages,
		"json": `{"packages":[{"name":"git","action":"install"}]}`,
		"toml": "[[packages]]\nname = \"git\"\naction = \"install\"\n",
	}

	for format, body := range bodies {
		t.Run(format, func(t *testing.T) {
			out, version, err := DecodeBlueprint([]byte(body), format, types.BlueprintTypePackages, 0)
			if err != nil {
				t.Fatalf("DecodeBlueprint: %v", err)
			}
			if version != 1 {
				t.Errorf("resolved version = %d, want 1", version)
			}
			pkgs, ok := out.(*types.PackagesData)
			if !ok {
				t.Fatalf("got %T, want *types.PackagesData", out)
			}
			if len(pkgs.Packages) != 1 || pkgs.Packages[0].Name != "git" {
				t.Errorf("decoded content is wrong: %+v", pkgs.Packages)
			}
		})
	}
}

// The tree-wide version from the init file applies when a file says nothing.
func TestDecodeBlueprint_TreeVersionApplies(t *testing.T) {
	_, version, err := DecodeBlueprint([]byte(v1Packages), "yaml", types.BlueprintTypePackages, 1)
	if err != nil {
		t.Fatalf("DecodeBlueprint: %v", err)
	}
	if version != 1 {
		t.Errorf("resolved version = %d, want 1", version)
	}
}

// A file declaring a version this build cannot read must fail loudly, naming the
// version it wanted. Reading it as v1 would silently misinterpret its fields.
func TestDecodeBlueprint_UnknownVersionIsRejected(t *testing.T) {
	body := "schema_version: 99\npackages:\n  - name: git\n    action: install\n"

	_, version, err := DecodeBlueprint([]byte(body), "yaml", types.BlueprintTypePackages, 0)
	if err == nil {
		t.Fatal("a blueprint declaring schema v99 must be rejected, not read as v1")
	}
	if version != 99 {
		t.Errorf("reported version = %d, want the 99 the file asked for", version)
	}
	for _, want := range []string{"packages", "99", "upgrade rwr"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %q", err, want)
		}
	}
}

// A file's declaration overrides the tree, which is what lets one resource type
// move to a new version while the rest of the tree stays put.
func TestDecodeBlueprint_FileOverridesTree(t *testing.T) {
	body := "schema_version: 1\npackages:\n  - name: git\n    action: install\n"

	// Tree says 99; the file says 1 and wins, so this decodes rather than failing.
	_, version, err := DecodeBlueprint([]byte(body), "yaml", types.BlueprintTypePackages, 99)
	if err != nil {
		t.Fatalf("the file's own version should win over the tree: %v", err)
	}
	if version != 1 {
		t.Errorf("resolved version = %d, want 1 from the file", version)
	}
}

// Every registered blueprint type must round-trip its v1 decoder, so no type is
// left without a schema when versioning is enforced.
func TestSchemaRegistry_CoversEveryBlueprintType(t *testing.T) {
	for _, blueprintType := range []string{
		types.BlueprintTypePackages, types.BlueprintTypeRepositories,
		types.BlueprintTypeFiles, types.BlueprintTypeServices,
		types.BlueprintTypeGit, types.BlueprintTypeScripts,
		types.BlueprintTypeSSHKeys, types.BlueprintTypeFonts,
		types.BlueprintTypeUsers, types.BlueprintTypeConfiguration,
	} {
		if !types.HasSchemaVariant(blueprintType, 1) {
			t.Errorf("%s has no v1 schema registered", blueprintType)
		}
	}
}
