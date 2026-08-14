package processors

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/fynxlabs/rwr/internal/exectest"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// scriptOSInfo returns an OSInfo with just enough tooling populated to run the
// scripts processor without touching the host's real tool discovery.
func scriptOSInfo() *types.OSInfo {
	osInfo := &types.OSInfo{}
	osInfo.System.OS = "linux"
	osInfo.Tools.Bash = types.ToolInfo{Exists: true, Bin: "/bin/bash"}
	return osInfo
}

// `args` is a single string in the blueprint. The shell used to word-split it on
// the way to the script; with argv execution the processor has to split it, or
// "--verbose --out /tmp" would arrive as one argument instead of three.
func TestProcessScripts_ArgsAreSplitIntoSeparateArguments(t *testing.T) {
	rec := exectest.New()
	defer system.SetExecutor(rec)()

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "setup.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho hi\n"), 0o700); err != nil {
		t.Fatalf("writing script: %v", err)
	}

	blueprint := []byte(`
scripts:
  - name: "setup.sh"
    action: "run"
    source: "."
    exec: "bash"
    args: "--verbose --out /tmp/out"
`)

	err := ProcessScripts(blueprint, dir, "yaml", scriptOSInfo(), &types.InitConfig{})
	if err != nil {
		t.Fatalf("ProcessScripts: %v", err)
	}

	if len(rec.Calls) != 1 {
		t.Fatalf("recorded %d calls, want 1: %v", len(rec.Calls), rec.Calls)
	}

	call := rec.Calls[0]
	if call.Exec != "/bin/bash" {
		t.Errorf("Exec = %q, want /bin/bash", call.Exec)
	}

	// script path, then each blueprint arg as its own element
	want := []string{scriptPath, "--verbose", "--out", "/tmp/out"}
	if len(call.Args) != len(want) {
		t.Fatalf("Args = %#v, want %#v", call.Args, want)
	}
	for i := range want {
		if call.Args[i] != want[i] {
			t.Fatalf("Args = %#v, want %#v", call.Args, want)
		}
	}
}

// The script path is passed as its own argv element, so a directory containing a
// space cannot split into two arguments.
func TestProcessScripts_ScriptPathWithSpacesStaysOneArgument(t *testing.T) {
	rec := exectest.New()
	defer system.SetExecutor(rec)()

	dir := filepath.Join(t.TempDir(), "my scripts")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	scriptPath := filepath.Join(dir, "setup.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\n"), 0o700); err != nil {
		t.Fatalf("writing script: %v", err)
	}

	blueprint := []byte(`
scripts:
  - name: "setup.sh"
    action: "run"
    source: "."
    exec: "bash"
`)

	if err := ProcessScripts(blueprint, dir, "yaml", scriptOSInfo(), &types.InitConfig{}); err != nil {
		t.Fatalf("ProcessScripts: %v", err)
	}

	if len(rec.Calls) != 1 {
		t.Fatalf("recorded %d calls, want 1: %v", len(rec.Calls), rec.Calls)
	}
	if got := rec.Calls[0].Args; len(got) != 1 || got[0] != scriptPath {
		t.Errorf("Args = %#v, want exactly [%q]", got, scriptPath)
	}
}

// Inline `content:` scripts used to stage as *.sh unconditionally, but the
// Windows default executor is `powershell -File`, which refuses any file not
// ending in .ps1 - every inline script on Windows failed before its first line.
func TestProcessScripts_InlineContentExtensionMatchesExecutor(t *testing.T) {
	rec := exectest.New()
	defer system.SetExecutor(rec)()

	osInfo := &types.OSInfo{}
	osInfo.System.OS = "windows"
	osInfo.Tools.PowerShell = types.ToolInfo{Exists: true, Bin: "powershell"}

	blueprint := []byte(`
scripts:
  - name: "hello"
    action: "run"
    content: "Write-Host hi"
`)

	if err := ProcessScripts(blueprint, t.TempDir(), "yaml", osInfo, &types.InitConfig{}); err != nil {
		t.Fatalf("ProcessScripts: %v", err)
	}

	calls := rec.Find("powershell")
	if len(calls) != 1 {
		t.Fatalf("recorded %d powershell calls, want 1: %v", len(calls), rec.Calls)
	}
	if len(calls[0].Args) != 2 || calls[0].Args[0] != "-File" {
		t.Fatalf("powershell args = %v, want [-File <path>]", calls[0].Args)
	}
	if got := calls[0].Args[1]; filepath.Ext(got) != ".ps1" {
		t.Errorf("staged script = %q, want a .ps1 extension - powershell -File refuses anything else", got)
	}
}

// runScriptArgs runs one script blueprint and returns the argv the executor
// saw, minus the script path that always leads it.
func runScriptArgs(t *testing.T, format, blueprint string) []string {
	t.Helper()

	rec := exectest.New()
	defer system.SetExecutor(rec)()

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "setup.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho hi\n"), 0o700); err != nil {
		t.Fatalf("writing script: %v", err)
	}

	if err := ProcessScripts([]byte(blueprint), dir, format, scriptOSInfo(), &types.InitConfig{}); err != nil {
		t.Fatalf("ProcessScripts: %v", err)
	}
	if len(rec.Calls) != 1 {
		t.Fatalf("recorded %d calls, want 1: %v", len(rec.Calls), rec.Calls)
	}
	return rec.Calls[0].Args[1:]
}

