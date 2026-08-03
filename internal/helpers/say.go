package helpers

import (
	"fmt"
	"io"
)

// Say writes progress lines to a command's output; a broken stdout must not
// abort the work being reported on, so the write error is deliberately
// dropped.
func Say(w io.Writer, format string, args ...interface{}) {
	fmt.Fprintf(w, format, args...) //nolint:errcheck
}
