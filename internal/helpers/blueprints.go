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
	defer file.Close()

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
	defer file.Close()

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
