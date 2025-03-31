package system

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/types"
)

//go:embed definitions/*.toml
var embeddedProviders embed.FS

// LoadEmbeddedProviders loads all provider definitions from the embedded filesystem
func LoadEmbeddedProviders() (map[string]*types.Provider, error) {
	providers := make(map[string]*types.Provider)

	// Read all .toml files from the embedded filesystem
	entries, err := embeddedProviders.ReadDir("definitions")
	if err != nil {
		return nil, fmt.Errorf("error reading embedded definitions: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".toml") {
			path := filepath.Join("definitions", entry.Name())

			// Read the file content
			data, err := embeddedProviders.ReadFile(path)
			if err != nil {
				log.Errorf("LoadEmbeddedProviders: Error reading %s: %v", path, err)
				continue
			}

			// Parse TOML
			var config struct {
				Provider types.Provider `toml:"provider"`
			}
			if _, err := toml.Decode(string(data), &config); err != nil {
				log.Errorf("LoadEmbeddedProviders: Failed to decode TOML %s: %v", path, err)
				continue
			}

			provider := config.Provider

			// Ensure provider name is set
			if provider.Name == "" {
				log.Errorf("LoadEmbeddedProviders: Provider name not set in %s", path)
				continue
			}

			log.Debugf("LoadEmbeddedProviders: Loaded provider %s with binary %s", provider.Name, provider.Detection.Binary)
			providers[provider.Name] = &provider
		}
	}

	return providers, nil
}

// GetEmbeddedProviderFiles returns a map of filename to content for all embedded provider files
func GetEmbeddedProviderFiles() (map[string][]byte, error) {
	files := make(map[string][]byte)

	err := fs.WalkDir(embeddedProviders, "definitions", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.HasSuffix(d.Name(), ".toml") {
			data, err := embeddedProviders.ReadFile(path)
			if err != nil {
				return err
			}
			files[d.Name()] = data
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}
