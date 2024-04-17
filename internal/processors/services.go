package processors

import (
	"fmt"
	"github.com/thefynx/rwr/internal/types"
	"os"
	"path/filepath"
	"runtime"

	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/helpers"
)

func ProcessServicesFromFile(blueprintFile string, initConfig *types.InitConfig) error {
	var services []types.Service

	// Read the blueprint file
	blueprintData, err := os.ReadFile(blueprintFile)
	if err != nil {
		return fmt.Errorf("error reading blueprint file: %w", err)
	}

	// Unmarshal the blueprint data
	err = helpers.UnmarshalBlueprint(blueprintData, filepath.Ext(blueprintFile), &services)
	if err != nil {
		return fmt.Errorf("error unmarshaling service blueprint: %w", err)
	}

	// Process the services
	err = ProcessServices(services, initConfig)
	if err != nil {
		return fmt.Errorf("error processing services: %w", err)
	}

	return nil
}

func ProcessServicesFromData(blueprintData []byte, initConfig *types.InitConfig) error {
	var services []types.Service

	// Unmarshal the resolved blueprint data
	err := helpers.UnmarshalBlueprint(blueprintData, initConfig.Init.Format, &services)
	if err != nil {
		return fmt.Errorf("error unmarshaling service blueprint data: %w", err)
	}

	// Process the services
	err = ProcessServices(services, initConfig)
	if err != nil {
		return fmt.Errorf("error processing services: %w", err)
	}

	return nil
}

func ProcessServices(services []types.Service, initConfig *types.InitConfig) error {
	for _, service := range services {
		switch runtime.GOOS {
		case "linux":
			if err := processLinuxService(service, initConfig); err != nil {
				return err
			}
		case "darwin":
			if err := processMacOSService(service, initConfig); err != nil {
				return err
			}
		case "windows":
			if err := processWindowsService(service, initConfig); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
		}
	}
	return nil
}

func createServiceFile(service types.Service) error {
	if service.Content != "" {
		if err := os.WriteFile(service.Target, []byte(service.Content), 0644); err != nil {
			return fmt.Errorf("error creating service file: %v", err)
		}
	} else if service.Source != "" {
		if err := helpers.CopyFile(service.Source, service.Target, service.Elevated); err != nil {
			return fmt.Errorf("error copying service file: %v", err)
		}
	} else {
		return fmt.Errorf("either content or source must be provided for create action")
	}
	return nil
}

func deleteServiceFile(service types.Service) error {
	if service.File != "" {
		if err := os.Remove(service.File); err != nil {
			return fmt.Errorf("error deleting service file: %v", err)
		}
	} else {
		return fmt.Errorf("file must be provided for delete action")
	}
	return nil
}

func processLinuxService(service types.Service, initConfig *types.InitConfig) error {
	var serviceCmd types.Command

	switch service.Action {
	case "enable":
		serviceCmd = types.Command{
			Exec: "systemctl",
			Args: []string{"enable", service.Name},
		}
	case "disable":
		serviceCmd = types.Command{
			Exec: "systemctl",
			Args: []string{"disable", service.Name},
		}
	case "start":
		serviceCmd = types.Command{
			Exec: "systemctl",
			Args: []string{"start", service.Name},
		}
	case "stop":
		serviceCmd = types.Command{
			Exec: "systemctl",
			Args: []string{"stop", service.Name},
		}
	case "restart":
		serviceCmd = types.Command{
			Exec: "systemctl",
			Args: []string{"restart", service.Name},
		}
	case "reload":
		serviceCmd = types.Command{
			Exec: "systemctl",
			Args: []string{"reload", service.Name},
		}
	case "status":
		serviceCmd = types.Command{
			Exec: "systemctl",
			Args: []string{"status", service.Name},
		}
	case "create":
		if err := createServiceFile(service); err != nil {
			return err
		}
		serviceCmd = types.Command{
			Exec: "systemctl",
			Args: []string{"daemon-reload"},
		}
	case "delete":
		if err := deleteServiceFile(service); err != nil {
			return err
		}
		serviceCmd = types.Command{
			Exec: "systemctl",
			Args: []string{"daemon-reload"},
		}
	default:
		return fmt.Errorf("unsupported action for service: %s", service.Action)
	}

	serviceCmd.Elevated = service.Elevated
	if err := helpers.RunCommand(serviceCmd, initConfig.Variables.Flags.Debug); err != nil {
		return fmt.Errorf("error running service command: %v", err)
	}

	log.Infof("Service %s: %s", service.Name, service.Action)
	return nil
}

func createLaunchDaemon(service types.Service) error {
	if service.Content != "" {
		if err := os.WriteFile(service.Target, []byte(service.Content), 0644); err != nil {
			return fmt.Errorf("error creating launch daemon: %v", err)
		}
	} else if service.Source != "" {
		if err := helpers.CopyFile(service.Source, service.Target, service.Elevated); err != nil {
			return fmt.Errorf("error copying launch daemon: %v", err)
		}
	} else {
		return fmt.Errorf("either content or source must be provided for create action")
	}
	return nil
}

func deleteLaunchDaemon(service types.Service) error {
	if service.File != "" {
		if err := os.Remove(service.File); err != nil {
			return fmt.Errorf("error deleting launch daemon: %v", err)
		}
	} else {
		return fmt.Errorf("file must be provided for delete action")
	}
	return nil
}

