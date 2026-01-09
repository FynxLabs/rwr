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
	"gopkg.in/yaml.v3"
)

// UnmarshalBlueprint unmarshals a blueprint file into a struct
func UnmarshalBlueprint(data []byte, format string, v interface{}) error {
	switch format {
	case ".yaml", ".yml", "yaml", "yml":
		log.Debug("Unmarshaling YAML")
		err := yaml.Unmarshal(data, v)
		if err != nil {
			return fmt.Errorf("error unmarshaling YAML: %w", err)
		}
	case ".json", "json":
		log.Debug("Unmarshaling JSON")
		err := json.Unmarshal(data, v)
		if err != nil {
			return fmt.Errorf("error unmarshaling JSON: %w", err)
		}
	case ".toml", "toml":
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
