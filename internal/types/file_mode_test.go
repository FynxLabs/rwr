package types

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// modeCase is one way of writing a mode, in each of the three formats. A blank
// literal means the format cannot express that form at all - JSON has no octal
// literal - and is skipped rather than asserted.
type modeCase struct {
	name string
	yaml string
	json string
	toml string

	want    os.FileMode
	wantErr string // substring; empty means the value must decode
}

var modeCases = []modeCase{
	{
		name: "quoted octal with leading zero",
		yaml: `mode: "0644"`, json: `"mode": "0644"`, toml: `mode = "0644"`,
		want: 0o644,
	},
	{
		name: "quoted octal without prefix",
		yaml: `mode: "644"`, json: `"mode": "644"`, toml: `mode = "644"`,
		want: 0o644,
	},
	{
		name: "quoted octal with 0o prefix",
		yaml: `mode: "0o644"`, json: `"mode": "0o644"`, toml: `mode = "0o644"`,
		want: 0o644,
	},
	{
		name: "quoted setuid mode",
		yaml: `mode: "4755"`, json: `"mode": "4755"`, toml: `mode = "4755"`,
		want: 0o4755,
	},
	{
		// The literal every parser converts for us, so all three agree on 0o644.
		name: "octal literal",
		yaml: `mode: 0644`, json: ``, toml: `mode = 0o644`,
		want: 0o644,
	},
	{
		name: "0o literal",
		yaml: `mode: 0o644`, json: ``, toml: `mode = 0o644`,
		want: 0o644,
	},
	{
		// The bug this type exists for: 644 as a number is 0o1204, a setuid mode
		// no blueprint means, and it used to be applied without a word.
		name: "bare 644",
		yaml: `mode: 644`, json: `"mode": 644`, toml: `mode = 644`,
		wantErr: "ambiguous file mode 644",
	},
	{
		name: "bare 755",
		yaml: `mode: 755`, json: `"mode": 755`, toml: `mode = 755`,
		wantErr: "ambiguous file mode 755",
	},
	{
		// The decimal value of a mode, which is the only way JSON can name one
		// numerically. 420 is 0o644 and 493 is 0o755.
		name: "decimal value of 0o644",
		yaml: `mode: 420`, json: `"mode": 420`, toml: `mode = 420`,
		want: 0o644,
	},
	{
		name: "decimal value with a 9 digit",
		yaml: `mode: 493`, json: `"mode": 493`, toml: `mode = 493`,
		want: 0o755,
	},
	{
		name: "mode absent",
		yaml: `target: /tmp/x`, json: `"target": "/tmp/x"`, toml: `target = "/tmp/x"`,
		want: 0,
	},
	{
		name: "mode zero",
		yaml: `mode: 0`, json: `"mode": 0`, toml: `mode = 0`,
		want: 0,
	},
	{
		name: "out of range number",
		yaml: `mode: 40000`, json: `"mode": 40000`, toml: `mode = 40000`,
		wantErr: "larger than the largest permission mode",
	},
	{
		name: "out of range string",
		yaml: `mode: "10000"`, json: `"mode": "10000"`, toml: `mode = "10000"`,
		wantErr: "larger than the largest permission mode",
	},
	{
		name: "string with an 8 digit",
		yaml: `mode: "0688"`, json: `"mode": "0688"`, toml: `mode = "0688"`,
		wantErr: `invalid file mode "0688"`,
	},
	{
		name: "string with a 9 digit",
		yaml: `mode: "0999"`, json: `"mode": "0999"`, toml: `mode = "0999"`,
		wantErr: `invalid file mode "0999"`,
	},
	{
		name: "negative number",
		yaml: `mode: -1`, json: `"mode": -1`, toml: `mode = -1`,
		wantErr: "cannot be negative",
	},
}

func TestFileMode_YAML(t *testing.T) {
	for _, tc := range modeCases {
		if tc.yaml == "" {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			var f File
			err := yaml.Unmarshal([]byte("name: x\n"+tc.yaml+"\n"), &f)
			checkMode(t, f.Mode, err, tc)
		})
	}
}

func TestFileMode_JSON(t *testing.T) {
	for _, tc := range modeCases {
		if tc.json == "" {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			var f File
			err := json.Unmarshal([]byte(`{"name": "x", `+tc.json+`}`), &f)
			checkMode(t, f.Mode, err, tc)
		})
	}
}

func TestFileMode_TOML(t *testing.T) {
	for _, tc := range modeCases {
		if tc.toml == "" {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			var f File
			_, err := toml.Decode("name = \"x\"\n"+tc.toml+"\n", &f)
			checkMode(t, f.Mode, err, tc)
		})
	}
}

// A directory carries the same type, so it accepts and refuses the same things.
func TestDirectoryMode_YAML(t *testing.T) {
	var d Directory
	if err := yaml.Unmarshal([]byte("name: x\nmode: \"0700\"\n"), &d); err != nil {
		t.Fatalf("decoding directory mode: %v", err)
	}
	if got := d.Mode.OSMode(); got != 0o700 {
		t.Errorf("directory mode: got %o, want %o", got, 0o700)
	}

	if err := yaml.Unmarshal([]byte("name: x\nmode: 755\n"), &d); err == nil {
		t.Error("directory mode 755 decoded, expected it to be refused as ambiguous")
	}
}

// The error has to name the value, because the point of refusing it is that the
// writer can see which entry to fix.
func TestFileMode_AmbiguousErrorSuggestsBothReadings(t *testing.T) {
	var f File
	err := yaml.Unmarshal([]byte("mode: 644\n"), &f)
	if err == nil {
		t.Fatal("mode: 644 decoded, expected it to be refused")
	}
	for _, want := range []string{"644", "0o1204", `"0644"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %s", err, want)
		}
	}
}

func TestFileMode_String(t *testing.T) {
	if got := FileMode(0o644).String(); got != "0644" {
		t.Errorf("String(): got %s, want 0644", got)
	}
}

func TestFileMode_IsSet(t *testing.T) {
	if FileMode(0).IsSet() {
		t.Error("an undeclared mode reports as set")
	}
	if !FileMode(0o600).IsSet() {
		t.Error("a declared mode reports as unset")
	}
}

func checkMode(t *testing.T, got FileMode, err error, tc modeCase) {
	t.Helper()

	if tc.wantErr != "" {
		if err == nil {
			t.Fatalf("decoded to %o, expected error containing %q", got, tc.wantErr)
		}
		if !strings.Contains(err.Error(), tc.wantErr) {
			t.Fatalf("error %q does not contain %q", err, tc.wantErr)
		}
		return
	}

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.OSMode() != tc.want {
		t.Errorf("mode: got %o, want %o", got.OSMode(), tc.want)
	}
}
