package helpers

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

func ReadYAMLFile(filePath string, data interface{}) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening YAML file: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("error closing YAML file: %v\n", err)
		}
	}(file)

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(data)
	if err != nil {
		return fmt.Errorf("error decoding YAML file: %w", err)
	}

	return nil
}

func ReadJSONFile(filePath string, data interface{}) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening JSON file: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("error closing JSON file: %v\n", err)
		}
	}(file)

	decoder := json.NewDecoder(file)
	err = decoder.Decode(data)
	if err != nil {
		return fmt.Errorf("error decoding JSON file: %w", err)
	}

	return nil
}

func ReadTOMLFile(filePath string, data interface{}) error {
	_, err := toml.DecodeFile(filePath, data)
	if err != nil {
		return fmt.Errorf("error decoding TOML file: %w", err)
	}

	return nil
}

func UnmarshalBlueprint(data []byte, format string, v interface{}) error {
	switch format {
	case ".yaml", ".yml":
		err := yaml.Unmarshal(data, v)
		if err != nil {
			return fmt.Errorf("error unmarshaling YAML: %w", err)
		}
	case ".json":
		err := json.Unmarshal(data, v)
		if err != nil {
			return fmt.Errorf("error unmarshaling JSON: %w", err)
		}
	case ".toml":
		err := toml.Unmarshal(data, v)
		if err != nil {
			return fmt.Errorf("error unmarshaling TOML: %w", err)
		}
	default:
		return fmt.Errorf("unsupported blueprint format: %s", format)
	}
	return nil
}
