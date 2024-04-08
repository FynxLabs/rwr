package processors

import (
	"fmt"
	"github.com/thefynx/rwr/internal/helpers"
	"github.com/thefynx/rwr/internal/processors/types"
	"os"
	"path/filepath"
)

func ProcessRepositoriesFromFile(blueprintFile string, osInfo types.OSInfo) error {
	var repositories []types.Repository

	// Read the blueprint file based on the file format
	switch filepath.Ext(blueprintFile) {
	case ".yaml", ".yml":
		err := helpers.ReadYAMLFile(blueprintFile, &repositories)
		if err != nil {
			return fmt.Errorf("error reading repository blueprint file: %w", err)
		}
	case ".json":
		err := helpers.ReadJSONFile(blueprintFile, &repositories)
		if err != nil {
			return fmt.Errorf("error reading repository blueprint file: %w", err)
		}
	case ".toml":
		err := helpers.ReadTOMLFile(blueprintFile, &repositories)
		if err != nil {
			return fmt.Errorf("error reading repository blueprint file: %w", err)
		}
	default:
		return fmt.Errorf("unsupported blueprint file format: %s", filepath.Ext(blueprintFile))
	}

	// Process the repositories
	err := ProcessRepositories(repositories, osInfo)
	if err != nil {
		return fmt.Errorf("error processing repositories: %w", err)
	}

	return nil
}

func ProcessRepositories(repositories []types.Repository, osInfo types.OSInfo) error {
	for _, repo := range repositories {
		switch repo.PackageManager {
		case "apt":
			if err := processAptRepository(repo, osInfo); err != nil {
				return err
			}
		case "brew":
			if err := processBrewRepository(repo, osInfo); err != nil {
				return err
			}
		case "dnf", "yum":
			if err := processDnfYumRepository(repo, osInfo); err != nil {
				return err
			}
		case "zypper":
			if err := processZypperRepository(repo, osInfo); err != nil {
				return err
			}
		case "pacman":
			if err := processPacmanRepository(repo, osInfo); err != nil {
				return err
			}
		case "choco":
			if err := processChocoRepository(repo, osInfo); err != nil {
				return err
			}
		case "scoop":
			if err := processScoopRepository(repo, osInfo); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported package manager: %s", repo.PackageManager)
		}
	}
	return nil
}

func processAptRepository(repo types.Repository, osInfo types.OSInfo) error {
	keyFile := filepath.Join("/usr/share/keyrings", repo.Name+".gpg")

	if repo.Action == "add" {
		// Download and add the GPG key
		if err := helpers.DownloadFile(repo.KeyURL, keyFile); err != nil {
			return fmt.Errorf("error downloading GPG key: %v", err)
		}
		if err := helpers.RunCommand("gpg", "--dearmor", "-o", keyFile, keyFile); err != nil {
			return fmt.Errorf("error dearmoring GPG key: %v", err)
		}

		// Create the repository list file
		repoListFile := filepath.Join("/etc/apt/sources.list.d", repo.Name+".list")
		repoLine := fmt.Sprintf("deb [arch=%s signed-by=%s] %s %s %s\n", repo.Arch, keyFile, repo.URL, repo.Channel, repo.Component)
		if err := os.WriteFile(repoListFile, []byte(repoLine), 0644); err != nil {
			return fmt.Errorf("error creating repository list file: %v", err)
		}

		// Update package lists
		if err := helpers.RunCommand(osInfo.PackageManager.Apt.Bin, osInfo.PackageManager.Apt.Update); err != nil {
			return fmt.Errorf("error updating package lists: %v", err)
		}
	} else if repo.Action == "remove" {
		// Remove the repository list file
		repoListFile := filepath.Join("/etc/apt/sources.list.d", repo.Name+".list")
		if err := os.Remove(repoListFile); err != nil {
			return fmt.Errorf("error removing repository list file: %v", err)
		}

		// Remove the GPG key
		if err := os.Remove(keyFile); err != nil {
			return fmt.Errorf("error removing GPG key: %v", err)
		}

		// Update package lists
		if err := helpers.RunCommand(osInfo.PackageManager.Apt.Bin, osInfo.PackageManager.Apt.Update); err != nil {
			return fmt.Errorf("error updating package lists: %v", err)
		}
	}
	return nil
}

