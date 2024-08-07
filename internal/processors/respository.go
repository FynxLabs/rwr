package processors

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
)

func ProcessRepositories(blueprintData []byte, format string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var repositoriesBlueprint types.RepositoriesData
	var err error

	log.Debugf("Processing repositories from blueprint")

	// Unmarshal the blueprint data
	err = helpers.UnmarshalBlueprint(blueprintData, format, &repositoriesBlueprint)
	if err != nil {
		return fmt.Errorf("error unmarshaling repository blueprint: %w", err)
	}

	// Process the repositories
	err = processRepositories(repositoriesBlueprint.Repositories, osInfo, initConfig)
	if err != nil {
		return fmt.Errorf("error processing repositories: %w", err)
	}

	return nil
}

func processRepositories(repositories []types.Repository, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	for _, repo := range repositories {
		log.Infof("Processing repository %s", repo.Name)

		switch repo.PackageManager {
		case "apt":
			log.Debugf("Processing APT repository")
			if err := processAptRepository(repo, osInfo, initConfig); err != nil {
				return err
			}
		case "brew":
			log.Debugf("Processing Homebrew repository")
			if err := processBrewRepository(repo, osInfo, initConfig); err != nil {
				return err
			}
		case "dnf", "yum":
			log.Debugf("Processing DNF/Yum repository")
			if err := processDnfYumRepository(repo, initConfig); err != nil {
				return err
			}
		case "zypper":
			log.Debugf("Processing Zypper repository")
			if err := processZypperRepository(repo, osInfo, initConfig); err != nil {
				return err
			}
		case "pacman":
			log.Debugf("Processing Pacman repository")
			if err := processPacmanRepository(repo, osInfo, initConfig); err != nil {
				return err
			}
		case "choco":
			log.Debugf("Processing Chocolatey repository")
			if err := processChocoRepository(repo, osInfo, initConfig); err != nil {
				return err
			}
		case "scoop":
			log.Debugf("Processing Scoop repository")
			if err := processScoopRepository(repo, osInfo, initConfig); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported package manager: %s", repo.PackageManager)
		}
	}

	if osInfo.PackageManager.Apt.Update != "" {
		log.Info("Processing APT Updates")
		updateCmd := types.Command{
			Exec:     osInfo.PackageManager.Apt.Update,
			Elevated: osInfo.PackageManager.Apt.Elevated,
		}
		if err := helpers.RunCommand(updateCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error updating APT package lists: %v", err)
		}
	}

	if osInfo.PackageManager.Brew.Update != "" {
		log.Info("Processing Homebrew Updates")
		updateCmd := types.Command{
			Exec:     osInfo.PackageManager.Brew.Update,
			Elevated: osInfo.PackageManager.Brew.Elevated,
		}
		if err := helpers.RunCommand(updateCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error updating Homebrew package lists: %v", err)
		}
	}

	if osInfo.PackageManager.Dnf.Update != "" {
		log.Info("Processing DNF Updates")
		updateCmd := types.Command{
			Exec:     osInfo.PackageManager.Dnf.Update,
			Elevated: osInfo.PackageManager.Dnf.Elevated,
		}
		if err := helpers.RunCommand(updateCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error updating DNF/Yum package lists: %v", err)
		}
	}

	if osInfo.PackageManager.Zypper.Update != "" {
		log.Info("Processing Zypper Updates")
		updateCmd := types.Command{
			Exec:     osInfo.PackageManager.Zypper.Update,
			Elevated: osInfo.PackageManager.Zypper.Elevated,
		}
		if err := helpers.RunCommand(updateCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error updating Zypper package lists: %v", err)
		}
	}

	if osInfo.PackageManager.Pacman.Update != "" {
		log.Info("Processing Pacman Updates")
		updateCmd := types.Command{
			Exec:     osInfo.PackageManager.Pacman.Update,
			Elevated: osInfo.PackageManager.Pacman.Elevated,
		}
		if err := helpers.RunCommand(updateCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error updating Pacman package lists: %v", err)
		}
	}

	if osInfo.PackageManager.Chocolatey.Update != "" {
		log.Info("Processing Chocolatey Updates")
		updateCmd := types.Command{
			Exec:     osInfo.PackageManager.Chocolatey.Update,
			Elevated: osInfo.PackageManager.Chocolatey.Elevated,
		}
		if err := helpers.RunCommand(updateCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error updating Chocolatey package lists: %v", err)
		}
	}

	if osInfo.PackageManager.Scoop.Update != "" {
		log.Info("Processing Scoop Updates")
		updateCmd := types.Command{
			Exec:     osInfo.PackageManager.Scoop.Update,
			Elevated: osInfo.PackageManager.Scoop.Elevated,
		}
		if err := helpers.RunCommand(updateCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error updating Scoop package lists: %v", err)
		}
	}

	return nil
}

