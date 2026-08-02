// Package helpers provides utility functions for blueprint processing.
// It includes profile filtering, import resolution, blueprint unmarshaling,
// Git operations, and configuration file creation. These helper functions
// support the core blueprint processing workflow by handling common tasks
// such as template rendering, system checks, and blueprint file parsing.
package helpers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"runtime"
	"strings"

	"charm.land/log/v2"
	"github.com/BurntSushi/toml"
	"github.com/fynxlabs/rwr/internal/types"
	"gopkg.in/yaml.v3"
)

// UnmarshalBlueprint parses blueprint data into the provided struct.
// It supports YAML, JSON, and TOML formats specified by the format parameter.
// Format accepts file extensions (".yaml", ".json", ".toml") or format names ("yaml", "json", "toml").
//
// Unknown keys are ignored. Use UnmarshalBlueprintStrict for blueprint content;
// this lenient form exists for probes that deliberately read one key out of a
// document, such as the schema_version declaration.
func UnmarshalBlueprint(data []byte, format string, v interface{}) error {
	return unmarshalBlueprint(data, format, v, false)
}

// UnmarshalBlueprintStrict parses blueprint data and rejects any key the target
// struct does not define.
//
// A silently ignored key is a blueprint that looks applied and is not: `pacakges:`
// yields an empty section, every processor finds nothing to do, and the run
// reports success having changed nothing. A misspelled `profiles` is worse — the
// entry loses its scoping and runs on every machine. Both failures are invisible
// at any log level, so they surface as "rwr didn't do anything" long after the
// typo was written.
func UnmarshalBlueprintStrict(data []byte, format string, v interface{}) error {
	return unmarshalBlueprint(data, format, v, true)
}

func unmarshalBlueprint(data []byte, format string, v interface{}, strict bool) error {
	// The registry canonicalizes every accepted spelling (name, alias, dotted
	// extension), so this switch is over canonical names only and unsupported
	// formats fail with one message instead of falling through per spelling.
	canonical, err := CanonicalFormat(format)
	if err != nil {
		return err
	}
	switch canonical {
	case types.FormatYAML:
		log.Debug("Unmarshaling YAML")
		if !strict {
			if err := yaml.Unmarshal(data, v); err != nil {
				return fmt.Errorf("error unmarshaling YAML: %w", err)
			}
			break
		}
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		if err := dec.Decode(v); err != nil {
			// An empty document is a valid blueprint section with nothing in it.
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("error unmarshaling YAML: %w", err)
		}
	case types.FormatJSON:
		log.Debug("Unmarshaling JSON")
		if !strict {
			if err := json.Unmarshal(data, v); err != nil {
				return fmt.Errorf("error unmarshaling JSON: %w", err)
			}
			break
		}
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		if err := dec.Decode(v); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("error unmarshaling JSON: %w", err)
		}
	case types.FormatTOML:
		log.Debug("Unmarshaling TOML")
		meta, err := toml.Decode(string(data), v)
		if err != nil {
			return fmt.Errorf("error unmarshaling TOML: %w", err)
		}
		// BurntSushi has no strict mode; it reports what it could not place
		// instead, so the check has to happen here rather than in the decoder.
		if strict {
			if undecoded := meta.Undecoded(); len(undecoded) > 0 {
				keys := make([]string, 0, len(undecoded))
				for _, key := range undecoded {
					keys = append(keys, key.String())
				}
				return fmt.Errorf("error unmarshaling TOML: unknown key(s): %s", strings.Join(keys, ", "))
			}
		}
	default:
		return fmt.Errorf("unsupported blueprint format: %s", format)
	}
	log.Debugf("Blueprint unmarshaled successfully")
	return nil
}

// DefaultVariables builds the variable set every blueprint template renders
// against: the running user, and an empty user-defined map.
//
// Both the run path and `rwr validate` need this. Validation used to render
// against a zero-valued set, so any blueprint referencing {{ .User.home }} rendered
// "<no value>" into the path it was about to be checked for.
func DefaultVariables() (types.Variables, error) {
	currentUser, err := user.Current()
	if err != nil {
		return types.Variables{}, fmt.Errorf("error retrieving current user information: %w", err)
	}

	names := strings.Fields(currentUser.Name)
	firstName, lastName := "", ""
	if len(names) > 0 {
		firstName = names[0]
	}
	if len(names) > 1 {
		lastName = names[len(names)-1]
	}

	groupName := ""
	if runtime.GOOS != types.OSWindows {
		group, err := user.LookupGroupId(currentUser.Gid)
		if err != nil {
			log.With("err", err).Warnf("Error retrieving primary group name for user %s", currentUser.Username)
		} else {
			groupName = group.Name
		}
	}

	return types.Variables{
		User: types.UserInfo{
			Username:  currentUser.Username,
			FirstName: firstName,
			LastName:  lastName,
			FullName:  currentUser.Name,
			GroupName: groupName,
			Home:      currentUser.HomeDir,
			Shell:     os.Getenv("SHELL"),
		},
		UserDefined: make(map[string]interface{}),
	}, nil
}
