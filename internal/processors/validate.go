package processors

//import (
//	"fmt"
//	"github.com/charmbracelet/log"
//	"github.com/fynxlabs/rwr/internal/helpers"
//	"github.com/fynxlabs/rwr/internal/types"
//	"os"
//	"path/filepath"
//)
//
//func ValidateBlueprints(initConfig *types.InitConfig) error {
//	blueprintDir := initConfig.Init.Location
//	log.Debugf("Validating blueprints in directory: %s", blueprintDir)
//
//	// Validate the init file
//	initFile := filepath.Join(blueprintDir, fmt.Sprintf("init.%s", initConfig.Init.Format))
//	err := validateInitFile(initFile)
//	if err != nil {
//		return fmt.Errorf("error validating init file: %w", err)
//	}
//
//	// Validate blueprint files
//	err = filepath.Walk(blueprintDir, func(path string, info os.FileInfo, err error) error {
//		if err != nil {
//			return err
//		}
//		if !info.IsDir() && filepath.Ext(path) == "."+initConfig.Init.Format {
//			err := validateBlueprintFile(path, initConfig)
//			if err != nil {
//				return fmt.Errorf("error validating blueprint file %s: %w", path, err)
//			}
//		}
//		return nil
//	})
//	if err != nil {
//		return fmt.Errorf("error validating blueprint files: %w", err)
//	}
//
//	return nil
//}
//
//func validateInitFile(initFile string) error {
//	var initConfig types.InitConfig
//
//	// Read the init file
//	initData, err := os.ReadFile(initFile)
//	if err != nil {
//		return fmt.Errorf("error reading init file: %w", err)
//	}
//
//	// Unmarshal the init file data
//	err = helpers.UnmarshalBlueprint(initData, filepath.Ext(initFile), &initConfig)
//	if err != nil {
//		return fmt.Errorf("error unmarshaling init file: %w", err)
//	}
//
//	// Validate the Init field
//	if initConfig.Init.Format == "" {
//		return fmt.Errorf("missing required field 'init.format' in init file")
//	}
//	if initConfig.Init.Location == "" {
//		return fmt.Errorf("missing required field 'init.location' in init file")
//	}
//
//	// Validate the PackageManagers field
//	if initConfig.PackageManagers != nil {
//		for _, pm := range initConfig.PackageManagers {
//			if pm.Name == "" {
//				return fmt.Errorf("missing required field 'packageManagers.name' in init file")
//			}
//			if pm.Action == "" {
//				return fmt.Errorf("missing required field 'packageManagers.action' in init file")
//			}
//		}
//	}
//
//	// Validate the Repositories field
//	if initConfig.Repositories != nil {
//		for _, repo := range initConfig.Repositories {
//			if repo.Name == "" {
//				return fmt.Errorf("missing required field 'repositories.name' in init file")
//			}
//			if repo.PackageManager == "" {
//				return fmt.Errorf("missing required field 'repositories.package_manager' in init file")
//			}
//			if repo.Action == "" {
//				return fmt.Errorf("missing required field 'repositories.action' in init file")
//			}
//		}
//	}
//
//	// Validate other fields as needed
//
//	return nil
//}
//
//func validateBlueprintFile(blueprintFile string, initConfig *types.InitConfig) error {
//	blueprintFileData, err := os.ReadFile(blueprintFile)
//	if err != nil {
//		return fmt.Errorf("error reading blueprint file: %w", err)
//	}
//
//	// Get the blueprint file order based on the init configuration
//	fileOrder, err := GetBlueprintFileOrder(initConfig.Init.Location, initConfig.Init.Order, initConfig.Init.RunOnlyListed, initConfig)
//	if err != nil {
//		return fmt.Errorf("error getting blueprint file order: %w", err)
//	}
//
//	// Determine the blueprint type based on the file order
//	var blueprintType string
//	for processor, files := range fileOrder {
//		for _, file := range files {
//			if file == blueprintFile {
//				blueprintType = processor
//				break
//			}
//		}
//		if blueprintType != "" {
//			break
//		}
//	}
//
//	log.Debugf("Processing %s from file: %s", blueprintType, blueprintFile)
//
//	switch blueprintType {
//	case "packages":
//		var packagesData types.PackagesData
//		err = helpers.UnmarshalBlueprint(blueprintFileData, filepath.Ext(blueprintFile), &packagesData)
//		if err != nil {
//			return fmt.Errorf("error unmarshaling packages blueprint: %w", err)
//		}
//	case "repositories":
//		var repositories []types.Repository
//		err = helpers.UnmarshalBlueprint(blueprintFileData, filepath.Ext(blueprintFile), &repositories)
//		if err != nil {
//			return fmt.Errorf("error unmarshaling repositories blueprint: %w", err)
//		}
//	case "files":
//		var files []types.File
//		err = helpers.UnmarshalBlueprint(blueprintFileData, filepath.Ext(blueprintFile), &files)
//		if err != nil {
//			return fmt.Errorf("error unmarshaling files blueprint: %w", err)
//		}
//	case "directories":
//		var directories []types.Directory
//		err = helpers.UnmarshalBlueprint(blueprintFileData, filepath.Ext(blueprintFile), &directories)
//		if err != nil {
//			return fmt.Errorf("error unmarshaling directories blueprint: %w", err)
//		}
//	case "git":
//		var gitRepositories []types.Git
//		err = helpers.UnmarshalBlueprint(blueprintFileData, filepath.Ext(blueprintFile), &gitRepositories)
//		if err != nil {
//			return fmt.Errorf("error unmarshaling git repositories blueprint: %w", err)
//		}
//	case "scripts":
//		var scripts []types.Script
//		err = helpers.UnmarshalBlueprint(blueprintFileData, filepath.Ext(blueprintFile), &scripts)
//		if err != nil {
//			return fmt.Errorf("error unmarshaling scripts blueprint: %w", err)
//		}
//	case "services":
//		var services []types.Service
//		err = helpers.UnmarshalBlueprint(blueprintFileData, filepath.Ext(blueprintFile), &services)
//		if err != nil {
//			return fmt.Errorf("error unmarshaling services blueprint: %w", err)
//		}
//	case "templates":
//		var templates []types.Template
//		err = helpers.UnmarshalBlueprint(blueprintFileData, filepath.Ext(blueprintFile), &templates)
//		if err != nil {
//			return fmt.Errorf("error unmarshaling templates blueprint: %w", err)
//		}
//	case "users":
//		var usersData types.UsersData
//		err = helpers.UnmarshalBlueprint(blueprintFileData, filepath.Ext(blueprintFile), &usersData)
//		if err != nil {
//			return fmt.Errorf("error unmarshaling users blueprint: %w", err)
//		}
//	default:
//		log.Warnf("Unsupported blueprint file: %s", blueprintFile)
//	}
//
//	return nil
//}
