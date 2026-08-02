// Package exectest provides a recording system.Executor for tests.
//
// It lets a test drive a real processor end to end and then assert on exactly what
// the processor asked to run — the argv, the elevation, the environment — without
// installing packages or touching the system.
package exectest

import (
	"fmt"

	"github.com/fynxlabs/rwr/internal/types"
)

// Call is one recorded command.
type Call struct {
	Exec     string
	Args     []string
	Elevated bool
	AsUser   string
	LogName  string
	Vars     map[string]string
	Stdin    string
}

// Argv is the full argument vector as the target program would receive it:
// the executable followed by its arguments.
func (c Call) Argv() []string {
	return append([]string{c.Exec}, c.Args...)
}

// String renders the call for assertion failure messages.
func (c Call) String() string {
	return fmt.Sprintf("%v (elevated=%v asUser=%q)", c.Argv(), c.Elevated, c.AsUser)
}

// Recorder is a system.Executor that records commands instead of running them.
type Recorder struct {
	Calls []Call

	// Err, when set, is returned from Run/Output for every call.
	Err error
	// Stdout is returned from Output.
	Stdout string
}

// New returns an empty Recorder.
func New() *Recorder { return &Recorder{} }

// Run records the command and returns r.Err.
func (r *Recorder) Run(cmd types.Command, _ bool) error {
	r.record(cmd)
	return r.Err
}

// Output records the command and returns r.Stdout and r.Err.
func (r *Recorder) Output(cmd types.Command, _ bool) (string, error) {
	r.record(cmd)
	return r.Stdout, r.Err
}

func (r *Recorder) record(cmd types.Command) {
	args := make([]string, len(cmd.Args))
	copy(args, cmd.Args)
	r.Calls = append(r.Calls, Call{
		Exec:     cmd.Exec,
		Args:     args,
		Elevated: cmd.Elevated,
		AsUser:   cmd.AsUser,
		LogName:  cmd.LogName,
		Vars:     cmd.Variables,
		Stdin:    cmd.Stdin,
	})
}

// Last returns the most recent call, or false if nothing was recorded.
func (r *Recorder) Last() (Call, bool) {
	if len(r.Calls) == 0 {
		return Call{}, false
	}
	return r.Calls[len(r.Calls)-1], true
}

// Find returns the calls whose executable path ends with the given name.
func (r *Recorder) Find(exec string) []Call {
	var out []Call
	for _, c := range r.Calls {
		if c.Exec == exec || hasSuffixPath(c.Exec, exec) {
			out = append(out, c)
		}
	}
	return out
}

func hasSuffixPath(path, name string) bool {
	if len(path) <= len(name) {
		return false
	}
	if path[len(path)-len(name):] != name {
		return false
	}
	sep := path[len(path)-len(name)-1]
	return sep == '/' || sep == '\\'
}
