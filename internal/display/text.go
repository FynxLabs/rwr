// Package display provides terminal-cell-aware text formatting shared by
// plain-text and TUI renderers.
package display

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Width returns the number of terminal cells occupied by s.
func Width(s string) int {
	return ansi.StringWidth(s)
}

// Truncate shortens s to width terminal cells and adds an ellipsis when needed.
func Truncate(s string, width int) string {
	if width <= 0 || Width(s) <= width {
		return s
	}
	return ansi.Truncate(s, width, "…")
}

// PadRight appends spaces until s occupies width terminal cells.
func PadRight(s string, width int) string {
	if padding := width - Width(s); padding > 0 {
		return s + strings.Repeat(" ", padding)
	}
	return s
}
