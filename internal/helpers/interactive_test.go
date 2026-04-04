package helpers

import (
	"testing"
)

func boolPtr(b bool) *bool {
	return &b
}

func TestResolveInteractive(t *testing.T) {
	tests := []struct {
		name               string
		blueprintOverride  *bool
		globalInteractive  bool
		want               bool
	}{
		{
			name:              "nil override uses global true",
			blueprintOverride: nil,
			globalInteractive: true,
			want:              true,
		},
		{
			name:              "nil override uses global false",
			blueprintOverride: nil,
			globalInteractive: false,
			want:              false,
		},
		{
			name:              "blueprint true overrides global false",
			blueprintOverride: boolPtr(true),
			globalInteractive: false,
			want:              true,
		},
		{
			name:              "blueprint false overrides global true",
			blueprintOverride: boolPtr(false),
			globalInteractive: true,
			want:              false,
		},
		{
			name:              "blueprint true with global true",
			blueprintOverride: boolPtr(true),
			globalInteractive: true,
			want:              true,
		},
		{
			name:              "blueprint false with global false",
			blueprintOverride: boolPtr(false),
			globalInteractive: false,
			want:              false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveInteractive(tt.blueprintOverride, tt.globalInteractive)
			if got != tt.want {
				t.Errorf("ResolveInteractive(%v, %v) = %v, want %v",
					tt.blueprintOverride, tt.globalInteractive, got, tt.want)
			}
		})
	}
}
