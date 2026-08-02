package processors

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/exectest"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// Schema versioning is only worth anything if the processors enforce it. It was
// written, tested at the helper level, and then wired to nothing: every processor
// called UnmarshalBlueprint directly, so a blueprint declaring a schema this build
// cannot read was decoded as the current schema and applied to the machine.
//
// These tests call the real processors with real blueprint bytes and a recording
// executor. Each one fails if the enforcement is removed again.

func newTestOSInfo() *types.OSInfo {
	osInfo := &types.OSInfo{}
	osInfo.PackageManager.Default = types.PackageManagerInfo{Name: "pacman", Bin: "/usr/bin/pacman"}
	osInfo.PackageManager.Managers = map[string]types.PackageManagerInfo{
		"pacman": osInfo.PackageManager.Default,
	}
	return osInfo
}

// useTestProvider installs a provider named "pacman" whose binary is one every
// platform has, so these tests assert how rwr builds a command rather than
// asserting what the runner happens to have installed.
//
// Without it the tests pass on a machine with pacman and fail on one without —
// which is exactly what happened: green on an Arch workstation, red on CI's
// Ubuntu runner.
func useTestProvider(t *testing.T) {
	t.Helper()

	bin := "sh"
	if runtime.GOOS == types.OSWindows {
		bin = "cmd"
	}

	restore := system.SetProvidersForTest(map[string]*types.Provider{
		"pacman": {
			Name:     "pacman",
			Elevated: true,
			Detection: types.DetectionConfig{
				Binary:        bin,
				Distributions: []string{types.OSLinux, types.OSDarwin, types.OSWindows},
			},
			Commands: types.CommandConfig{
				Install: "-Sy --noconfirm",
				Remove:  "-Rns --noconfirm",
				Update:  "-Syu --noconfirm",
				Clean:   "-Sc --noconfirm",
				List:    "-Q",
				Search:  "-Ss",
			},
		},
	})
	t.Cleanup(restore)
}

// unsupportedVersionCases covers every blueprint type a processor decodes. Each
// blueprint is otherwise valid, so the only reason to refuse it is the version.
var unsupportedVersionCases = []struct {
	name      string
	blueprint string
	run       func(data []byte, osInfo *types.OSInfo, init *types.InitConfig) error
}{
	{
		name:      "packages",
		blueprint: "schema_version: 99\npackages:\n  - name: git\n    action: install\n    package_manager: pacman\n",
		run: func(data []byte, osInfo *types.OSInfo, init *types.InitConfig) error {
			return ProcessPackages(data, nil, "", "yaml", osInfo, init)
		},
	},
	{
		name:      "services",
		blueprint: "schema_version: 99\nservices:\n  - name: sshd\n    action: enable\n",
		run: func(data []byte, osInfo *types.OSInfo, init *types.InitConfig) error {
			return ProcessServices(data, "", "yaml", osInfo, init)
		},
	},
	{
		name:      "git",
		blueprint: "schema_version: 99\nrepositories:\n  - name: dots\n    url: https://example.invalid/d.git\n    path: /tmp/rwr-test-dots\n",
		run: func(data []byte, osInfo *types.OSInfo, init *types.InitConfig) error {
			return ProcessGitRepositories(data, "", "yaml", init)
		},
	},
	{
		name:      "scripts",
		blueprint: "schema_version: 99\nscripts:\n  - name: hello\n    exec: bash\n    content: \"echo hi\"\n",
		run: func(data []byte, osInfo *types.OSInfo, init *types.InitConfig) error {
			return ProcessScripts(data, ".", "yaml", osInfo, init)
		},
	},
	{
		name:      "users",
		blueprint: "schema_version: 99\nusers:\n  - name: tester\n    action: create\n",
		run: func(data []byte, osInfo *types.OSInfo, init *types.InitConfig) error {
			return ProcessUsers(data, "", "yaml", init)
		},
	},
	{
		name:      "configuration",
		blueprint: "schema_version: 99\nconfiguration:\n  - name: theme\n    type: dconf\n    action: set\n    key: /org/gnome/theme\n    value: dark\n",
		run: func(data []byte, osInfo *types.OSInfo, init *types.InitConfig) error {
			return ProcessConfiguration(data, ".", "yaml", init)
		},
	},
}

func TestProcessors_RefuseUnsupportedSchemaVersion(t *testing.T) {
	for _, tc := range unsupportedVersionCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := exectest.New()
			defer system.SetExecutor(rec)()

			init := &types.InitConfig{}
			init.Init.Location = t.TempDir()

			err := tc.run([]byte(tc.blueprint), newTestOSInfo(), init)
			if err == nil {
				t.Fatalf("schema_version 99 was accepted; blueprint applied with %d command(s): %+v",
					len(rec.Calls), rec.Calls)
			}
			if !strings.Contains(err.Error(), "99") {
				t.Errorf("error should name the requested version, got: %v", err)
			}
			if len(rec.Calls) != 0 {
				t.Errorf("refused blueprint still ran %d command(s): %+v", len(rec.Calls), rec.Calls)
			}
		})
	}
}