func processMacOSService(service types.Service, initConfig *types.InitConfig) error {
	var serviceCmd types.Command

	switch service.Action {
	case "enable":
		serviceCmd = types.Command{
			Exec: "launchctl",
			Args: []string{"load", fmt.Sprintf("/Library/LaunchDaemons/%s.plist", service.Name)},
		}
	case "disable":
		serviceCmd = types.Command{
			Exec: "launchctl",
			Args: []string{"unload", fmt.Sprintf("/Library/LaunchDaemons/%s.plist", service.Name)},
		}
	case "start":
		serviceCmd = types.Command{
			Exec: "launchctl",
			Args: []string{"start", service.Name},
		}
	case "stop":
		serviceCmd = types.Command{
			Exec: "launchctl",
			Args: []string{"stop", service.Name},
		}
	case "restart":
		if err := processMacOSService(types.Service{Name: service.Name, Action: "stop", Elevated: service.Elevated}, initConfig); err != nil {
			return err
		}
		if err := processMacOSService(types.Service{Name: service.Name, Action: "start", Elevated: service.Elevated}, initConfig); err != nil {
			return err
		}
		return nil
	case "reload":
		return fmt.Errorf("reload action not supported for macOS services")
	case "status":
		serviceCmd = types.Command{
			Exec: "launchctl",
			Args: []string{"list", "|", "grep", service.Name},
		}
	case "create":
		if err := createLaunchDaemon(service); err != nil {
			return err
		}
		return nil
	case "delete":
		if err := deleteLaunchDaemon(service); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported action for service: %s", service.Action)
	}

	serviceCmd.Elevated = service.Elevated
	if err := helpers.RunCommand(serviceCmd, initConfig.Variables.Flags.Debug); err != nil {
		return fmt.Errorf("error running service command: %v", err)
	}

	log.Infof("Service %s: %s", service.Name, service.Action)
	return nil
}

func createWindowsService(service types.Service, initConfig *types.InitConfig) error {
	if service.Content != "" {
		if err := os.WriteFile(service.Target, []byte(service.Content), 0644); err != nil {
			return fmt.Errorf("error creating service file: %v", err)
		}
	} else if service.Source != "" {
		if err := helpers.CopyFile(service.Source, service.Target, service.Elevated); err != nil {
			return fmt.Errorf("error copying service file: %v", err)
		}
	} else {
		return fmt.Errorf("either content or source must be provided for create action")
	}

	createCmd := types.Command{
		Exec:     "sc",
		Args:     []string{"create", service.Name, "binPath=", service.Target},
		Elevated: true,
	}
	if err := helpers.RunCommand(createCmd, initConfig.Variables.Flags.Debug); err != nil {
		return fmt.Errorf("error creating Windows service: %v", err)
	}

	return nil
}

func deleteWindowsService(service types.Service, initConfig *types.InitConfig) error {
	deleteCmd := types.Command{
		Exec:     "sc",
		Args:     []string{"delete", service.Name},
		Elevated: true,
	}
	if err := helpers.RunCommand(deleteCmd, initConfig.Variables.Flags.Debug); err != nil {
		return fmt.Errorf("error deleting Windows service: %v", err)
	}

	if service.File != "" {
		if err := os.Remove(service.File); err != nil {
			return fmt.Errorf("error deleting service file: %v", err)
		}
	}

	return nil
}

func processWindowsService(service types.Service, initConfig *types.InitConfig) error {
	var serviceCmd types.Command

	switch service.Action {
	case "enable":
		serviceCmd = types.Command{
			Exec:     "sc",
			Args:     []string{"config", service.Name, "start=auto"},
			Elevated: true,
		}
	case "disable":
		serviceCmd = types.Command{
			Exec:     "sc",
			Args:     []string{"config", service.Name, "start=disabled"},
			Elevated: true,
		}
	case "start":
		serviceCmd = types.Command{
			Exec:     "sc",
			Args:     []string{"start", service.Name},
			Elevated: true,
		}
	case "stop":
		serviceCmd = types.Command{
			Exec:     "sc",
			Args:     []string{"stop", service.Name},
			Elevated: true,
		}
	case "restart":
		if err := processWindowsService(types.Service{Name: service.Name, Action: "stop"}, initConfig); err != nil {
			return err
		}
		if err := processWindowsService(types.Service{Name: service.Name, Action: "start"}, initConfig); err != nil {
			return err
		}
		return nil
	case "reload":
		return fmt.Errorf("reload action not supported for Windows services")
	case "status":
		serviceCmd = types.Command{
			Exec:     "sc",
			Args:     []string{"query", service.Name},
			Elevated: true,
		}
	case "create":
		if err := createWindowsService(service, initConfig); err != nil {
			return err
		}
		return nil
	case "delete":
		if err := deleteWindowsService(service, initConfig); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported action for service: %s", service.Action)
	}

	if err := helpers.RunCommand(serviceCmd, initConfig.Variables.Flags.Debug); err != nil {
		return fmt.Errorf("error running service command: %v", err)
	}

	log.Infof("Service %s: %s", service.Name, service.Action)
	return nil
}
