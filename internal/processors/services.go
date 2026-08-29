package processors

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
)

// ProcessServices manages system services (enable, disable, start, stop, restart)
// as defined in blueprint data, with cross-platform support via systemctl, launchctl, etc.
func ProcessServices(blueprintData []byte, blueprintDir string, format string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var servicesData types.ServiceData
	var err error

	// Unmarshal the blueprint data
	err = helpers.DecodeBlueprintInto(blueprintData, format, types.BlueprintTypeServices,
		helpers.TreeSchemaVersion(initConfig), &servicesData)
	if err != nil {
		return fmt.Errorf("error unmarshaling service blueprint: %w", err)
	}

	// Process imports and merge imported services
	allServices, err := processServiceImports(servicesData.Services, blueprintDir, format, helpers.TreeSchemaVersion(initConfig))
	if err != nil {
		return fmt.Errorf("error processing service imports: %w", err)
	}
	servicesData.Services = allServices

	// Filter services based on active profiles
	filteredServices := helpers.FilterByProfiles(servicesData.Services, initConfig.Variables.Flags.Profiles)

	log.Debugf("Filtering services: %d total, %d matching active profiles %v",
		len(servicesData.Services), len(filteredServices), initConfig.Variables.Flags.Profiles)

	// Process the filtered services
	err = processServices(filteredServices, osInfo, initConfig)
	if err != nil {
		return fmt.Errorf("error processing services: %w", err)
	}

	return nil
}

// One service failing does not stop the rest: the failure goes to the ledger,
// which puts it in the run's exit code, and processing continues.
func processServices(services []types.Service, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	track := newProgress(types.BlueprintTypeServices)
	for _, service := range services {
		track.expect(service.Provider, 1)
	}
	for _, service := range services {
		started := time.Now()
		var err error
		if service.Provider != "" {
			err = processProviderService(service, initConfig)
		} else {
			switch runtime.GOOS {
			case "linux":
				err = processLinuxService(service, osInfo, initConfig)
			case "darwin":
				err = processMacOSService(service, osInfo, initConfig)
			case "windows":
				err = processWindowsService(service, osInfo, initConfig)
			default:
				return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
			}
		}
		switch {
		case err != nil:
			recordFailure("services", service.Name, err)
			track.item(service.Provider, service.Name, service.Action, types.StatusFailed, err.Error(), time.Since(started))
		case system.IsDryRun():
			track.item(service.Provider, service.Name, service.Action, types.StatusPlanned, "dry-run", 0)
		default:
			track.item(service.Provider, service.Name, service.Action, types.StatusOK, "", time.Since(started))
		}
	}
	return nil
}

// processProviderService manages a service through the provider that installed
// it. Homebrew is the first such provider: formula service files live under the
// brew prefix and must be registered through `brew services`, not by guessing a
// /Library/LaunchDaemons path.
func processProviderService(service types.Service, initConfig *types.InitConfig) error {
	provider, ok := system.GetProvider(service.Provider)
	if !ok {
		return fmt.Errorf("service provider %q is not available", service.Provider)
	}

	serviceCmd, err := providerServiceCommand(provider, service)
	if err != nil {
		return err
	}
	serviceCmd.Interactive = helpers.ResolveInteractive(service.Interactive, initConfig.Variables.Flags.Interactive)
	if err := system.RunCommand(serviceCmd, initConfig.Variables.Flags.Debug); err != nil {
		return fmt.Errorf("error running %s service command: %w", service.Provider, err)
	}

	log.Infof("Service %s (%s): %s", service.Name, service.Provider, service.Action)
	return nil
}

func providerServiceCommand(provider *types.Provider, service types.Service) (types.Command, error) {
	args := provider.Services.Command(service.Action)
	if len(args) == 0 {
		return types.Command{}, fmt.Errorf("service provider %q does not support action %q", provider.Name, service.Action)
	}
	args = append(append([]string(nil), args...), service.Name)
	return types.Command{
		Exec:      provider.BinPath,
		Args:      args,
		Variables: provider.Environment,
		Elevated:  service.Elevated,
	}, nil
}

