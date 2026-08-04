package types

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// MaxFileMode is the largest value a permission mode can hold: the nine
// permission bits plus setuid, setgid and sticky.
const MaxFileMode = 0o7777

// FileMode is a Unix permission mode declared by a blueprint.
//
// A plain int could not carry one safely. `mode: 644` is decimal 644 in YAML and
// in JSON - 0o1204, a setuid mode nobody asked for - and JSON has no octal
// literal at all, so a JSON blueprint could only say 0o644 by writing its decimal
// value 420. Every one of those decoded without complaint, so a wrong mode
// reached the file and nothing reported it.
//
// Two forms are accepted:
//
//   - A string of octal digits, with or without a prefix: "0644", "644", "0o644".
//     This means the same thing in all three formats and is the recommended form.
//
//   - A number, read as the mode's own value. That is what every parser already
//     produces for an octal literal, so YAML `0644`, TOML `0o644` and JSON `420`
//     are one and the same mode. A number that instead reads like octal digits
//     typed without quotes - `644`, `755` - is refused rather than guessed at,
//     because as a value it is a mode the blueprint cannot have meant.
//
// Zero means "no mode declared". A mode of 0000 leaves a file nobody can read,
// which no blueprint intends, and the processors need one value that means
// "apply the default".
type FileMode uint32

// IsSet reports whether the blueprint declared a mode.
func (m FileMode) IsSet() bool { return m != 0 }

// OSMode returns the mode in the form the os package takes.
func (m FileMode) OSMode() os.FileMode { return os.FileMode(m) }

// String renders the mode the way a blueprint should write it.
func (m FileMode) String() string { return "0" + strconv.FormatUint(uint64(m), 8) }

// ParseFileMode reads the string form of a mode: octal digits, optionally
// prefixed with 0 or 0o.
func ParseFileMode(s string) (FileMode, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return 0, nil
	}

	digits := trimmed
	if len(digits) > 2 && (strings.HasPrefix(digits, "0o") || strings.HasPrefix(digits, "0O")) {
		digits = digits[2:]
	}

	v, err := strconv.ParseUint(digits, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid file mode %q: a mode is octal digits with an optional 0 or 0o prefix, such as \"0644\"", s)
	}
	if v > MaxFileMode {
		return 0, fmt.Errorf("invalid file mode %q: 0o%o is larger than the largest permission mode 0o7777", s, v)
	}
	return FileMode(v), nil
}

// fileModeFromInt reads the numeric form of a mode. literal is the raw token the
// blueprint used when the format preserves it (YAML, JSON); TOML hands its
// decoders an int64 with the literal already lost, so it passes "".
func fileModeFromInt(v int64, literal string) (FileMode, error) {
	if v < 0 {
		return 0, fmt.Errorf("invalid file mode %d: a mode cannot be negative", v)
	}
	if v > MaxFileMode {
		return 0, fmt.Errorf("invalid file mode %d (0o%o): larger than the largest permission mode 0o7777", v, v)
	}

	// An explicitly octal literal says what it means, whatever digits follow.
	if !isExplicitlyBased(literal) && readsAsUnquotedOctal(v) {
		return 0, fmt.Errorf("ambiguous file mode %d: as a number it is 0o%o, but it reads like the octal mode 0%d typed without quotes; write mode: \"0%d\" for 0o%d, or mode: \"%o\" for 0o%o",
			v, v, v, v, v, v, v)
	}

	return FileMode(v), nil
}

// isExplicitlyBased reports whether a numeric literal carried its own base -
// 0644, 0o644, 0x1a4. Such a literal has already been converted by the parser
// and cannot be a decimal mistaken for octal.
func isExplicitlyBased(literal string) bool {
	digits := strings.TrimLeft(strings.TrimSpace(literal), "+-")
	return len(digits) > 1 && digits[0] == '0'
}

// readsAsUnquotedOctal reports whether a bare decimal number is far more likely
// to be an octal mode someone forgot to quote than the mode its value names.
// Nothing at or below 0o777 qualifies: those are the plain permission bits, and
// a JSON blueprint has no other way to write them.
func readsAsUnquotedOctal(v int64) bool {
	if v <= 0o777 {
		return false
	}
	for _, d := range strconv.FormatInt(v, 10) {
		if d < '0' || d > '7' {
			return false
		}
	}
	return true
}

// UnmarshalYAML reads a mode from YAML. The raw scalar is inspected rather than
// the decoded int because `0644` and `644` are the same number to yaml.v3 and
// very much not the same intent.
func (m *FileMode) UnmarshalYAML(node *yaml.Node) error {
	var (
		parsed FileMode
		err    error
	)

	switch node.Tag {
	case "!!null":
		*m = 0
		return nil
	case "!!str":
		parsed, err = ParseFileMode(node.Value)
	case "!!int":
		var v int64
		v, err = strconv.ParseInt(node.Value, 0, 64)
		if err != nil {
			err = fmt.Errorf("invalid file mode %q: not a number", node.Value)
			break
		}
		parsed, err = fileModeFromInt(v, node.Value)
	default:
		err = fmt.Errorf("invalid file mode: expected an octal string such as \"0644\" or a number, got %s", node.Tag)
	}

	if err != nil {
		return fmt.Errorf("line %d: %w", node.Line, err)
	}

	*m = parsed
	return nil
}

