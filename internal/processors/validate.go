package processors

import (
	"fmt"
	"github.com/charmbracelet/log"
	"os"
	"path/filepath"
	"reflect"

	"github.com/thefynx/rwr/internal/helpers"
	"github.com/thefynx/rwr/internal/processors/types"
)

func ValidateBlueprints(initConfig *types.InitConfig) error {
	blueprintDir := initConfig.Init.Location
	log.Debugf("Validating blueprints in directory: %s", blueprintDir)

	// Validate the init file
	initFile := filepath.Join(blueprintDir, fmt.Sprintf("init.%s", initConfig.Init.Format))
	err := validateInitFile(initFile)
	if err != nil {
		return fmt.Errorf("error validating init file: %w", err)
	}

	// Validate blueprint files
	err = filepath.Walk(blueprintDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == "."+initConfig.Init.Format {
			err := validateBlueprintFile(path)
			if err != nil {
				return fmt.Errorf("error validating blueprint file %s: %w", path, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error validating blueprint files: %w", err)
	}

	return nil
}

func validateInitFile(initFile string) error {
	var initConfig types.InitConfig

	// Read the init file based on its format
	switch filepath.Ext(initFile) {
	case ".yaml", ".yml":
		log.Debugf("Reading YAML file: %s", initFile)
		err := helpers.ReadYAMLFile(initFile, &initConfig)
		if err != nil {
			return fmt.Errorf("error reading init file: %w", err)
		}
	case ".json":
		log.Debugf("Reading JSON file: %s", initFile)
		err := helpers.ReadJSONFile(initFile, &initConfig)
		if err != nil {
			return fmt.Errorf("error reading init file: %w", err)
		}
	case ".toml":
		log.Debugf("Reading TOML file: %s", initFile)
		err := helpers.ReadTOMLFile(initFile, &initConfig)
		if err != nil {
			return fmt.Errorf("error reading init file: %w", err)
		}
	default:
		return fmt.Errorf("unsupported init file format: %s", filepath.Ext(initFile))
	}

	// Validate the structure of the init file
	expectedFields := []string{"Init", "PackageManagers", "Variables"}
	actualFields := reflect.ValueOf(initConfig).Type()

	for i := 0; i < actualFields.NumField(); i++ {
		field := actualFields.Field(i)
		log.Debugf("Field: %s", field.Name)
		if !helpers.Contains(expectedFields, field.Name) {
			return fmt.Errorf("unexpected field '%s' in init file", field.Name)
		}
	}

	// Validate the structure of the Init field
	if initConfig.Init.Format == "" {
		return fmt.Errorf("missing required field 'init.format' in init file")
	}
	if initConfig.Init.Location == "" {
		return fmt.Errorf("missing required field 'init.location' in init file")
	}

	return nil
}

func validateBlueprintFile(blueprintFile string) error {
	var blueprintData map[string]interface{}

	// Read the blueprint file based on its format
	switch filepath.Ext(blueprintFile) {
	case ".yaml", ".yml":
		log.Debugf("Reading YAML file: %s", blueprintFile)
		err := helpers.ReadYAMLFile(blueprintFile, &blueprintData)
		if err != nil {
			return fmt.Errorf("error reading blueprint file: %w", err)
		}
	case ".json":
		log.Debugf("Reading JSON file: %s", blueprintFile)
		err := helpers.ReadJSONFile(blueprintFile, &blueprintData)
		if err != nil {
			return fmt.Errorf("error reading blueprint file: %w", err)
		}
	case ".toml":
		log.Debugf("Reading TOML file: %s", blueprintFile)
		err := helpers.ReadTOMLFile(blueprintFile, &blueprintData)
		if err != nil {
			return fmt.Errorf("error reading blueprint file: %w", err)
		}
	default:
		return fmt.Errorf("unsupported blueprint file format: %s", filepath.Ext(blueprintFile))
	}

	// Validate the structure of the package blueprints
	if packages, ok := blueprintData["packages"]; ok {
		packagesList, ok := packages.([]interface{})
		if !ok {
			return fmt.Errorf("invalid structure for 'packages' section in blueprint file")
		}

		for _, pkg := range packagesList {
			pkgMap, ok := pkg.(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid structure for package in blueprint file")
			}

			packageType := reflect.TypeOf(types.Package{})
			for i := 0; i < packageType.NumField(); i++ {
				field := packageType.Field(i)
				tagValues := []string{field.Tag.Get("yaml"), field.Tag.Get("json"), field.Tag.Get("toml")}

				for _, tagValue := range tagValues {
					if tagValue != "" {
						if _, ok := pkgMap[tagValue]; !ok {
							return fmt.Errorf("missing field '%s' for package in blueprint file", tagValue)
						}
					}
				}
			}
		}
	}

	return nil
}