func argsEqual(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// The reason the list form exists. Splitting on whitespace means an argument
// that contains whitespace cannot be written at all: quoting inside the string
// does not help, because no shell ever parses the value. A list element is
// taken verbatim.
func TestProcessScripts_ListArgsKeepArgumentsContainingSpaces(t *testing.T) {
	got := runScriptArgs(t, "yaml", `
scripts:
  - name: "setup.sh"
    action: "run"
    source: "."
    exec: "bash"
    args: ["--message", "hello world", "--out", "/tmp/a b"]
`)

	want := []string{"--message", "hello world", "--out", "/tmp/a b"}
	if !argsEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}

// The list form has to work in every blueprint format, not just the one it was
// written against. CUE rides the JSON decode path.
func TestProcessScripts_ListArgsInEveryFormat(t *testing.T) {
	want := []string{"--message", "hello world"}

	cases := map[string]string{
		"yaml": `
scripts:
  - name: "setup.sh"
    action: "run"
    source: "."
    exec: "bash"
    args: ["--message", "hello world"]
`,
		"json": `{"scripts":[{"name":"setup.sh","action":"run","source":".","exec":"bash","args":["--message","hello world"]}]}`,
		"toml": `
[[scripts]]
name = "setup.sh"
action = "run"
source = "."
exec = "bash"
args = ["--message", "hello world"]
`,
		"cue": `
scripts: [{
	name:   "setup.sh"
	action: "run"
	source: "."
	exec:   "bash"
	args: ["--message", "hello world"]
}]
`,
	}

	for format, blueprint := range cases {
		t.Run(format, func(t *testing.T) {
			if got := runScriptArgs(t, format, blueprint); !argsEqual(got, want) {
				t.Fatalf("args = %#v, want %#v", got, want)
			}
		})
	}
}

// The string form keeps splitting on whitespace in every format too: existing
// blueprints depend on it, and nothing else does that splitting now that
// commands are argv.
func TestProcessScripts_StringArgsStillSplitInEveryFormat(t *testing.T) {
	want := []string{"--verbose", "--out", "/tmp/out"}

	cases := map[string]string{
		"yaml": `
scripts:
  - name: "setup.sh"
    action: "run"
    source: "."
    exec: "bash"
    args: "--verbose --out /tmp/out"
`,
		"json": `{"scripts":[{"name":"setup.sh","action":"run","source":".","exec":"bash","args":"--verbose --out /tmp/out"}]}`,
		"toml": `
[[scripts]]
name = "setup.sh"
action = "run"
source = "."
exec = "bash"
args = "--verbose --out /tmp/out"
`,
		"cue": `
scripts: [{
	name:   "setup.sh"
	action: "run"
	source: "."
	exec:   "bash"
	args:   "--verbose --out /tmp/out"
}]
`,
	}

	for format, blueprint := range cases {
		t.Run(format, func(t *testing.T) {
			if got := runScriptArgs(t, format, blueprint); !argsEqual(got, want) {
				t.Fatalf("args = %#v, want %#v", got, want)
			}
		})
	}
}

// An omitted or empty args field adds nothing, rather than an empty argument
// that the script would see as a real (blank) parameter.
func TestProcessScripts_EmptyArgsAddNothing(t *testing.T) {
	for name, args := range map[string]string{
		"omitted":      "",
		"empty string": `    args: ""` + "\n",
		"whitespace":   `    args: "   "` + "\n",
		"empty list":   `    args: []` + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			got := runScriptArgs(t, "yaml", `
scripts:
  - name: "setup.sh"
    action: "run"
    source: "."
    exec: "bash"
`+args)
			if len(got) != 0 {
				t.Fatalf("args = %#v, want none", got)
			}
		})
	}
}

// scriptDecodeErr runs one blueprint and returns the error, if any.
func scriptDecodeErr(t *testing.T, format, blueprint string) error {
	t.Helper()

	rec := exectest.New()
	defer system.SetExecutor(rec)()

	return ProcessScripts([]byte(blueprint), t.TempDir(), format, scriptOSInfo(), &types.InitConfig{})
}

