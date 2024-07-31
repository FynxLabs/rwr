package processors

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
	"path/filepath"
)

func ProcessFonts(blueprintData []byte, blueprintDir string, format string, initConfig *types.InitConfig) error {
	var fontsData types.FontsData
	var err error

	log.Debug("Processing fonts from blueprint")

	// Unmarshal the blueprint data
	err = helpers.UnmarshalBlueprint(blueprintData, format, &fontsData)
	if err != nil {
		return fmt.Errorf("error unmarshaling fonts blueprint data: %w", err)
	}

	// Process the fonts
	for _, font := range fontsData.Fonts {
		if len(font.Names) > 0 {
			for _, name := range font.Names {
				fontWithName := font
				fontWithName.Name = name
				if err := processFont(fontWithName, blueprintDir, initConfig); err != nil {
					return fmt.Errorf("error processing font %s: %w", name, err)
				}
			}
		} else {
			if err := processFont(font, blueprintDir, initConfig); err != nil {
				return fmt.Errorf("error processing font %s: %w", font.Name, err)
			}
		}
	}

	return nil
}

func processFont(font types.Font, blueprintDir string, initConfig *types.InitConfig) error {
	log.Infof("Processing font: %s", font.Name)

	if font.Provider == "" {
		font.Provider = "nerd"
	}

	elevated := font.Location == "system"

	switch font.Action {
	case "install":
		return installFont(font, elevated, blueprintDir, initConfig)
	case "remove":
		return removeFont(font, elevated, blueprintDir, initConfig)
	default:
		return fmt.Errorf("unsupported action for font: %s", font.Action)
	}
}

func installFont(font types.Font, elevated bool, blueprintDir string, initConfig *types.InitConfig) error {
	log.Infof("Installing font: %s", font.Name)

	installScript := filepath.Join(blueprintDir, "install.sh")

	var args []string
	if font.Name == "AllFonts" {
		args = append(args, "-q") // quiet mode for all fonts
	} else {
		args = append(args, font.Name)
	}

	if elevated {
		args = append([]string{"sudo"}, args...)
	}

	cmd := types.Command{
		Exec:     installScript,
		Args:     args,
		Elevated: elevated,
	}

	err := helpers.RunCommand(cmd, initConfig.Variables.Flags.Debug)
	if err != nil {
		return fmt.Errorf("error installing font %s: %v", font.Name, err)
	}

	log.Infof("Font %s installed successfully", font.Name)
	return nil
}

func removeFont(font types.Font, elevated bool, blueprintDir string, initConfig *types.InitConfig) error {
	log.Infof("Removing font: %s", font.Name)

	// For now, we'll just use the install script with the --remove flag
	// In the future, you might want to implement a separate uninstall script or method
	installScript := filepath.Join(blueprintDir, "install.sh")

	args := []string{"--remove"}
	if font.Name != "AllFonts" {
		args = append(args, font.Name)
	}

	if elevated {
		args = append([]string{"sudo"}, args...)
	}

	cmd := types.Command{
		Exec:     installScript,
		Args:     args,
		Elevated: elevated,
	}

	err := helpers.RunCommand(cmd, initConfig.Variables.Flags.Debug)
	if err != nil {
		return fmt.Errorf("error removing font %s: %v", font.Name, err)
	}

	log.Infof("Font %s removed successfully", font.Name)
	return nil
}
