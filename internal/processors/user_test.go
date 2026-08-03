package processors

import (
	"errors"
	"reflect"
	"testing"

	"github.com/fynxlabs/rwr/internal/exectest"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// A real crypt(3) hash: the processor now refuses cleartext, because
// useradd/usermod write the value verbatim into /etc/shadow.
const testCryptHash = "$6$rounds=5000$abcdefgh$0123456789abcdefghijklmnopqrstuvwxyz"

func newTestInitConfig() *types.InitConfig {
	return &types.InitConfig{
		Variables: types.Variables{
			Flags: types.Flags{
				Debug: false,
			},
		},
	}
}

// Interactive override tests

func boolPtrUser(b bool) *bool {
	return &b
}

func isProbe(cmd types.Command) bool {
	switch cmd.Exec {
	case "getent":
		return true
	case "dscl":
		return len(cmd.Args) >= 2 && cmd.Args[1] == "-read"
	case "dseditgroup":
		return len(cmd.Args) >= 2 && cmd.Args[1] == "checkmember"
	}
	return false
}

func (p probeExec) Run(cmd types.Command, debug bool) error {
	_ = p.rec.Run(cmd, debug)
	if isProbe(cmd) && !p.exists {
		return errors.New("exit status 1")
	}
	return nil
}

func (p probeExec) Output(cmd types.Command, debug bool) (string, error) {
	p.rec.Stdout = p.stdout
	return p.rec.Output(cmd, debug)
}

// platform points the processor at a GOOS and a tool set, and records every
// command it builds.
func platform(t *testing.T, goos string, exists bool, haveSysadminctl bool, stdout string) *exectest.Recorder {
	t.Helper()

	prevOS := userGOOS
	userGOOS = goos
	prevExists := commandExists
	commandExists = func(name string) bool {
		if name == "sysadminctl" {
			return haveSysadminctl
		}
		return true
	}

	rec := exectest.New()
	restore := system.SetExecutor(probeExec{rec: rec, exists: exists, stdout: stdout})

	t.Cleanup(func() {
		restore()
		commandExists = prevExists
		userGOOS = prevOS
	})
	return rec
}

func argvs(rec *exectest.Recorder) [][]string {
	out := make([][]string, 0, len(rec.Calls))
	for _, c := range rec.Calls {
		out = append(out, c.Argv())
	}
	return out
}

func assertArgv(t *testing.T, rec *exectest.Recorder, want [][]string) {
	t.Helper()
	got := argvs(rec)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("commands:\ngot:  %v\nwant: %v", got, want)
	}
	// Everything that changes the directory must be elevated; probes must not be.
	for _, c := range rec.Calls {
		if isProbeCall(c) {
			if c.Elevated {
				t.Errorf("probe %v should not be elevated", c.Argv())
			}
			continue
		}
		if c.Exec == "dscl" && len(c.Args) >= 2 && c.Args[1] == "-list" {
			continue // read-only ID enumeration
		}
		if !c.Elevated {
			t.Errorf("mutating command %v should be elevated", c.Argv())
		}
	}
}

func isProbeCall(c exectest.Call) bool {
	return isProbe(types.Command{Exec: c.Exec, Args: c.Args})
}

// Two runs of the same blueprint must produce the same end state, and the second
// must not error.
func TestProcessUsers_SecondRunDoesNotAbort(t *testing.T) {
	rec := platform(t, "linux", true, false, "")

	users := []types.User{{Name: "alice", Shell: "/bin/zsh", Action: "create"}}
	if err := processUsers(users, newTestInitConfig(), newProgress(types.BlueprintTypeUsers)); err != nil {
		t.Fatalf("second run aborted: %v", err)
	}
	if len(rec.Calls) == 0 || isProbeCall(rec.Calls[len(rec.Calls)-1]) {
		t.Fatalf("expected a converging usermod, got %v", argvs(rec))
	}
}