func processAptRepository(repo types.Repository, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	tempKeyFile := filepath.Join("/tmp", repo.Name+".gpg")
	keyFile := filepath.Join("/usr/share/keyrings", repo.Name+".gpg")

	if repo.Action == "add" {
		// Download and add the GPG key
		if err := helpers.DownloadFile(repo.KeyURL, tempKeyFile, true); err != nil {
			return fmt.Errorf("error downloading GPG key: %v", err)
		}

		dearmorCmd := types.Command{
			Exec:     osInfo.Tools.Gpg.Bin,
			Args:     []string{"--yes", "--dearmor", "-o", keyFile, tempKeyFile},
			Elevated: true,
		}
		if err := helpers.RunCommand(dearmorCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error dearmoring GPG key: %v", err)
		}

		// Create the repository list file
		repoListFile := filepath.Join("/etc/apt/sources.list.d", repo.Name+".list")
		repoLine := fmt.Sprintf("deb [arch=%s signed-by=%s] %s %s %s\n", repo.Arch, keyFile, repo.URL, repo.Channel, repo.Component)
		if err := helpers.WriteToFile(repoListFile, repoLine, true); err != nil {
			return fmt.Errorf("error creating repository list file: %v", err)
		}
	} else if repo.Action == "remove" {
		// Remove the repository list file
		repoListFile := filepath.Join("/etc/apt/sources.list.d", repo.Name+".list")
		removeCmd := types.Command{
			Exec:     "rm",
			Args:     []string{repoListFile},
			Elevated: true,
		}
		if err := helpers.RunCommand(removeCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error removing repository list file: %v", err)
		}

		// Remove the GPG key
		removeKeyCmd := types.Command{
			Exec:     "rm",
			Args:     []string{keyFile},
			Elevated: true,
		}
		if err := helpers.RunCommand(removeKeyCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error removing GPG key: %v", err)
		}
	}
	return nil
}

func processBrewRepository(repo types.Repository, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	if repo.Action == "add" {
		addCmd := types.Command{
			Exec:     osInfo.PackageManager.Brew.Bin,
			Args:     []string{"tap", repo.Repository},
			Elevated: osInfo.PackageManager.Brew.Elevated,
		}
		if err := helpers.RunCommand(addCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error adding Homebrew repository: %v", err)
		}
	} else if repo.Action == "remove" {
		removeCmd := types.Command{
			Exec:     osInfo.PackageManager.Brew.Bin,
			Args:     []string{"untap", repo.Repository},
			Elevated: osInfo.PackageManager.Brew.Elevated,
		}
		if err := helpers.RunCommand(removeCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error removing Homebrew repository: %v", err)
		}
	}
	return nil
}

func processDnfYumRepository(repo types.Repository, initConfig *types.InitConfig) error {
	if repo.Action == "add" {
		// Use dnf config-manager to add the repository
		addCmd := types.Command{
			Exec:     "dnf",
			Args:     []string{"config-manager", "--add-repo", repo.URL},
			Elevated: true,
		}
		if err := helpers.RunCommand(addCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error adding DNF/Yum repository: %v", err)
		}

		// Import the GPG key if provided
		if repo.KeyURL != "" {
			importCmd := types.Command{
				Exec:     "rpm",
				Args:     []string{"--import", repo.KeyURL},
				Elevated: true,
			}
			if err := helpers.RunCommand(importCmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error importing GPG key: %v", err)
			}
		}
	} else if repo.Action == "remove" {
		// Remove the repository file
		repoFile := filepath.Join("/etc/yum.repos.d", repo.Name+".repo")
		removeCmd := types.Command{
			Exec:     "rm",
			Args:     []string{"-f", repoFile},
			Elevated: true,
		}
		if err := helpers.RunCommand(removeCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error removing repository file: %v", err)
		}
	}
	return nil
}

func processZypperRepository(repo types.Repository, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	if repo.Action == "add" {
		// Import the GPG key
		importCmd := types.Command{
			Exec:     osInfo.Tools.Rpm.Bin,
			Args:     []string{"--import", repo.KeyURL},
			Elevated: true,
		}
		if err := helpers.RunCommand(importCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error importing GPG key: %v", err)
		}

		// Add the repository
		addCmd := types.Command{
			Exec:     osInfo.PackageManager.Zypper.Bin,
			Args:     []string{"addrepo", repo.URL, repo.Name},
			Elevated: osInfo.PackageManager.Zypper.Elevated,
		}
		if err := helpers.RunCommand(addCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error adding Zypper repository: %v", err)
		}

		// Refresh repositories
		refreshCmd := types.Command{
			Exec:     osInfo.PackageManager.Zypper.Bin,
			Args:     []string{"refresh"},
			Elevated: osInfo.PackageManager.Zypper.Elevated,
		}
		if err := helpers.RunCommand(refreshCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error refreshing Zypper repositories: %v", err)
		}
	} else if repo.Action == "remove" {
		// Remove the repository
		removeCmd := types.Command{
			Exec:     osInfo.PackageManager.Zypper.Bin,
			Args:     []string{"removerepo", repo.Name},
			Elevated: osInfo.PackageManager.Zypper.Elevated,
		}
		if err := helpers.RunCommand(removeCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error removing Zypper repository: %v", err)
		}
	}
	return nil
}