// A supported version must still work, or the check above passes for the wrong
// reason.
func TestProcessors_AcceptSupportedSchemaVersion(t *testing.T) {
	useTestProvider(t)
	rec := exectest.New()
	defer system.SetExecutor(rec)()

	init := &types.InitConfig{}
	init.Init.Location = t.TempDir()

	blueprint := []byte("schema_version: 1\npackages:\n  - name: git\n    action: install\n    package_manager: pacman\n")
	if err := ProcessPackages(blueprint, nil, t.TempDir(), "yaml", newTestOSInfo(), init); err != nil {
		t.Fatalf("schema_version 1 should be accepted: %v", err)
	}
	if len(rec.Calls) == 0 {
		t.Fatal("supported blueprint issued no command")
	}
}

// No declaration resolves to the latest supported version, so an undeclared
// blueprint keeps working without boilerplate.
func TestProcessors_UndeclaredVersionIsAccepted(t *testing.T) {
	useTestProvider(t)
	rec := exectest.New()
	defer system.SetExecutor(rec)()

	init := &types.InitConfig{}
	init.Init.Location = t.TempDir()

	blueprint := []byte("packages:\n  - name: git\n    action: install\n    package_manager: pacman\n")
	if err := ProcessPackages(blueprint, nil, t.TempDir(), "yaml", newTestOSInfo(), init); err != nil {
		t.Fatalf("undeclared version should resolve to latest: %v", err)
	}
	if len(rec.Calls) == 0 {
		t.Fatal("undeclared blueprint issued no command")
	}
}

// The tree-wide version from the init file applies to a blueprint that declares
// none of its own.
func TestProcessors_TreeVersionApplies(t *testing.T) {
	rec := exectest.New()
	defer system.SetExecutor(rec)()

	init := &types.InitConfig{}
	init.Init.Location = t.TempDir()
	init.Init.SchemaVersion = 99

	blueprint := []byte("packages:\n  - name: git\n    action: install\n    package_manager: pacman\n")
	err := ProcessPackages(blueprint, nil, t.TempDir(), "yaml", newTestOSInfo(), init)
	if err == nil {
		t.Fatalf("tree version 99 was accepted; %d command(s) ran: %+v", len(rec.Calls), rec.Calls)
	}
	if len(rec.Calls) != 0 {
		t.Errorf("refused blueprint still ran %d command(s)", len(rec.Calls))
	}
}

// A file's own declaration overrides the tree's. This is what makes a
// single-resource migration possible, so it has to hold in both directions.
func TestProcessors_FileVersionOverridesTree(t *testing.T) {
	useTestProvider(t)
	rec := exectest.New()
	defer system.SetExecutor(rec)()

	init := &types.InitConfig{}
	init.Init.Location = t.TempDir()
	init.Init.SchemaVersion = 99 // unreadable tree-wide

	blueprint := []byte("schema_version: 1\npackages:\n  - name: git\n    action: install\n    package_manager: pacman\n")
	if err := ProcessPackages(blueprint, nil, t.TempDir(), "yaml", newTestOSInfo(), init); err != nil {
		t.Fatalf("file declaration should override the tree version: %v", err)
	}
	if len(rec.Calls) == 0 {
		t.Fatal("blueprint pinned to a supported version issued no command")
	}
}

// Imported files are blueprints too, and an import is exactly where a file
// written for a different rwr arrives from somebody else's repository.
func TestProcessors_ImportedFileVersionIsEnforced(t *testing.T) {
	rec := exectest.New()
	defer system.SetExecutor(rec)()

	dir := t.TempDir()
	imported := filepath.Join(dir, "extra.yaml")
	if err := os.WriteFile(imported,
		[]byte("schema_version: 99\npackages:\n  - name: curl\n    action: install\n    package_manager: pacman\n"),
		0o600); err != nil {
		t.Fatal(err)
	}

	init := &types.InitConfig{}
	init.Init.Location = dir

	blueprint := []byte("packages:\n  - import: extra.yaml\n")
	err := ProcessPackages(blueprint, nil, t.TempDir(), "yaml", newTestOSInfo(), init)
	if err == nil {
		t.Fatalf("imported blueprint with version 99 was accepted; %d command(s) ran: %+v",
			len(rec.Calls), rec.Calls)
	}
	if len(rec.Calls) != 0 {
		t.Errorf("refused import still ran %d command(s)", len(rec.Calls))
	}
}

// The error has to say which version was asked for and which are supported —
// "invalid blueprint" sends the operator looking in the wrong place.
func TestProcessors_UnsupportedVersionErrorIsActionable(t *testing.T) {
	defer system.SetExecutor(exectest.New())()

	init := &types.InitConfig{}
	init.Init.Location = t.TempDir()

	blueprint := []byte("schema_version: 7\npackages:\n  - name: git\n    action: install\n")
	err := ProcessPackages(blueprint, nil, t.TempDir(), "yaml", newTestOSInfo(), init)
	if err == nil {
		t.Fatal("expected refusal")
	}
	for _, want := range []string{"packages", "7", "supports 1", "upgrade rwr"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error missing %q: %v", want, err)
		}
	}
}
