package processors

import (
	"os"
	"path/filepath"
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
