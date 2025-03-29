package helpers

import (
	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/pkg/providers"
	"github.com/fynxlabs/rwr/internal/types"
)

func CleanPackageManagers(osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	// Get available providers
	available := providers.GetAvailableProviders()

	// Clean each available provider
	for name, provider := range available {
		if provider.Commands.Clean == "" {
			continue
		}

		pmInfo := providers.GetPackageManagerInfo(provider, provider.BinPath)
		log.Debugf("Running clean command for package manager: %s", name)
		log.Debugf(" Running clean command: %s", pmInfo.Clean)

		cleanCmd := types.Command{
			Exec:     pmInfo.Clean,
			Elevated: pmInfo.Elevated,
		}

		if err := RunCommand(cleanCmd, initConfig.Variables.Flags.Debug); err != nil {
			log.Errorf("Error cleaning package manager %s: %v", name, err)
			continue
		}

		log.Infof("Cleaned package manager: %s", name)
	}

	return nil
}
