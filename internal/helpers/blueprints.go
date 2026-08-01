// Package helpers provides utility functions for blueprint processing.
// It includes profile filtering, import resolution, blueprint unmarshaling,
// Git operations, and configuration file creation. These helper functions
// support the core blueprint processing workflow by handling common tasks
// such as template rendering, system checks, and blueprint file parsing.
package helpers

import (
	"encoding/json"
	"fmt"
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
func UnmarshalBlueprint(data []byte, format string, v interface{}) error {
	switch format {
	case types.FormatExtYAML, types.FormatExtYAMLAlt, types.FormatYAML, types.FormatYAMLAlt:
		log.Debug("Unmarshaling YAML")
		err := yaml.Unmarshal(data, v)
		if err != nil {
			return fmt.Errorf("error unmarshaling YAML: %w", err)
		}
	case types.FormatExtJSON, types.FormatJSON:
		log.Debug("Unmarshaling JSON")
		err := json.Unmarshal(data, v)
		if err != nil {
			return fmt.Errorf("error unmarshaling JSON: %w", err)
		}
	case types.FormatExtTOML, types.FormatTOML:
		log.Debug("Unmarshaling TOML")
		err := toml.Unmarshal(data, v)
		if err != nil {
			return fmt.Errorf("error unmarshaling TOML: %w", err)
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
