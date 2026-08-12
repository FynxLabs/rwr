package types

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ScriptArgs is a script's argument list, written either as a list or as a
// single string.
//
// The string form splits on whitespace, which is what `args` has always done
// and what every existing blueprint relies on: `args: "--verbose --out /tmp"`
// reaches the script as three arguments. Commands are argv now, so nothing
// else does that splitting - without it the whole string would arrive as one
// argument.
//
// That split is also the limitation the list form exists for. Splitting on
// whitespace means an argument containing whitespace cannot be written at all:
// no amount of quoting inside the string helps, because the value is never
// parsed by a shell. A commit message, a path with a space, a --message flag -
// none of them could be passed. The list form takes each element verbatim:
//
//	args: ["--message", "hello world"]
//
// Both forms stay supported. The string is the common case and reads better
// for simple flags; the list is the one that can express everything.
type ScriptArgs []string

// UnmarshalYAML reads args from YAML as either a scalar or a sequence.
func (a *ScriptArgs) UnmarshalYAML(node *yaml.Node) error {
	switch node.Tag {
	case "!!null":
		*a = nil
		return nil
	case "!!str":
		*a = splitScriptArgs(node.Value)
		return nil
	case "!!seq":
		var list []string
		if err := node.Decode(&list); err != nil {
			return fmt.Errorf("line %d: args list must contain only strings: %w", node.Line, err)
		}
		*a = list
		return nil
	default:
		return fmt.Errorf("line %d: args must be a string or a list of strings, got %s", node.Line, node.Tag)
	}
}

// UnmarshalJSON reads args from JSON as either a string or an array. CUE
// blueprints arrive through here too, since CUE evaluates to JSON.
func (a *ScriptArgs) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	switch {
	case raw == "null":
		*a = nil
		return nil
	case strings.HasPrefix(raw, "["):
		var list []string
		if err := json.Unmarshal(data, &list); err != nil {
			return fmt.Errorf("args list must contain only strings: %w", err)
		}
		*a = list
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("args must be a string or a list of strings")
	}
	*a = splitScriptArgs(s)
	return nil
}

// UnmarshalTOML reads args from TOML. BurntSushi hands over the decoded value,
// so a list arrives as []interface{} whose elements have to be checked one by
// one rather than asserted wholesale.
func (a *ScriptArgs) UnmarshalTOML(data any) error {
	switch v := data.(type) {
	case nil:
		*a = nil
		return nil
	case string:
		*a = splitScriptArgs(v)
		return nil
	case []string:
		*a = v
		return nil
	case []any:
		list := make([]string, 0, len(v))
		for i, element := range v {
			s, ok := element.(string)
			if !ok {
				return fmt.Errorf("args[%d] must be a string, got %T", i, element)
			}
			list = append(list, s)
		}
		*a = list
		return nil
	default:
		return fmt.Errorf("args must be a string or a list of strings, got %T", data)
	}
}

// splitScriptArgs divides the string form on whitespace.
func splitScriptArgs(s string) []string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return nil
	}
	return fields
}
