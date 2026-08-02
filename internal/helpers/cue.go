package helpers

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	cueerrors "cuelang.org/go/cue/errors"
)

// EvalCUEToJSON evaluates a single CUE file and returns its concrete value as
// JSON, which feeds the same strict decode path every other format uses — so a
// .cue blueprint has semantics identical to its YAML twin.
//
// Evaluation is sandboxed by construction: the bytes are compiled with no
// module loader and no filesystem root, so a `.cue` file can import only CUE's
// built-in standard library. Package imports that would resolve through the
// module system — a path on disk, a network registry — fail to compile, and
// @embed is not enabled. Blueprints are untrusted input; evaluation must not
// become a way to read arbitrary files or phone home.
//
// filename is used for positions in diagnostics; pass "" when unknown.
func EvalCUEToJSON(data []byte, filename string) ([]byte, error) {
	ctx := cuecontext.New()

	options := []cue.BuildOption{}
	if filename != "" {
		options = append(options, cue.Filename(filename))
	}

	v := ctx.CompileBytes(data, options...)
	if err := v.Err(); err != nil {
		return nil, fmt.Errorf("CUE evaluation failed:\n%s", cueerrors.Details(err, nil))
	}

	// Concrete only: an unresolved field (`version: string`) is an authoring
	// error to report with its position, not a value to guess at.
	if err := v.Validate(cue.Concrete(true), cue.Final()); err != nil {
		return nil, fmt.Errorf("CUE value is not concrete:\n%s", cueerrors.Details(err, nil))
	}

	out, err := v.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("error exporting CUE value: %w", err)
	}
	return out, nil
}