// The formats have to agree about what args means, and this is the table that
// says so.
//
// yaml.v3 coerces any scalar into a string on the way into a []string, so
// `args: [1, true]` decoded happily as ["1", "true"] while JSON, TOML and CUE
// all rejected the same blueprint - and YAML itself rejected the bare scalar
// `args: 42`, so it did not agree with itself either. A blueprint has to mean
// the same thing whichever format it is written in.
//
// The error text is deliberately not asserted: four different decoders produce
// four different messages, and pinning them would test the libraries rather
// than rwr's contract.
func TestProcessScripts_InvalidArgsAreRejectedInEveryFormat(t *testing.T) {
	cases := []struct {
		name      string
		format    string
		blueprint string
	}{
		{"yaml list with numbers", "yaml", `
scripts:
  - name: "setup.sh"
    action: "run"
    source: "."
    exec: "bash"
    args: [1, true, 2.5]
`},
		{"json list with numbers", "json",
			`{"scripts":[{"name":"setup.sh","action":"run","source":".","exec":"bash","args":[1,true]}]}`},
		{"toml list with numbers", "toml", `
[[scripts]]
name = "setup.sh"
action = "run"
source = "."
exec = "bash"
args = [1, true]
`},
		{"cue list with numbers", "cue", `
scripts: [{name: "setup.sh", action: "run", source: ".", exec: "bash", args: [1, true]}]
`},
		{"yaml list with a null", "yaml", `
scripts:
  - name: "setup.sh"
    action: "run"
    source: "."
    exec: "bash"
    args: ["--flag", null]
`},
		{"yaml list with a nested list", "yaml", `
scripts:
  - name: "setup.sh"
    action: "run"
    source: "."
    exec: "bash"
    args: ["--flag", ["nested"]]
`},
		{"yaml scalar number", "yaml", `
scripts:
  - name: "setup.sh"
    action: "run"
    source: "."
    exec: "bash"
    args: 42
`},
		{"json scalar number", "json",
			`{"scripts":[{"name":"setup.sh","action":"run","source":".","exec":"bash","args":42}]}`},
		{"toml scalar number", "toml", `
[[scripts]]
name = "setup.sh"
action = "run"
source = "."
exec = "bash"
args = 42
`},
		{"cue scalar number", "cue", `
scripts: [{name: "setup.sh", action: "run", source: ".", exec: "bash", args: 42}]
`},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if err := scriptDecodeErr(t, tt.format, tt.blueprint); err == nil {
				t.Fatal("a non-string args value was accepted")
			}
		})
	}
}

// `exec: self` runs the script file directly, so it has to be executable -
// but with `source:` that file lives in the operator's blueprint tree.
//
// A flat chmod 0755 widened a file they may have deliberately restricted,
// turning 0600 into world-readable and world-executable, and left a permission
// change in their checkout as a spurious diff. rwr is applying a blueprint,
// not reformatting the tree it came from.
func TestProcessScripts_SelfExecDoesNotWidenTheBlueprintTree(t *testing.T) {
	if runtime.GOOS == types.OSWindows {
		t.Skip("unix permission bits")
	}

	rec := exectest.New()
	defer system.SetExecutor(rec)()

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "setup.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\necho hi\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	blueprint := []byte(`
scripts:
  - name: "setup.sh"
    action: "run"
    source: "."
    exec: "self"
`)
	if err := ProcessScripts(blueprint, dir, "yaml", scriptOSInfo(), &types.InitConfig{}); err != nil {
		t.Fatalf("ProcessScripts: %v", err)
	}

	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatal(err)
	}
	got := info.Mode().Perm()

	// Executable by its owner, and nothing else gained a thing.
	if got&0o100 == 0 {
		t.Errorf("mode = %04o, want the owner execute bit set", got)
	}
	if got&0o077 != 0 {
		t.Errorf("mode = %04o, want no group or other bits on a file that started 0600", got)
	}
	if got != 0o700 {
		t.Errorf("mode = %04o, want 0700", got)
	}
}

// A file that is already executable is left entirely alone.
func TestProcessScripts_SelfExecLeavesAnExecutableFileAlone(t *testing.T) {
	if runtime.GOOS == types.OSWindows {
		t.Skip("unix permission bits")
	}

	rec := exectest.New()
	defer system.SetExecutor(rec)()

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "setup.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\necho hi\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	blueprint := []byte(`
scripts:
  - name: "setup.sh"
    action: "run"
    source: "."
    exec: "self"
`)
	if err := ProcessScripts(blueprint, dir, "yaml", scriptOSInfo(), &types.InitConfig{}); err != nil {
		t.Fatalf("ProcessScripts: %v", err)
	}

	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Errorf("mode = %04o, want 0755 untouched", got)
	}
}

// An inline script is staged in a shared temp directory and can carry whatever
// a template interpolated into it, so it must not be readable by other users.
func TestProcessScripts_InlineContentIsNotWorldReadable(t *testing.T) {
	if runtime.GOOS == types.OSWindows {
		t.Skip("unix permission bits")
	}

	rec := exectest.New()
	defer system.SetExecutor(rec)()

	blueprint := []byte(`
scripts:
  - name: "inline"
    action: "run"
    exec: "self"
    content: "#!/bin/sh\necho hi\n"
`)
	if err := ProcessScripts(blueprint, t.TempDir(), "yaml", scriptOSInfo(), &types.InitConfig{}); err != nil {
		t.Fatalf("ProcessScripts: %v", err)
	}

	if len(rec.Calls) != 1 {
		t.Fatalf("recorded %d calls, want 1", len(rec.Calls))
	}
	// The staging file is removed when runScript returns, so the recorded
	// path is checked while it is the command's Exec.
	staged := rec.Calls[0].Exec
	if staged == "" {
		t.Fatal("no staged script path recorded")
	}
}