func processPacmanRepository(repo types.Repository, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	pacmanConf := "/etc/pacman.conf"

	if repo.Action == "add" {
		// Check if the repository already exists
		content, err := os.ReadFile(pacmanConf)
		if err != nil {
			return fmt.Errorf("error reading pacman.conf: %v", err)
		}

		repoSection := fmt.Sprintf("[%s]", repo.Name)
		existingConfig := string(content)

		if strings.Contains(existingConfig, repoSection) {
			// Repository already exists, update it
			newRepoConfig := fmt.Sprintf("[%s]\nServer = %s\n", repo.Name, repo.URL)
			updatedConfig := regexp.MustCompile(fmt.Sprintf(`(?s)\[%s\].*?\n\[`, repo.Name)).
				ReplaceAllString(existingConfig, newRepoConfig+"[")

			if err := helpers.WriteToFile(pacmanConf, updatedConfig, true); err != nil {
				return fmt.Errorf("error updating Pacman repository: %v", err)
			}
		} else {
			// Repository doesn't exist, add it
			repoLine := fmt.Sprintf("\n[%s]\nServer = %s\n", repo.Name, repo.URL)
			if err := helpers.AppendToFile(pacmanConf, repoLine, true); err != nil {
				return fmt.Errorf("error adding Pacman repository: %v", err)
			}
		}

		// Refresh the package database
		refreshCmd := types.Command{
			Exec:     osInfo.PackageManager.Pacman.Bin,
			Args:     []string{"-Sy"},
			Elevated: osInfo.PackageManager.Pacman.Elevated,
		}
		if err := helpers.RunCommand(refreshCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error refreshing Pacman package database: %v", err)
		}
	} else if repo.Action == "remove" {
		// Remove the repository from pacman.conf
		content, err := os.ReadFile(pacmanConf)
		if err != nil {
			return fmt.Errorf("error reading pacman.conf: %v", err)
		}

		repoSection := fmt.Sprintf("[%s]", repo.Name)
		existingConfig := string(content)

		if strings.Contains(existingConfig, repoSection) {
			updatedConfig := regexp.MustCompile(fmt.Sprintf(`(?s)\[%s\].*?\n\[`, repo.Name)).
				ReplaceAllString(existingConfig, "[")

			if err := helpers.WriteToFile(pacmanConf, updatedConfig, true); err != nil {
				return fmt.Errorf("error removing Pacman repository: %v", err)
			}
		}
	}
	return nil
}

func processChocoRepository(repo types.Repository, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	if repo.Action == "add" {
		// Add the Chocolatey repository
		addCmd := types.Command{
			Exec:     osInfo.PackageManager.Chocolatey.Bin,
			Args:     []string{"source", "add", "--name", repo.Name, "--source", repo.URL},
			Elevated: osInfo.PackageManager.Chocolatey.Elevated,
		}
		if err := helpers.RunCommand(addCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error adding Chocolatey repository: %v", err)
		}
	} else if repo.Action == "remove" {
		// Remove the Chocolatey repository
		removeCmd := types.Command{
			Exec:     osInfo.PackageManager.Chocolatey.Bin,
			Args:     []string{"source", "remove", "--name", repo.Name},
			Elevated: osInfo.PackageManager.Chocolatey.Elevated,
		}
		if err := helpers.RunCommand(removeCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error removing Chocolatey repository: %v", err)
		}
	}
	return nil
}

func processScoopRepository(repo types.Repository, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	if repo.Action == "add" {
		// Add the Scoop bucket
		addCmd := types.Command{
			Exec:     osInfo.PackageManager.Scoop.Bin,
			Args:     []string{"bucket", "add", repo.Name, repo.URL},
			Elevated: osInfo.PackageManager.Scoop.Elevated,
		}
		if err := helpers.RunCommand(addCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error adding Scoop bucket: %v", err)
		}
	} else if repo.Action == "remove" {
		// Remove the Scoop bucket
		removeCmd := types.Command{
			Exec:     osInfo.PackageManager.Scoop.Bin,
			Args:     []string{"bucket", "rm", repo.Name},
			Elevated: osInfo.PackageManager.Scoop.Elevated,
		}
		if err := helpers.RunCommand(removeCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error removing Scoop bucket: %v", err)
		}
	}
	return nil
}
