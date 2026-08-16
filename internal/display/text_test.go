package display

import "testing"

func TestTruncateUsesTerminalCells(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		width int
		want  string
	}{
		{name: "ascii", value: "abcdef", width: 4, want: "abc…"},
		{name: "wide runes", value: "界界界", width: 5, want: "界界…"},
		{name: "combining sequence", value: "e\u0301clair", width: 4, want: "e\u0301cl…"},
		{name: "already fits", value: "界界", width: 4, want: "界界"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Truncate(tt.value, tt.width); got != tt.want {
				t.Fatalf("Truncate(%q, %d) = %q, want %q", tt.value, tt.width, got, tt.want)
			}
		})
	}
}

func TestPadRightUsesTerminalCells(t *testing.T) {
	t.Parallel()

	if got := PadRight("界", 4); got != "界  " {
		t.Fatalf("PadRight() = %q, want %q", got, "界  ")
	}
}
