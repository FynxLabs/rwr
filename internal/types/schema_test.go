package types

import (
	"strings"
	"testing"
)

func TestResolveSchemaVersion(t *testing.T) {
	tests := []struct {
		name         string
		fileDeclared int
		treeDeclared int
		want         int
	}{
		{
			name: "nothing declared falls back to the default",
			want: DefaultSchemaVersion,
		},
		{
			name:         "tree version applies when the file says nothing",
			treeDeclared: 2,
			want:         2,
		},
		{
			name:         "the file wins over the tree",
			fileDeclared: 2,
			treeDeclared: 1,
			want:         2,
		},
		{
			name:         "the file wins even when older than the tree",
			fileDeclared: 1,
			treeDeclared: 2,
			want:         1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveSchemaVersion(tt.fileDeclared, tt.treeDeclared); got != tt.want {
				t.Errorf("ResolveSchemaVersion(%d, %d) = %d, want %d",
					tt.fileDeclared, tt.treeDeclared, got, tt.want)
			}
		})
	}
}

// Absence must mean v1. Every blueprint written before versioning existed
// declares nothing, so if absence meant anything else, adding versioning would
// break every tree in the wild.
func TestResolveSchemaVersion_UndeclaredIsV1(t *testing.T) {
	if got := ResolveSchemaVersion(0, 0); got != 1 {
		t.Errorf("an undeclared blueprint resolves to %d, want 1", got)
	}
}

func TestValidateSchemaVersion(t *testing.T) {
	if err := ValidateSchemaVersion(BlueprintTypePackages, 1); err != nil {
		t.Errorf("v1 packages should be supported: %v", err)
	}

	err := ValidateSchemaVersion(BlueprintTypePackages, 2)
	if err == nil {
		t.Fatal("v2 packages is not implemented yet and must be rejected")
	}
	// The message has to say what went wrong and what to do; a version this build
	// cannot read means the blueprint would be misread, not merely unsupported.
	for _, want := range []string{"packages", "2", "supports 1", "upgrade rwr"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %q", err, want)
		}
	}

	if err := ValidateSchemaVersion("nonsense", 1); err == nil {
		t.Error("an unknown blueprint type should be rejected")
	}
	if err := ValidateSchemaVersion(BlueprintTypePackages, 0); err == nil {
		t.Error("version 0 should be rejected as invalid")
	}
	if err := ValidateSchemaVersion(BlueprintTypePackages, -1); err == nil {
		t.Error("a negative version should be rejected")
	}
}

func TestValidateTreeSchemaVersion(t *testing.T) {
	if err := ValidateTreeSchemaVersion(0); err != nil {
		t.Errorf("an undeclared tree version is valid: %v", err)
	}
	if err := ValidateTreeSchemaVersion(1); err != nil {
		t.Errorf("v1 is supported everywhere: %v", err)
	}

	err := ValidateTreeSchemaVersion(2)
	if err == nil {
		t.Fatal("a tree-wide v2 must be rejected while no type implements v2")
	}
	// A tree-wide version applies to every type, so when only some types have a
	// v2 the fix is to declare it per file — the error should say so.
	if !strings.Contains(err.Error(), "per blueprint file") {
		t.Errorf("error should suggest declaring per file, got: %v", err)
	}
	if err := ValidateTreeSchemaVersion(-1); err == nil {
		t.Error("a negative tree version should be rejected")
	}
}

// The point of per-type versioning: moving one resource to v2 must not require
// moving anything else. This asserts the registry is genuinely per type rather
// than one shared list.
func TestSupportedSchemaVersions_AreIndependentPerType(t *testing.T) {
	for _, blueprintType := range []string{
		BlueprintTypePackages, BlueprintTypeFiles, BlueprintTypeServices,
		BlueprintTypeGit, BlueprintTypeScripts, BlueprintTypeSSHKeys,
		BlueprintTypeFonts, BlueprintTypeUsers, BlueprintTypeConfiguration,
		BlueprintTypeRepositories,
	} {
		versions := SupportedSchemaVersions(blueprintType)
		if len(versions) == 0 {
			t.Errorf("%s declares no supported schema versions", blueprintType)
			continue
		}
		if versions[0] != 1 {
			t.Errorf("%s should still support v1 for backwards compatibility, got %v",
				blueprintType, versions)
		}
	}
}
