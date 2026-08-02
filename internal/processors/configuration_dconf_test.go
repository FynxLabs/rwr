package processors

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/fynxlabs/rwr/internal/exectest"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// dconf load takes the keyfile on stdin. Commands run without a shell, so a
// "<" in argv is not a redirection — it reaches dconf as a literal argument
// and the keyfile is never read.
func TestProcessDconf_LoadsKeyfileViaStdin(t *testing.T) {
	rec := exectest.New()
	defer system.SetExecutor(rec)()

	blueprintDir := t.TempDir()
	keyfile := "[org/gnome/desktop/interface]\ncolor-scheme='prefer-dark'\n"
	if err := os.WriteFile(filepath.Join(blueprintDir, "desktop.ini"), []byte(keyfile), 0o644); err != nil {
		t.Fatal(err)
	}

	config := types.Configuration{
		Name: "desktop",
		Tool: "dconf",
		File: "desktop.ini",
	}
	initConfig := &types.InitConfig{}

	if err := processDconf(blueprintDir, config, initConfig); err != nil {
		t.Fatalf("processDconf: %v", err)
	}

	if len(rec.Calls) != 1 {
		t.Fatalf("recorded %d calls, want 1: %v", len(rec.Calls), rec.Calls)
	}
	call := rec.Calls[0]

	if want := []string{"dconf", "load", "/"}; !reflect.DeepEqual(call.Argv(), want) {
		t.Errorf("argv = %v, want %v", call.Argv(), want)
	}
	if call.Stdin != keyfile {
		t.Errorf("stdin = %q, want the keyfile contents %q", call.Stdin, keyfile)
	}
}

// A missing keyfile must error before any command runs.
func TestProcessDconf_MissingKeyfile(t *testing.T) {
	rec := exectest.New()
	defer system.SetExecutor(rec)()

	config := types.Configuration{
		Name: "desktop",
		Tool: "dconf",
		File: "nope.ini",
	}

	if err := processDconf(t.TempDir(), config, &types.InitConfig{}); err == nil {
		t.Fatal("expected an error for a missing keyfile")
	}
	if len(rec.Calls) != 0 {
		t.Fatalf("no command should run, got %v", rec.Calls)
	}
}
