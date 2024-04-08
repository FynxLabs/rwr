package processors

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/helpers"
	"github.com/thefynx/rwr/internal/processors/types"
)

func ProcessServicesFromFile(blueprintFile string) error {
	var services []types.Service

	// Read the blueprint file based on the file format
	switch filepath.Ext(blueprintFile) {
	case ".yaml", ".yml":
		err := helpers.ReadYAMLFile(blueprintFile, &services)
		if err != nil {
			return fmt.Errorf("error reading service blueprint file: %w", err)
		}
	case ".json":
		err := helpers.ReadJSONFile(blueprintFile, &services)
		if err != nil {
			return fmt.Errorf("error reading service blueprint file: %w", err)
		}
	case ".toml":
		err := helpers.ReadTOMLFile(blueprintFile, &services)
		if err != nil {
			return fmt.Errorf("error reading service blueprint file: %w", err)
		}
	default:
		return fmt.Errorf("unsupported blueprint file format: %s", filepath.Ext(blueprintFile))
	}

	// Process the services
	err := ProcessServices(services)
	if err != nil {
		return fmt.Errorf("error processing services: %w", err)
	}

	return nil
}

func ProcessServices(services []types.Service) error {
	for _, service := range services {
		switch runtime.GOOS {
		case "linux":
			if err := processLinuxService(service); err != nil {
				return err
			}
		case "darwin":
			if err := processMacOSService(service); err != nil {
				return err
			}
		case "windows":
			if err := processWindowsService(service); err != nil {
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
		if err := helpers.CopyFile(service.Source, service.Target); err != nil {
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

func processLinuxService(service types.Service) error {
	var args []string

	switch service.Action {
	case "enable":
		args = []string{"enable", service.Name}
	case "disable":
		args = []string{"disable", service.Name}
	case "start":
		args = []string{"start", service.Name}
	case "stop":
		args = []string{"stop", service.Name}
	case "restart":
		args = []string{"restart", service.Name}
	case "reload":
		args = []string{"reload", service.Name}
	case "status":
		args = []string{"status", service.Name}
	case "create":
		if err := createServiceFile(service); err != nil {
			return err
		}
		args = []string{"daemon-reload"}
	case "delete":
		if err := deleteServiceFile(service); err != nil {
			return err
		}
		args = []string{"daemon-reload"}
	default:
		return fmt.Errorf("unsupported action for service: %s", service.Action)
	}

	if service.Elevated {
		if err := helpers.RunWithElevatedPrivileges("systemctl", args...); err != nil {
			return fmt.Errorf("error running service command: %v", err)
		}
	} else {
		if err := helpers.RunCommand("systemctl", args...); err != nil {
			return fmt.Errorf("error running service command: %v", err)
		}
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
		if err := helpers.CopyFile(service.Source, service.Target); err != nil {
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

func processMacOSService(service types.Service) error {
	var args []string

	switch service.Action {
	case "enable":
		args = []string{"load", fmt.Sprintf("/Library/LaunchDaemons/%s.plist", service.Name)}
	case "disable":
		args = []string{"unload", fmt.Sprintf("/Library/LaunchDaemons/%s.plist", service.Name)}
	case "start":
		args = []string{"start", service.Name}
	case "stop":
		args = []string{"stop", service.Name}
	case "restart":
		if err := processMacOSService(types.Service{Name: service.Name, Action: "stop", Elevated: service.Elevated}); err != nil {
			return err
		}
		if err := processMacOSService(types.Service{Name: service.Name, Action: "start", Elevated: service.Elevated}); err != nil {
			return err
		}
		return nil
	case "reload":
		return fmt.Errorf("reload action not supported for macOS services")
	case "status":
		args = []string{"list", "|", "grep", service.Name}
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

	if service.Elevated {
		if err := helpers.RunWithElevatedPrivileges("launchctl", args...); err != nil {
			return fmt.Errorf("error running service command: %v", err)
		}
	} else {
		if err := helpers.RunCommand("launchctl", args...); err != nil {
			return fmt.Errorf("error running service command: %v", err)
		}
	}

	log.Infof("Service %s: %s", service.Name, service.Action)
	return nil
}

func createWindowsService(service types.Service) error {
	if service.Content != "" {
		if err := os.WriteFile(service.Target, []byte(service.Content), 0644); err != nil {
			return fmt.Errorf("error creating service file: %v", err)
		}
	} else if service.Source != "" {
		if err := helpers.CopyFile(service.Source, service.Target); err != nil {
			return fmt.Errorf("error copying service file: %v", err)
		}
	} else {
		return fmt.Errorf("either content or source must be provided for create action")
	}

	args := []string{"create", service.Name, "binPath=", service.Target}
	if err := helpers.RunWithElevatedPrivileges("sc", args...); err != nil {
		return fmt.Errorf("error creating Windows service: %v", err)
	}

	return nil
}

func deleteWindowsService(service types.Service) error {
	args := []string{"delete", service.Name}
	if err := helpers.RunWithElevatedPrivileges("sc", args...); err != nil {
		return fmt.Errorf("error deleting Windows service: %v", err)
	}

	if service.File != "" {
		if err := os.Remove(service.File); err != nil {
			return fmt.Errorf("error deleting service file: %v", err)
		}
	}

	return nil
}

func processWindowsService(service types.Service) error {
	var args []string

	switch service.Action {
	case "enable":
		args = []string{"config", service.Name, "start=auto"}
	case "disable":
		args = []string{"config", service.Name, "start=disabled"}
	case "start":
		args = []string{"start", service.Name}
	case "stop":
		args = []string{"stop", service.Name}
	case "restart":
		if err := processWindowsService(types.Service{Name: service.Name, Action: "stop"}); err != nil {
			return err
		}
		if err := processWindowsService(types.Service{Name: service.Name, Action: "start"}); err != nil {
			return err
		}
		return nil
	case "reload":
		return fmt.Errorf("reload action not supported for Windows services")
	case "status":
		args = []string{"query", service.Name}
	case "create":
		if err := createWindowsService(service); err != nil {
			return err
		}
		return nil
	case "delete":
		if err := deleteWindowsService(service); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported action for service: %s", service.Action)
	}

	if err := helpers.RunWithElevatedPrivileges("sc", args...); err != nil {
		return fmt.Errorf("error running service command: %v", err)
	}

	log.Infof("Service %s: %s", service.Name, service.Action)
	return nil
}
