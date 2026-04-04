// Package helpers provides utility functions for blueprint processing.
// It includes profile filtering, import resolution, blueprint unmarshaling,
// Git operations, and configuration file creation. These helper functions
// support the core blueprint processing workflow by handling common tasks
// such as template rendering, system checks, and blueprint file parsing.
package helpers

import (
	"encoding/json"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/log"
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
