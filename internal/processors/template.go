package processors

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/thefynx/rwr/internal/helpers"
	"github.com/thefynx/rwr/internal/processors/types"
)

func ProcessTemplatesFromFile(blueprintFile string) error {
	var templates []types.Template

	// Read the blueprint file based on its format
	switch filepath.Ext(blueprintFile) {
	case ".yaml", ".yml":
		err := helpers.ReadYAMLFile(blueprintFile, &templates)
		if err != nil {
			return fmt.Errorf("error reading template blueprint file: %w", err)
		}
	case ".json":
		err := helpers.ReadJSONFile(blueprintFile, &templates)
		if err != nil {
			return fmt.Errorf("error reading template blueprint file: %w", err)
		}
	case ".toml":
		err := helpers.ReadTOMLFile(blueprintFile, &templates)
		if err != nil {
			return fmt.Errorf("error reading template blueprint file: %w", err)
		}
	default:
		return fmt.Errorf("unsupported blueprint file format: %s", filepath.Ext(blueprintFile))
	}

	// Process the templates
	for _, tmpl := range templates {
		err := processTemplate(tmpl)
		if err != nil {
			return fmt.Errorf("error processing template: %w", err)
		}
	}

	return nil
}

func processTemplate(tmpl types.Template) error {
	// Read the template file
	tmplContent, err := os.ReadFile(tmpl.Source)
	if err != nil {
		return fmt.Errorf("error reading template file: %w", err)
	}

	// Create a new template
	t, err := template.New(filepath.Base(tmpl.Source)).Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("error parsing template: %w", err)
	}

	// Create the target file
	targetFile, err := os.Create(tmpl.Target)
	if err != nil {
		return fmt.Errorf("error creating target file: %w", err)
	}
	defer targetFile.Close()

	// Execute the template
	err = t.Execute(targetFile, tmpl.Variables)
	if err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}

	return nil
}
