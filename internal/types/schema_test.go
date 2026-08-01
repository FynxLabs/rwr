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
			name: "nothing declared falls back to the latest supported",
			want: LatestSchemaVersion(BlueprintTypePackages),
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
			got := ResolveSchemaVersion(tt.fileDeclared, tt.treeDeclared, BlueprintTypePackages)
			if got != tt.want {
				t.Errorf("ResolveSchemaVersion(%d, %d) = %d, want %d",
					tt.fileDeclared, tt.treeDeclared, got, tt.want)
			}
		})
	}
}

// An undeclared blueprint is read as the latest schema that type supports, so a
// blueprint written today gets today's format without boilerplate.
//
// This asserts against LatestSchemaVersion rather than a literal, so it keeps
// testing the intent once a type gains a v2 — a hardcoded 1 here would silently
// keep passing while the behaviour it describes had changed.
func TestResolveSchemaVersion_UndeclaredIsLatest(t *testing.T) {
	for _, blueprintType := range []string{
		BlueprintTypePackages, BlueprintTypeFiles, BlueprintTypeServices,
	} {
		want := LatestSchemaVersion(blueprintType)
		if got := ResolveSchemaVersion(0, 0, blueprintType); got != want {
			t.Errorf("%s: undeclared resolved to %d, want the latest %d",
				blueprintType, got, want)
		}
	}
}

// A declaration always wins over the fallback, which is what pins a tree to a
// version while newer ones exist.
func TestResolveSchemaVersion_DeclarationPinsAgainstLatest(t *testing.T) {
	if got := ResolveSchemaVersion(1, 0, BlueprintTypePackages); got != 1 {
		t.Errorf("a file declaring v1 resolved to %d, want 1", got)
	}
	if got := ResolveSchemaVersion(0, 1, BlueprintTypePackages); got != 1 {
		t.Errorf("a tree declaring v1 resolved to %d, want 1", got)
	}
}

func TestLatestSchemaVersion(t *testing.T) {
	for _, blueprintType := range []string{BlueprintTypePackages, BlueprintTypeFiles} {
		latest := LatestSchemaVersion(blueprintType)
		versions := SupportedSchemaVersions(blueprintType)
		if latest != versions[len(versions)-1] {
			t.Errorf("%s: latest = %d, but supported versions are %v",
				blueprintType, latest, versions)
		}
	}
	// An unknown type must not report 0, which would resolve to an invalid version.
	if got := LatestSchemaVersion("nonsense"); got != DefaultSchemaVersion {
		t.Errorf("unknown type reported latest %d, want %d", got, DefaultSchemaVersion)
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

// withExtraVersion temporarily teaches the registry that a type supports another
// version, so the resolution rules can be exercised before any real v2 exists.
func withExtraVersion(t *testing.T, blueprintType string, version int) {
	t.Helper()
	original := append([]int(nil), supportedSchemaVersions[blueprintType]...)
	supportedSchemaVersions[blueprintType] = append(original, version)
	t.Cleanup(func() { supportedSchemaVersions[blueprintType] = original })
}

// The real point of "undeclared means latest": when packages gains a v2, a
// packages blueprint that declares nothing is read as v2 — while files, which did
// not change, stays where it is. Today latest is 1 everywhere, so this simulates
// the v2 to test the rule rather than the current numbers.
func TestResolveSchemaVersion_UndeclaredFollowsANewVersion(t *testing.T) {
	withExtraVersion(t, BlueprintTypePackages, 2)

	if got := ResolveSchemaVersion(0, 0, BlueprintTypePackages); got != 2 {
		t.Errorf("undeclared packages resolved to %d, want the new latest 2", got)
	}
	// A type that did not gain a version is unaffected — this is what keeps a
	// breaking change contained to one resource.
	if got := ResolveSchemaVersion(0, 0, BlueprintTypeFiles); got != 1 {
		t.Errorf("undeclared files resolved to %d, want 1 — files did not change", got)
	}
	// Declaring a version still pins, which is how a tree stays on v1 after v2
	// ships.
	if got := ResolveSchemaVersion(1, 0, BlueprintTypePackages); got != 1 {
		t.Errorf("packages pinned to v1 resolved to %d, want 1", got)
	}
	if got := ResolveSchemaVersion(0, 1, BlueprintTypePackages); got != 1 {
		t.Errorf("packages under a v1 tree resolved to %d, want 1", got)
	}
}