func createServiceFile(service types.Service, osInfo *types.OSInfo) error {
	if service.Content != "" {
		if system.IsDryRun() {
			log.Infof("[DRY-RUN] Would create service file: %s", service.Target)
			return nil
		}
		// system.WriteToFile, not a plain os.WriteFile: the unit file usually
		// lives under /etc, and the plain write ignored `elevated` and died
		// with EACCES for any non-root run that declared it.
		if err := system.WriteToFile(service.Target, service.Content, service.Elevated); err != nil {
			return fmt.Errorf("error creating service file: %v", err)
		}
	} else if service.Source != "" {
		if system.IsDryRun() {
			log.Infof("[DRY-RUN] Would copy service file: %s -> %s", service.Source, service.Target)
			return nil
		}
		if err := system.CopyFile(service.Source, service.Target, service.Elevated, osInfo); err != nil {
			return fmt.Errorf("error copying service file: %v", err)
		}
	} else {
		return fmt.Errorf("either content or source must be provided for create action")
	}
	return nil
}

func deleteServiceFile(service types.Service) error {
	if service.File != "" {
		if system.IsDryRun() {
			log.Infof("[DRY-RUN] Would delete service file: %s", service.File)
			return nil
		}
		if err := os.Remove(service.File); err != nil {
			// A file an earlier run already deleted is the state the blueprint
			// asked for, so a rerun converges instead of failing.
			if os.IsNotExist(err) {
				log.Infof("Service file already absent: %s", service.File)
				return nil
			}
			return fmt.Errorf("error deleting service file: %v", err)
		}
	} else {
		return fmt.Errorf("file must be provided for delete action")
	}
	return nil
}

