package provider

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var providers = make(map[string]*Provider)

func LoadProviders(definitionsPath string) error {
	entries, err := os.ReadDir(definitionsPath)
	if err != nil {
		return fmt.Errorf("error reading definitions directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".yaml" {
			path := filepath.Join(definitionsPath, entry.Name())
			provider, err := loadProviderDefinition(path)
			if err != nil {
				return fmt.Errorf("error loading provider %s: %w", entry.Name(), err)
			}
			providers[provider.Name] = provider
		}
	}
	return nil
}

func loadProviderDefinition(path string) (*Provider, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var provider Provider
	if err := yaml.Unmarshal(data, &provider); err != nil {
		return nil, err
	}

	return &provider, nil
}
