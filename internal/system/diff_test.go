package system

import (
	"testing"
)

func TestCountDigits(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{0, 0},
		{1, 1},
		{9, 1},
		{10, 2},
		{99, 2},
		{100, 3},
		{999, 3},
		{1000, 4},
	}

	for _, tt := range tests {
		got := countDigits(tt.input)
		if got != tt.want {
			t.Errorf("countDigits(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestFormatTextLine(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		tabSize int
		want    string
	}{
		{"plain text", "hello world", 4, "hello world"},
		{"trailing newline", "hello\n", 4, "hello"},
		{"tab replacement", "hello\tworld", 4, "hello    world"},
		{"tab size 2", "a\tb", 2, "a  b"},
		{"multiple tabs", "\t\t", 4, "        "},
		{"empty string", "", 4, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTextLine(tt.text, tt.tabSize)
			if got != tt.want {
				t.Errorf("formatTextLine(%q, %d) = %q, want %q", tt.text, tt.tabSize, got, tt.want)
			}
		})
	}
}

func TestSplitText(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		length  int
		tabSize int
		want    int // number of chunks
	}{
		{"short text", "hello", 80, 4, 1},
		{"exact length", "abcd", 4, 4, 1},
		{"needs split", "abcdefghij", 5, 4, 2},
		{"empty text", "", 80, 4, 1},
		{"with tab", "a\tb", 80, 4, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitText(tt.text, tt.length, tt.tabSize)
			if len(got) != tt.want {
				t.Errorf("splitText(%q, %d, %d) returned %d chunks, want %d: %v",
					tt.text, tt.length, tt.tabSize, len(got), tt.want, got)
			}
		})
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{1, 2, 2},
		{2, 1, 2},
		{0, 0, 0},
		{-1, 1, 1},
		{-5, -3, -3},
	}

	for _, tt := range tests {
		got := max(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("max(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestUnifiedHeader(t *testing.T) {
	got := unifiedHeader("old.txt", "new.txt")
	want := "--- old.txt\n+++ new.txt\n"
	if got != want {
		t.Errorf("unifiedHeader = %q, want %q", got, want)
	}
}