func processLinuxService(service types.Service, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var serviceCmd types.Command
	interactive := helpers.ResolveInteractive(service.Interactive, initConfig.Variables.Flags.Interactive)

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
		if err := createServiceFile(service, osInfo); err != nil {
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
	serviceCmd.Interactive = interactive
	if err := system.RunCommand(serviceCmd, initConfig.Variables.Flags.Debug); err != nil {
		return fmt.Errorf("error running service command: %v", err)
	}

	log.Infof("Service %s: %s", service.Name, service.Action)
	return nil
}

func createLaunchDaemon(service types.Service, osInfo *types.OSInfo) error {
	if service.Content != "" {
		if system.IsDryRun() {
			log.Infof("[DRY-RUN] Would create launch daemon: %s", service.Target)
			return nil
		}
		// system.WriteToFile, not a plain os.WriteFile: /Library/LaunchDaemons
		// is root-owned, and the plain write ignored `elevated` and died with
		// EACCES for any non-root run that declared it.
		if err := system.WriteToFile(service.Target, service.Content, service.Elevated); err != nil {
			return fmt.Errorf("error creating launch daemon: %v", err)
		}
	} else if service.Source != "" {
		if system.IsDryRun() {
			log.Infof("[DRY-RUN] Would copy launch daemon: %s -> %s", service.Source, service.Target)
			return nil
		}
		if err := system.CopyFile(service.Source, service.Target, service.Elevated, osInfo); err != nil {
			return fmt.Errorf("error copying launch daemon: %v", err)
		}
	} else {
		return fmt.Errorf("either content or source must be provided for create action")
	}
	return nil
}

func deleteLaunchDaemon(service types.Service) error {
	if service.File != "" {
		if system.IsDryRun() {
			log.Infof("[DRY-RUN] Would delete launch daemon: %s", service.File)
			return nil
		}
		if err := os.Remove(service.File); err != nil {
			// Already deleted by an earlier run: the asked-for state, not an error.
			if os.IsNotExist(err) {
				log.Infof("Launch daemon already absent: %s", service.File)
				return nil
			}
			return fmt.Errorf("error deleting launch daemon: %v", err)
		}
	} else {
		return fmt.Errorf("file must be provided for delete action")
	}
	return nil
}

func processMacOSService(service types.Service, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var serviceCmd types.Command
	interactive := helpers.ResolveInteractive(service.Interactive, initConfig.Variables.Flags.Interactive)

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
		if err := processMacOSService(types.Service{Name: service.Name, Action: "stop", Elevated: service.Elevated}, osInfo, initConfig); err != nil {
			return err
		}
		if err := processMacOSService(types.Service{Name: service.Name, Action: "start", Elevated: service.Elevated}, osInfo, initConfig); err != nil {
			return err
		}
		return nil
	case "reload":
		return fmt.Errorf("reload action not supported for macOS services")
	case "status":
		serviceCmd = types.Command{
			Exec: "launchctl",
			// `launchctl list <label>` prints the job's dictionary and exits
			// non-zero when it is not loaded. The "|" and "grep" that used to be
			// here were argv elements, not a pipeline: launchctl received them as
			// job labels.
			Args: []string{"list", service.Name},
		}
	case "create":
		if err := createLaunchDaemon(service, osInfo); err != nil {
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
	serviceCmd.Interactive = interactive
	if err := system.RunCommand(serviceCmd, initConfig.Variables.Flags.Debug); err != nil {
		return fmt.Errorf("error running service command: %v", err)
	}

	log.Infof("Service %s: %s", service.Name, service.Action)
	return nil
}

func createWindowsService(service types.Service, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	if service.Content != "" {
		if system.IsDryRun() {
			log.Infof("[DRY-RUN] Would create Windows service file: %s", service.Target)
		} else if err := system.WriteToFile(service.Target, service.Content, service.Elevated); err != nil {
			return fmt.Errorf("error creating service file: %v", err)
		}
	} else if service.Source != "" {
		if system.IsDryRun() {
			log.Infof("[DRY-RUN] Would copy Windows service file: %s -> %s", service.Source, service.Target)
		} else if err := system.CopyFile(service.Source, service.Target, service.Elevated, osInfo); err != nil {
			return fmt.Errorf("error copying service file: %v", err)
		}
	} else {
		return fmt.Errorf("either content or source must be provided for create action")
	}

	// `sc create` fails when the service already exists, so a rerun on a
	// converged system errored. `sc query` succeeding means it is already
	// registered; the service file above was still refreshed. The probe is
	// skipped in dry-run, where every command "succeeds" without running.
	if !system.IsDryRun() && windowsServiceExists(service.Name, initConfig) {
		log.Infof("Windows service already exists: %s", service.Name)
		return nil
	}

	createCmd := types.Command{
		Exec:     "sc",
		Args:     []string{"create", service.Name, "binPath=", service.Target},
		Elevated: true,
	}
	if err := system.RunCommand(createCmd, initConfig.Variables.Flags.Debug); err != nil {
		return fmt.Errorf("error creating Windows service: %v", err)
	}

	return nil
}

// windowsServiceExists probes the service manager; `sc query` exits non-zero
// for a service that is not registered.
func windowsServiceExists(name string, initConfig *types.InitConfig) bool {
	queryCmd := types.Command{
		Exec:     "sc",
		Args:     []string{"query", name},
		Elevated: true,
	}
	return system.RunCommand(queryCmd, initConfig.Variables.Flags.Debug) == nil
}

func deleteWindowsService(service types.Service, initConfig *types.InitConfig) error {
	// `sc delete` fails for a service that is not registered, so a rerun on a
	// converged system errored. Absent already is the asked-for state. The
	// probe is skipped in dry-run, where every command "succeeds" without
	// running.
	if system.IsDryRun() || windowsServiceExists(service.Name, initConfig) {
		deleteCmd := types.Command{
			Exec:     "sc",
			Args:     []string{"delete", service.Name},
			Elevated: true,
		}
		if err := system.RunCommand(deleteCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error deleting Windows service: %v", err)
		}
	} else {
		log.Infof("Windows service already absent: %s", service.Name)
	}

	if service.File != "" {
		if system.IsDryRun() {
			log.Infof("[DRY-RUN] Would delete service file: %s", service.File)
		} else if err := os.Remove(service.File); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error deleting service file: %v", err)
		}
	}

	return nil
}

func processWindowsService(service types.Service, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var serviceCmd types.Command
	interactive := helpers.ResolveInteractive(service.Interactive, initConfig.Variables.Flags.Interactive)

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
		if err := processWindowsService(types.Service{Name: service.Name, Action: "stop"}, osInfo, initConfig); err != nil {
			return err
		}
		if err := processWindowsService(types.Service{Name: service.Name, Action: "start"}, osInfo, initConfig); err != nil {
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
		if err := createWindowsService(service, osInfo, initConfig); err != nil {
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

	serviceCmd.Interactive = interactive
	if err := system.RunCommand(serviceCmd, initConfig.Variables.Flags.Debug); err != nil {
		return fmt.Errorf("error running service command: %v", err)
	}

	log.Infof("Service %s: %s", service.Name, service.Action)
	return nil
}

func processServiceImports(items []types.Service, blueprintDir string, format string, treeVersion int) ([]types.Service, error) {
	return helpers.ResolveImports(items, blueprintDir,
		func(item types.Service) string { return item.Import },
		func(data []byte, fileFormat string) ([]types.Service, error) {
			var d types.ServiceData
			if err := helpers.DecodeBlueprintInto(data, fileFormat, types.BlueprintTypeServices, treeVersion, &d); err != nil {
				return nil, err
			}
			return d.Services, nil
		}, format)
}
