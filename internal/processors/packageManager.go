// packageManagerProcessor.go

package processors

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/helpers"
	"github.com/thefynx/rwr/internal/processors/types"
	"os"
	"os/user"
	"path/filepath"
)

func ProcessPackageManagers(packageManagers []types.PackageManagerInfo, osInfo types.OSInfo) error {
	for _, pm := range packageManagers {
		switch pm.Name {
		case "brew":
			if err := processBrew(pm, osInfo); err != nil {
				return err
			}
		case "nix":
			if err := processNix(pm, osInfo); err != nil {
				return err
			}
		case "chocolatey":
			if err := processChocolatey(pm, osInfo); err != nil {
				return err
			}
		case "scoop":
			if err := processScoop(pm, osInfo); err != nil {
				return err
			}
		case "yay", "paru", "trizen", "yaourt", "pamac", "aura":
			if err := processAURManager(pm, osInfo); err != nil {
				return err
			}
		case "npm", "pnpm", "yarn":
			if err := processNodePackageManager(pm, osInfo); err != nil {
				return err
			}
		case "pip":
			if err := processPip(pm, osInfo); err != nil {
				return err
			}
		case "gem":
			if err := processGem(pm, osInfo); err != nil {
				return err
			}
		case "cargo":
			if err := processCargo(pm, osInfo); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported package manager: %s", pm.Name)
		}
	}

	return nil
}