// UnmarshalJSON reads a mode from JSON. JSON has no octal literal, so a number
// here is always the mode's value and a quoted octal string is the only way to
// write one in the familiar notation.
func (m *FileMode) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))

	switch {
	case raw == "null":
		*m = 0
		return nil
	case strings.HasPrefix(raw, `"`):
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		parsed, err := ParseFileMode(s)
		if err != nil {
			return err
		}
		*m = parsed
		return nil
	}

	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid file mode %s: a mode is a number or a quoted octal string such as \"0644\"", raw)
	}
	parsed, err := fileModeFromInt(v, "")
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

// UnmarshalTOML reads a mode from TOML. BurntSushi hands over the decoded value
// only, so `0o644` and `420` arrive identically - as the number 420, the mode
// both of them name. The ambiguity check therefore works from the value alone,
// which is why a setuid mode has to be written as a string in TOML.
func (m *FileMode) UnmarshalTOML(data any) error {
	switch v := data.(type) {
	case string:
		parsed, err := ParseFileMode(v)
		if err != nil {
			return err
		}
		*m = parsed
	case int64:
		parsed, err := fileModeFromInt(v, "")
		if err != nil {
			return err
		}
		*m = parsed
	default:
		return fmt.Errorf("invalid file mode %v: expected an octal string such as \"0644\" or a number", data)
	}
	return nil
}

type File struct {
	Name        string                 `mapstructure:"name" yaml:"name" json:"name" toml:"name"`
	Names       []string               `mapstructure:"names,omitempty" yaml:"names,omitempty" json:"names,omitempty" toml:"names,omitempty"`
	Profiles    []string               `mapstructure:"profiles,omitempty" yaml:"profiles,omitempty" json:"profiles,omitempty" toml:"profiles,omitempty"`
	Action      string                 `mapstructure:"action" yaml:"action" json:"action" toml:"action"`
	Content     string                 `mapstructure:"content,omitempty" yaml:"content,omitempty" json:"content,omitempty" toml:"content,omitempty"`
	Source      string                 `mapstructure:"source,omitempty" yaml:"source,omitempty" json:"source,omitempty" toml:"source,omitempty"`
	Sha256      string                 `mapstructure:"sha256,omitempty" yaml:"sha256,omitempty" json:"sha256,omitempty" toml:"sha256,omitempty"` // digest of a URL source, verified before install
	Target      string                 `mapstructure:"target" yaml:"target" json:"target" toml:"target"`
	Owner       string                 `mapstructure:"owner,omitempty" yaml:"owner,omitempty" json:"owner,omitempty" toml:"owner,omitempty"`
	Group       string                 `mapstructure:"group,omitempty" yaml:"group,omitempty" json:"group,omitempty" toml:"group,omitempty"`
	Mode        FileMode               `mapstructure:"mode,omitempty" yaml:"mode,omitempty" json:"mode,omitempty" toml:"mode,omitempty"`
	Elevated    bool                   `mapstructure:"elevated" yaml:"elevated" json:"elevated" toml:"elevated"`
	Interactive *bool                  `mapstructure:"interactive,omitempty" yaml:"interactive,omitempty" json:"interactive,omitempty" toml:"interactive,omitempty"` // Override global interactive mode
	Variables   map[string]interface{} `mapstructure:"variables,omitempty" yaml:"variables,omitempty" json:"variables,omitempty" toml:"variables,omitempty"`
	Import      string                 `mapstructure:"import,omitempty" yaml:"import,omitempty" json:"import,omitempty" toml:"import,omitempty"`
}

type Directory struct {
	Name        string   `mapstructure:"name,omitempty" yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`
	Names       []string `mapstructure:"names,omitempty" yaml:"names,omitempty" json:"names,omitempty" toml:"names,omitempty"`
	Profiles    []string `mapstructure:"profiles,omitempty" yaml:"profiles,omitempty" json:"profiles,omitempty" toml:"profiles,omitempty"`
	Action      string   `mapstructure:"action" yaml:"action" json:"action" toml:"action"`
	Source      string   `mapstructure:"source,omitempty" yaml:"source,omitempty" json:"source,omitempty" toml:"source,omitempty"`
	Target      string   `mapstructure:"target" yaml:"target" json:"target" toml:"target"`
	Owner       string   `mapstructure:"owner,omitempty" yaml:"owner,omitempty" json:"owner,omitempty" toml:"owner,omitempty"`
	Group       string   `mapstructure:"group,omitempty" yaml:"group,omitempty" json:"group,omitempty" toml:"group,omitempty"`
	Mode        FileMode `mapstructure:"mode,omitempty" yaml:"mode,omitempty" json:"mode,omitempty" toml:"mode,omitempty"`
	Elevated    bool     `mapstructure:"elevated" yaml:"elevated" json:"elevated" toml:"elevated"`
	Interactive *bool    `mapstructure:"interactive,omitempty" yaml:"interactive,omitempty" json:"interactive,omitempty" toml:"interactive,omitempty"` // Override global interactive mode
	Import      string   `mapstructure:"import,omitempty" yaml:"import,omitempty" json:"import,omitempty" toml:"import,omitempty"`
}

type FileData struct {
	// SchemaVersion, when set, overrides the tree-wide version from the init file.
	SchemaVersion `mapstructure:",squash" yaml:",inline" json:",inline" toml:",inline"`
	Files         []File      `yaml:"files" json:"files" toml:"files"`
	Templates     []File      `yaml:"templates" json:"templates" toml:"templates"`
	Directories   []Directory `yaml:"directories" json:"directories" toml:"directories"`
}

// GetProfiles returns the profiles for this file.
func (f File) GetProfiles() []string {
	return f.Profiles
}

// GetProfiles returns the profiles for this directory.
func (d Directory) GetProfiles() []string {
	return d.Profiles
}
