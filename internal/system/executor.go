package system

import (
	"github.com/fynxlabs/rwr/internal/types"
)

// Executor runs commands on behalf of the processors. It exists so command
// construction can be observed in tests without spawning real processes: the
// production implementation shells out, while tests substitute a recorder and
// assert on the exact argv, elevation and environment each processor produced.
type Executor interface {
	// Run executes the command, streaming or capturing output per the command's
	// Interactive and LogName settings.
	Run(cmd types.Command, debug bool) error
	// Output executes the command and returns its stdout.
	Output(cmd types.Command, debug bool) (string, error)
}

// current is the executor used by RunCommand and RunCommandOutput.
var current Executor = osExecutor{}

// SetExecutor swaps in a different Executor and returns a function restoring the
// previous one. Intended for tests:
//
//	rec := exectest.New()
//	defer system.SetExecutor(rec)()
func SetExecutor(e Executor) (restore func()) {
	previous := current
	current = e
	return func() { current = previous }
}

// osExecutor is the production Executor; it runs commands as real processes.
type osExecutor struct{}

func (osExecutor) Run(cmd types.Command, debug bool) error {
	return runCommand(cmd, debug)
}

func (osExecutor) Output(cmd types.Command, debug bool) (string, error) {
	return runCommandOutput(cmd, debug)
}