func processBrew(pm types.PackageManagerInfo, osInfo types.OSInfo) error {
	if osInfo.OS == "linux" || osInfo.OS == "macos" {
		if pm.Action == "install" {
			var installCmd string
			if osInfo.OS == "macos" && pm.AsUser != "" {
				installCmd = fmt.Sprintf("sudo -u %s /bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"", pm.AsUser)
			} else {
				installCmd = "/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
			}
			if err := helpers.RunCommand("bash", "-c", installCmd); err != nil {
				return fmt.Errorf("error installing Homebrew: %v", err)
			}
		} else if pm.Action == "remove" {
			if err := helpers.RunCommand("bash", "-c", "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/uninstall.sh)"); err != nil {
				return fmt.Errorf("error removing Homebrew: %v", err)
			}
		}
	}
	return nil
}

func processNix(pm types.PackageManagerInfo, osInfo types.OSInfo) error {
	if osInfo.OS == "linux" || osInfo.OS == "macos" {
		if pm.Action == "install" {
			if err := helpers.RunCommand("bash", "-c", "$(curl -L https://nixos.org/nix/install)"); err != nil {
				return fmt.Errorf("error installing Nix: %v", err)
			}
		} else if pm.Action == "remove" {
			if err := helpers.RunCommand("bash", "-c", "$(curl -L https://nixos.org/nix/uninstall)"); err != nil {
				return fmt.Errorf("error removing Nix: %v", err)
			}
		}
	}
	return nil
}

func processChocolatey(pm types.PackageManagerInfo, osInfo types.OSInfo) error {
	if osInfo.OS == "windows" {
		if pm.Action == "install" {
			if err := helpers.RunWithElevatedPrivileges("powershell", "-Command", "Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))"); err != nil {
				return fmt.Errorf("error installing Chocolatey: %v", err)
			}
		} else if pm.Action == "remove" {
			if err := helpers.RunWithElevatedPrivileges("powershell", "-Command", "choco uninstall chocolatey -y"); err != nil {
				return fmt.Errorf("error removing Chocolatey: %v", err)
			}
		}
	}
	return nil
}

func processScoop(pm types.PackageManagerInfo, osInfo types.OSInfo) error {
	if osInfo.OS == "windows" {
		if pm.Action == "install" {
			if err := helpers.RunCommand("powershell", "-Command", "Set-ExecutionPolicy RemoteSigned -Scope CurrentUser -Force; iex (New-Object System.Net.WebClient).DownloadString('https://get.scoop.sh')"); err != nil {
				return fmt.Errorf("error installing Scoop: %v", err)
			}
		} else if pm.Action == "remove" {
			if err := helpers.RunCommand("powershell", "-Command", "scoop uninstall scoop -p"); err != nil {
				return fmt.Errorf("error removing Scoop: %v", err)
			}
		}
	}
	return nil
}

func processAURManager(pm types.PackageManagerInfo, osInfo types.OSInfo) error {
	if osInfo.OS == "linux" {
		if pm.Action == "install" {
			var installCmd string
			switch pm.Name {
			case "yay":
				installCmd = "pacman -S --needed git base-devel && git clone https://aur.archlinux.org/yay.git && cd yay && makepkg -si"
			case "paru":
				installCmd = "pacman -S --needed git base-devel && git clone https://aur.archlinux.org/paru.git && cd paru && makepkg -si"
			case "trizen":
				installCmd = "pacman -S --needed git base-devel && git clone https://aur.archlinux.org/trizen.git && cd trizen && makepkg -si"
			case "yaourt":
				installCmd = "pacman -S --needed git base-devel && git clone https://aur.archlinux.org/packages/yaourt && cd yaourt && makepkg -si"
			case "pamac":
				installCmd = "pacman -S --needed base-devel git && git clone https://aur.archlinux.org/pamac-aur.git && cd pamac-aur && makepkg -si"
			case "aura":
				installCmd = "pacman -S --needed base-devel git && git clone https://aur.archlinux.org/aura-bin.git && cd aura-bin && makepkg -si"
			}
			if err := helpers.RunCommand("bash", "-c", installCmd); err != nil {
				return fmt.Errorf("error installing %s: %v", pm.Name, err)
			}
		} else if pm.Action == "remove" {
			var removeCmd string
			switch pm.Name {
			case "yay":
				removeCmd = "yay -Rns yay"
			case "paru":
				removeCmd = "paru -Rns paru"
			case "trizen":
				removeCmd = "trizen -Rns trizen"
			case "yaourt":
				removeCmd = "yaourt -Rns yaourt"
			case "pamac":
				removeCmd = "pamac remove pamac-aur"
			case "aura":
				removeCmd = "aura -Rns aura-bin"
			}
			if err := helpers.RunCommand("bash", "-c", removeCmd); err != nil {
				return fmt.Errorf("error removing %s: %v", pm.Name, err)
			}
		}
	}
	return nil
}

func processNodePackageManager(pm types.PackageManagerInfo, osInfo types.OSInfo) error {
	if pm.Action == "install" {
		var installCmd string
		switch pm.Name {
		case "npm":
			installCmd = "curl -fsSL https://install.npmjs.com | bash"
		case "pnpm":
			installCmd = "curl -fsSL https://get.pnpm.io/install.sh | bash"
		case "yarn":
			installCmd = "curl -fsSL https://yarnpkg.com/install.sh | bash"
		}
		if err := helpers.RunCommand("bash", "-c", installCmd); err != nil {
			return fmt.Errorf("error installing %s: %v", pm.Name, err)
		}
	} else if pm.Action == "remove" {
		var removeCmd string
		switch pm.Name {
		case "npm":
			removeCmd = "npm uninstall -g npm"
		case "pnpm":
			removeCmd = "pnpm self-uninstall"
		case "yarn":
			removeCmd = "yarn global remove yarn"
		}
		if err := helpers.RunCommand("bash", "-c", removeCmd); err != nil {
			return fmt.Errorf("error removing %s: %v", pm.Name, err)
		}
	}
	return nil
}

func processPip(pm types.PackageManagerInfo, osInfo types.OSInfo) error {
	if pm.Action == "install" {
		if err := helpers.RunCommand("bash", "-c", "curl https://bootstrap.pypa.io/get-pip.py | python"); err != nil {
			return fmt.Errorf("error installing pip: %v", err)
		}
	} else if pm.Action == "remove" {
		if err := helpers.RunCommand("bash", "-c", "python -m pip uninstall pip -y"); err != nil {
			return fmt.Errorf("error removing pip: %v", err)
		}
	}
	return nil
}

func processGem(pm types.PackageManagerInfo, osInfo types.OSInfo) error {
	if pm.Action == "install" {
		if err := helpers.RunCommand("bash", "-c", "gem update --system"); err != nil {
			return fmt.Errorf("error updating Ruby Gems: %v", err)
		}
	} else if pm.Action == "remove" {
		if err := helpers.RunCommand("bash", "-c", "gem uninstall rubygems-update"); err != nil {
			return fmt.Errorf("error removing Ruby Gems: %v", err)
		}
	}
	return nil
}

func processCargo(pm types.PackageManagerInfo, osInfo types.OSInfo) error {
	if pm.Action == "install" {
		// Install Rust and Cargo
		installCmd := "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y"
		if pm.AsUser != "" {
			installCmd = fmt.Sprintf("sudo -u %s %s", pm.AsUser, installCmd)
		}
		if err := helpers.RunCommand("bash", "-c", installCmd); err != nil {
			return fmt.Errorf("error installing Cargo: %v", err)
		}

		// Update PATH environment variable
		cargoPath := filepath.Join(os.Getenv("HOME"), ".cargo", "bin")
		if pm.AsUser != "" {
			u, err := user.Lookup(pm.AsUser)
			if err != nil {
				log.Warnf("Error getting user information for %s: %v", pm.AsUser, err)
				return fmt.Errorf("error getting user information for %s: %v", pm.AsUser, err)
			}
			cargoPath = filepath.Join(u.HomeDir, ".cargo", "bin")
		}
		err := os.Setenv("PATH", fmt.Sprintf("%s:%s", cargoPath, os.Getenv("PATH")))
		if err != nil {
			return err
		}

		// Install cargo-update
		if err := helpers.RunCommand(filepath.Join(cargoPath, "cargo"), "install", "cargo-update"); err != nil {
			log.Warnf("Error installing cargo-update: %v", err)
			// Continue execution even if cargo-update installation fails
		}
	} else if pm.Action == "remove" {
		// Update PATH environment variable
		cargoPath := filepath.Join(os.Getenv("HOME"), ".cargo", "bin")
		if pm.AsUser != "" {
			u, err := user.Lookup(pm.AsUser)
			if err != nil {
				log.Warnf("Error getting user information for %s: %v", pm.AsUser, err)
				return fmt.Errorf("error getting user information for %s: %v", pm.AsUser, err)
			}
			cargoPath = filepath.Join(u.HomeDir, ".cargo", "bin")
		}
		err := os.Setenv("PATH", fmt.Sprintf("%s:%s", cargoPath, os.Getenv("PATH")))
		if err != nil {
			return err
		}

		// Uninstall cargo-update
		if err := helpers.RunCommand(filepath.Join(cargoPath, "cargo"), "uninstall", "cargo-update"); err != nil {
			log.Warnf("Error uninstalling cargo-update: %v", err)
			// Continue execution even if cargo-update uninstallation fails
		}

		// Uninstall Rust and Cargo
		uninstallCmd := "rustup self uninstall -y"
		if pm.AsUser != "" {
			uninstallCmd = fmt.Sprintf("sudo -u %s %s", pm.AsUser, uninstallCmd)
		}
		if err := helpers.RunCommand("bash", "-c", uninstallCmd); err != nil {
			return fmt.Errorf("error removing Cargo: %v", err)
		}
	}
	return nil
}