func processBrewRepository(repo types.Repository, osInfo types.OSInfo) error {
	if repo.Action == "add" {
		if err := helpers.RunCommand("brew", "tap", repo.Repository); err != nil {
			return fmt.Errorf("error adding Homebrew repository: %v", err)
		}
	} else if repo.Action == "remove" {
		if err := helpers.RunCommand("brew", "untap", repo.Repository); err != nil {
			return fmt.Errorf("error removing Homebrew repository: %v", err)
		}
	}
	return nil
}

func processDnfYumRepository(repo types.Repository, osInfo types.OSInfo) error {
	if repo.Action == "add" {
		// Download the repository file
		repoFile := filepath.Join("/etc/yum.repos.d", repo.Name+".repo")
		if err := helpers.DownloadFile(repo.Repository, repoFile); err != nil {
			return fmt.Errorf("error downloading repository file: %v", err)
		}

		// Import the GPG key
		if err := helpers.RunCommand("rpm", "--import", repo.KeyURL); err != nil {
			return fmt.Errorf("error importing GPG key: %v", err)
		}
	} else if repo.Action == "remove" {
		// Remove the repository file
		repoFile := filepath.Join("/etc/yum.repos.d", repo.Name+".repo")
		if err := os.Remove(repoFile); err != nil {
			return fmt.Errorf("error removing repository file: %v", err)
		}
	}
	return nil
}

func processZypperRepository(repo types.Repository, osInfo types.OSInfo) error {
	if repo.Action == "add" {
		// Add the repository
		if err := helpers.RunCommand("zypper", "addrepo", repo.URL, repo.Name); err != nil {
			return fmt.Errorf("error adding Zypper repository: %v", err)
		}

		// Refresh repositories
		if err := helpers.RunCommand("zypper", "refresh"); err != nil {
			return fmt.Errorf("error refreshing Zypper repositories: %v", err)
		}
	} else if repo.Action == "remove" {
		// Remove the repository
		if err := helpers.RunCommand("zypper", "removerepo", repo.Name); err != nil {
			return fmt.Errorf("error removing Zypper repository: %v", err)
		}
	}
	return nil
}

func processPacmanRepository(repo types.Repository, osInfo types.OSInfo) error {
	pacmanConf := "/etc/pacman.conf"

	if repo.Action == "add" {
		// Add the repository to pacman.conf
		repoLine := fmt.Sprintf("\n[%s]\nServer = %s\n", repo.Name, repo.URL)
		if err := helpers.AppendToFile(pacmanConf, repoLine); err != nil {
			return fmt.Errorf("error adding Pacman repository: %v", err)
		}

		// Refresh the package database
		if err := helpers.RunCommand("pacman", "-Sy"); err != nil {
			return fmt.Errorf("error refreshing Pacman package database: %v", err)
		}
	} else if repo.Action == "remove" {
		// Remove the repository from pacman.conf
		if err := helpers.RemoveLineFromFile(pacmanConf, fmt.Sprintf("[%s]", repo.Name)); err != nil {
			return fmt.Errorf("error removing Pacman repository: %v", err)
		}
	}
	return nil
}

func processChocoRepository(repo types.Repository, osInfo types.OSInfo) error {
	if repo.Action == "add" {
		// Add the Chocolatey repository
		if err := helpers.RunCommand("choco", "source", "add", "--name", repo.Name, "--source", repo.URL); err != nil {
			return fmt.Errorf("error adding Chocolatey repository: %v", err)
		}
	} else if repo.Action == "remove" {
		// Remove the Chocolatey repository
		if err := helpers.RunCommand("choco", "source", "remove", "--name", repo.Name); err != nil {
			return fmt.Errorf("error removing Chocolatey repository: %v", err)
		}
	}
	return nil
}

func processScoopRepository(repo types.Repository, osInfo types.OSInfo) error {
	if repo.Action == "add" {
		// Add the Scoop bucket
		if err := helpers.RunCommand("scoop", "bucket", "add", repo.Name, repo.URL); err != nil {
			return fmt.Errorf("error adding Scoop bucket: %v", err)
		}
	} else if repo.Action == "remove" {
		// Remove the Scoop bucket
		if err := helpers.RunCommand("scoop", "bucket", "rm", repo.Name); err != nil {
			return fmt.Errorf("error removing Scoop bucket: %v", err)
		}
	}
	return nil
}
