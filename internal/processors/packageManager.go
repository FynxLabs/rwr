package processors

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
)

func ProcessPackageManagers(packageManagers []types.PackageManagerInfo, osInfo *types.OSInfo, initConfig *types.InitConfig) error {

	log.Infof("Installing package manager common dependencies")

	// Install OpenSSL
	log.Infof("Installing OpenSSL")
	err := helpers.InstallOpenSSL(osInfo, initConfig)
	if err != nil {
		return fmt.Errorf("error installing OpenSSL: %v", err)
	}

	// Install build essentials
	log.Infof("Installing build essentials")
	err = helpers.InstallBuildEssentials(osInfo, initConfig)
	if err != nil {
		return fmt.Errorf("error installing build essentials: %v", err)
	}

	for _, pm := range packageManagers {
		switch pm.Name {
		case "brew":
			if err := processBrew(pm, osInfo, initConfig); err != nil {
				return err
			}
		case "nix":
			if err := processNix(pm, osInfo, initConfig); err != nil {
				return err
			}
		case "chocolatey":
			if err := processChocolatey(pm, osInfo, initConfig); err != nil {
				return err
			}
		case "scoop":
			if err := processScoop(pm, osInfo, initConfig); err != nil {
				return err
			}
		case "yay", "paru", "trizen", "yaourt", "pamac", "aura":
			if err := processAURManager(pm, osInfo, initConfig); err != nil {
				return err
			}
		case "npm", "pnpm", "yarn":
			if err := processNodePackageManager(pm, osInfo, initConfig); err != nil {
				return err
			}
		case "pip":
			if err := processPip(pm, osInfo, initConfig); err != nil {
				return err
			}
		case "gem":
			if err := processGem(pm, osInfo, initConfig); err != nil {
				return err
			}
		case "cargo":
			if err := processCargo(pm, osInfo, initConfig); err != nil {
				return err
			}
		case "winget":
			if err := processWinget(pm, osInfo, initConfig); err != nil {
				return err
			}
		case "gnome-extensions":
			if err := processGnomeExtensionsCLI(pm, osInfo, initConfig); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported package manager: %s", pm.Name)
		}
	}
	return nil
}

func processBrew(pm types.PackageManagerInfo, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	if osInfo.System.OS == "linux" || osInfo.System.OS == "macos" {
		if pm.Action == "install" {
			if helpers.FindTool("brew").Exists {
				log.Infof("Homebrew is already installed")
				return nil
			}

			// Create a temporary file for the installation script
			tmpFile, err := os.CreateTemp("", "homebrew-install-*.sh")
			if err != nil {
				return fmt.Errorf("error creating temporary file: %v", err)
			}
			defer func(name string) {
				err := os.Remove(name)
				if err != nil {
					log.Warnf("Error removing temporary file %s: %v", name, err)
				}
			}(tmpFile.Name())

			osInfo.Tools.Curl = helpers.FindTool("curl")

			if osInfo.Tools.Curl.Bin == "" {
				log.Warn("Brew Install: Curl not found")
				return nil
			}

			// Download the Homebrew installation script
			downloadCmd := types.Command{
				Exec: osInfo.Tools.Curl.Bin,
				Args: []string{
					"-fsSL",
					"-o",
					tmpFile.Name(),
					"https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh",
				},
			}
			if err := helpers.RunCommand(downloadCmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error downloading Homebrew installation script: %v", err)
			}

			// Make the installation script executable
			chmodCmd := types.Command{
				Exec: "chmod",
				Args: []string{
					"+x",
					tmpFile.Name(),
				},
			}
			if err := helpers.RunCommand(chmodCmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error making Homebrew installation script executable: %v", err)
			}

			log.Infof("Installing Homebrew")
			// Run the installation script
			installCmd := types.Command{
				Exec: osInfo.Tools.Bash.Bin,
				Args: []string{
					tmpFile.Name(),
				},
				Interactive: true,
				AsUser:      pm.AsUser,
			}
			err = helpers.RunCommand(installCmd, initConfig.Variables.Flags.Debug)
			if err != nil {
				return fmt.Errorf("error installing Homebrew: %v", err)
			}

			var brewPath string

			if osInfo.System.OS == "linux" {
				brewPath = "/home/linuxbrew/.linuxbrew/bin/"
			} else if osInfo.System.OS == "macos" {
				currentUser, err := user.Current()
				if err != nil {
					log.Warnf("Error getting current user: %v", err)
					return err
				}
				brewPath = filepath.Join(currentUser.HomeDir, ".brew/bin/")
			}

			log.Infof("Homebrew installed successfully")
			// Populate the Brew package manager struct based on the operating system
			osInfo.PackageManager.Brew = types.PackageManagerInfo{
				Name:     "brew",
				Bin:      filepath.Join(brewPath, "brew"),
				List:     filepath.Join(brewPath, "brew list"),
				Search:   filepath.Join(brewPath, "brew search"),
				Install:  filepath.Join(brewPath, "brew install -fq"),
				Remove:   filepath.Join(brewPath, "brew uninstall -fq"),
				Update:   filepath.Join(brewPath, "brew update && ", brewPath, "brew upgrade"),
				Clean:    filepath.Join(brewPath, "brew cleanup -q"),
				Elevated: false,
			}
		} else if pm.Action == "remove" {
			// Create a temporary file for the removal script
			tmpFile, err := os.CreateTemp("", "homebrew-uninstall-*.sh")
			if err != nil {
				return fmt.Errorf("error creating temporary file: %v", err)
			}
			defer func(name string) {
				err := os.Remove(name)
				if err != nil {
					log.Warnf("Error removing temporary file %s: %v", name, err)
				}
			}(tmpFile.Name())

			// Download the Homebrew removal script
			downloadCmd := types.Command{
				Exec: osInfo.Tools.Curl.Bin,
				Args: []string{
					"-fsSL",
					"-o",
					tmpFile.Name(),
					"https://raw.githubusercontent.com/Homebrew/install/HEAD/uninstall.sh",
				},
			}
			if err := helpers.RunCommand(downloadCmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error downloading Homebrew removal script: %v", err)
			}

			// Make the removal script executable
			chmodCmd := types.Command{
				Exec: "chmod",
				Args: []string{
					"+x",
					tmpFile.Name(),
				},
			}
			if err := helpers.RunCommand(chmodCmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error making Homebrew removal script executable: %v", err)
			}

			// Run the removal script
			removeCmd := types.Command{
				Exec: osInfo.Tools.Bash.Bin,
				Args: []string{
					tmpFile.Name(),
				},
			}
			if err := helpers.RunCommand(removeCmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error removing Homebrew: %v", err)
			}
		}
	}
	return nil
}

func processNix(pm types.PackageManagerInfo, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	if osInfo.System.OS == "linux" || osInfo.System.OS == "macos" {
		if pm.Action == "install" {
			if helpers.FindTool("nix").Exists {
				log.Infof("Nix is already installed")
				return nil
			}
			// TODO: No Curl Bash, create installer
			installCmd := types.Command{
				Exec: osInfo.Tools.Bash.Bin,
				Args: []string{
					"-c",
					"curl -L https://nixos.org/nix/install | sh",
				},
			}
			if err := helpers.RunCommand(installCmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error installing Nix: %v", err)
			}

			log.Infof("Nix installed successfully")
			nixBinDir := "/nix/var/nix/profiles/default/bin/"
			osInfo.PackageManager.Nix = types.PackageManagerInfo{
				Name:     "nix",
				Bin:      filepath.Join(nixBinDir, "nix-env"),
				List:     filepath.Join(nixBinDir, "nix-env -q"),
				Search:   filepath.Join(nixBinDir, "nix search"),
				Install:  filepath.Join(nixBinDir, "nix-env -i"),
				Remove:   filepath.Join(nixBinDir, "nix-env -e"),
				Update:   filepath.Join(nixBinDir, "nix-channel --update && ", nixBinDir, "nix-env -u '*'"),
				Clean:    filepath.Join(nixBinDir, "nix-collect-garbage -d"),
				Elevated: false,
			}
		} else if pm.Action == "remove" {
			// TODO: No Curl Bash, create uninstaller
			removeCmd := types.Command{
				Exec: osInfo.Tools.Bash.Bin,
				Args: []string{
					"-c",
					"curl -L https://nixos.org/nix/uninstall | sh",
				},
			}
			if err := helpers.RunCommand(removeCmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error removing Nix: %v", err)
			}
		}
	}
	return nil
}

func processChocolatey(pm types.PackageManagerInfo, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	if osInfo.System.OS == "windows" {
		if pm.Action == "install" {
			if helpers.FindTool("choco").Exists {
				log.Infof("Chocolatey is already installed")
				return nil
			}
			installCmd := types.Command{
				Exec: osInfo.Tools.PowerShell.Bin,
				Args: []string{
					"-Command",
					"Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))",
				},
				Elevated: true,
			}
			if err := helpers.RunCommand(installCmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error installing Chocolatey: %v", err)
			}

			log.Infof("Chocolatey installed successfully")
			// Populate the Chocolatey package manager struct
			osInfo.PackageManager.Chocolatey = types.PackageManagerInfo{
				Name:     "chocolatey",
				Bin:      "choco",
				List:     "choco list --local-only",
				Search:   "choco search",
				Install:  "choco install -y",
				Remove:   "choco uninstall -y",
				Update:   "choco upgrade all -y",
				Clean:    "choco uninstall --allversions -y",
				Elevated: true,
			}
		} else if pm.Action == "remove" {
			removeCmd := types.Command{
				Exec: osInfo.Tools.PowerShell.Bin,
				Args: []string{
					"-Command",
					"choco uninstall chocolatey -y",
				},
				Elevated: true,
			}
			if err := helpers.RunCommand(removeCmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error removing Chocolatey: %v", err)
			}
		}
	}
	return nil
}

func processWinget(pm types.PackageManagerInfo, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	if osInfo.System.OS == "windows" {
		if pm.Action == "install" {
			if helpers.FindTool("winget").Exists {
				log.Infof("Winget is already installed")
				return nil
			}
			installCmd := types.Command{
				Exec: osInfo.Tools.PowerShell.Bin,
				Args: []string{
					"-Command",
					"Invoke-WebRequest -Uri https://github.com/microsoft/winget-cli/releases/download/v1.5.9371.0/Microsoft.DesktopAppInstaller_8wekyb3d8bbwe.appxbundle -OutFile winget.appxbundle; Add-AppxPackage -Path winget.appxbundle",
				},
				Elevated: true,
			}
			if err := helpers.RunCommand(installCmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error installing Winget: %v", err)
			}

			log.Infof("Winget installed successfully")
			// Populate the Winget package manager struct
			osInfo.PackageManager.Winget = types.PackageManagerInfo{
				Name:     "winget",
				Bin:      "winget",
				List:     "winget list --installed",
				Search:   "winget search",
				Install:  "winget install --silent",
				Remove:   "winget uninstall --silent",
				Update:   "winget upgrade --all",
				Clean:    "winget source reset",
				Elevated: false,
			}
		} else if pm.Action == "remove" {
			removeCmd := types.Command{
				Exec: osInfo.Tools.PowerShell.Bin,
				Args: []string{
					"-Command",
					"Get-AppxPackage Microsoft.DesktopAppInstaller | Remove-AppxPackage",
				},
				Elevated: true,
			}
			if err := helpers.RunCommand(removeCmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error removing Winget: %v", err)
			}
		}
	}
	return nil
}

func processScoop(pm types.PackageManagerInfo, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	if osInfo.System.OS == "windows" {
		if pm.Action == "install" {
			if helpers.FindTool("scoop").Exists {
				log.Infof("Scoop is already installed")
				return nil
			}
			installCmd := types.Command{
				Exec: osInfo.Tools.PowerShell.Bin,
				Args: []string{
					"-Command",
					"Set-ExecutionPolicy RemoteSigned -Scope CurrentUser -Force; iex (New-Object System.Net.WebClient).DownloadString('https://get.scoop.sh')",
				},
			}
			if err := helpers.RunCommand(installCmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error installing Scoop: %v", err)
			}

			log.Infof("Scoop installed successfully")
			// Populate the Scoop package manager struct
			osInfo.PackageManager.Scoop = types.PackageManagerInfo{
				Name:     "scoop",
				Bin:      "scoop",
				List:     "scoop list",
				Search:   "scoop search",
				Install:  "scoop install",
				Remove:   "scoop uninstall",
				Update:   "scoop update",
				Clean:    "scoop cleanup",
				Elevated: false,
			}
		} else if pm.Action == "remove" {
			removeCmd := types.Command{
				Exec: osInfo.Tools.PowerShell.Bin,
				Args: []string{
					"-Command",
					"scoop uninstall scoop -p",
				},
			}
			if err := helpers.RunCommand(removeCmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error removing Scoop: %v", err)
			}
			log.Infof("Scoop removed successfully")
		}
	}
	log.Warnf("Scoop is only available on Windows")
	return nil
}

func processAURManager(pm types.PackageManagerInfo, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	if osInfo.System.OS == "linux" {
		if pm.Action == "install" {
			var installed bool
			switch pm.Name {
			case "yay":
				installed = helpers.FindTool("yay").Exists
			case "paru":
				installed = helpers.FindTool("paru").Exists
			case "trizen":
				installed = helpers.FindTool("trizen").Exists
			case "yaourt":
				installed = helpers.FindTool("yaourt").Exists
			case "pamac":
				installed = helpers.FindTool("pamac").Exists
			case "aura":
				installed = helpers.FindTool("aura").Exists
			}
			if installed {
				log.Infof("%s is already installed", pm.Name)
				return nil
			}

			// Create temporary directory with unique path
			tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("rwr-aur-%s", pm.Name))

			// Remove the directory if it already exists
			if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
				if err := os.RemoveAll(tempDir); err != nil {
					return fmt.Errorf("error removing existing temporary directory: %v", err)
				}
			}

			// Create the new directory
			if err := os.MkdirAll(tempDir, 0755); err != nil {
				return fmt.Errorf("error creating temporary directory: %v", err)
			}

			// Install dependencies
			depsCmd := types.Command{
				Exec:     "pacman",
				Args:     []string{"-S", "--needed", "--noconfirm", "git", "base-devel"},
				Elevated: true,
			}
			if err := helpers.RunCommand(depsCmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error installing dependencies for %s: %v", pm.Name, err)
			}

			// Clone the repository
			var repoURL string
			switch pm.Name {
			case "yay":
				repoURL = "https://aur.archlinux.org/yay.git"
			case "paru":
				repoURL = "https://aur.archlinux.org/paru.git"
			case "trizen":
				repoURL = "https://aur.archlinux.org/trizen.git"
			case "yaourt":
				repoURL = "https://aur.archlinux.org/packages/yaourt"
			case "pamac":
				repoURL = "https://aur.archlinux.org/pamac-aur.git"
			case "aura":
				repoURL = "https://aur.archlinux.org/aura-bin.git"
			}

			cloneCmd := types.Command{
				Exec: "git",
				Args: []string{"clone", repoURL, tempDir},
			}

			if err := helpers.RunCommand(cloneCmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error cloning %s repository: %v", pm.Name, err)
			}

			// Build and install package
			buildCmd := types.Command{
				Exec: osInfo.Tools.Bash.Bin,
				Args: []string{"-c", "cd", tempDir, "makepkg -si --noconfirm"},
			}
			if err := helpers.RunCommand(buildCmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error building %s: %v", pm.Name, err)
			}

			log.Infof("%s installed successfully", pm.Name)
		} else if pm.Action == "remove" {
			var removeCmd types.Command
			switch pm.Name {
			case "yay":
				removeCmd = types.Command{
					Exec:     "pacman",
					Args:     []string{"-Rns", "--noconfirm", "yay"},
					Elevated: true,
				}
			case "paru":
				removeCmd = types.Command{
					Exec:     "pacman",
					Args:     []string{"-Rns", "--noconfirm", "paru"},
					Elevated: true,
				}
			case "trizen":
				removeCmd = types.Command{
					Exec:     "pacman",
					Args:     []string{"-Rns", "--noconfirm", "trizen"},
					Elevated: true,
				}
			case "yaourt":
				removeCmd = types.Command{
					Exec:     "pacman",
					Args:     []string{"-Rns", "--noconfirm", "yaourt"},
					Elevated: true,
				}
			case "pamac":
				removeCmd = types.Command{
					Exec:     "pacman",
					Args:     []string{"-Rns", "--noconfirm", "pamac-aur"},
					Elevated: true,
				}
			case "aura":
				removeCmd = types.Command{
					Exec:     "pacman",
					Args:     []string{"-Rns", "--noconfirm", "aura-bin"},
					Elevated: true,
				}
			}
			if err := helpers.RunCommand(removeCmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error removing %s: %v", pm.Name, err)
			}
			log.Infof("%s removed successfully", pm.Name)
		}
	} else {
		log.Warnf("AUR managers are only available on Linux")
	}
	return nil
}

func processNodePackageManager(pm types.PackageManagerInfo, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	if pm.Action == "install" {
		var installed bool
		switch pm.Name {
		case "npm":
			installed = helpers.FindTool("npm").Exists
		case "pnpm":
			installed = helpers.FindTool("pnpm").Exists
		case "yarn":
			installed = helpers.FindTool("yarn").Exists
		}
		if installed {
			log.Infof("%s is already installed", pm.Name)
			return nil
		}
		var installCmd types.Command
		switch pm.Name {
		// TODO: No Curl Bash, create installer
		case "npm":
			installCmd = types.Command{
				Exec: osInfo.Tools.Bash.Bin,
				Args: []string{
					"-c",
					"curl -fsSL https://install.npmjs.com | bash",
				},
			}
		// TODO: No Curl Bash, create installer
		case "pnpm":
			installCmd = types.Command{
				Exec: osInfo.Tools.Bash.Bin,
				Args: []string{
					"-c",
					"curl -fsSL https://get.pnpm.io/install.sh | bash",
				},
			}
		// TODO: No Curl Bash, create installer
		case "yarn":
			installCmd = types.Command{
				Exec: osInfo.Tools.Bash.Bin,
				Args: []string{
					"-c",
					"curl -fsSL https://yarnpkg.com/install.sh | bash",
				},
			}
		}
		if err := helpers.RunCommand(installCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error installing %s: %v", pm.Name, err)
		}
		log.Infof("%s installed successfully", pm.Name)
	} else if pm.Action == "remove" {
		var removeCmd types.Command
		switch pm.Name {
		case "npm":
			removeCmd = types.Command{
				Exec: osInfo.Tools.Bash.Bin,
				Args: []string{
					"-c",
					"npm uninstall -g npm",
				},
			}
		case "pnpm":
			removeCmd = types.Command{
				Exec: osInfo.Tools.Bash.Bin,
				Args: []string{
					"-c",
					"pnpm self-uninstall",
				},
			}
		case "yarn":
			removeCmd = types.Command{
				Exec: osInfo.Tools.Bash.Bin,
				Args: []string{
					"-c",
					"yarn global remove yarn",
				},
			}
		}
		if err := helpers.RunCommand(removeCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error removing %s: %v", pm.Name, err)
		}
		log.Infof("%s removed successfully", pm.Name)
	}
	return nil
}

func processPip(pm types.PackageManagerInfo, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	if pm.Action == "install" {
		if helpers.FindTool("pip").Exists {
			log.Infof("pip is already installed")
			return nil
		}
		installCmd := types.Command{
			Exec: osInfo.Tools.Bash.Bin,
			Args: []string{
				"-c",
				"curl https://bootstrap.pypa.io/get-pip.py | python",
			},
		}
		if err := helpers.RunCommand(installCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error installing pip: %v", err)
		}
		log.Infof("pip installed successfully")
	} else if pm.Action == "remove" {
		removeCmd := types.Command{
			Exec: osInfo.Tools.Bash.Bin,
			Args: []string{
				"-c",
				"python -m pip uninstall pip -y",
			},
		}
		if err := helpers.RunCommand(removeCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error removing pip: %v", err)
		}
		log.Infof("pip removed successfully")
	}
	return nil
}

func processGem(pm types.PackageManagerInfo, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	if pm.Action == "install" {
		if helpers.FindTool("gem").Exists {
			log.Infof("RubyGems is already installed")
			return nil
		}
		updateCmd := types.Command{
			Exec: osInfo.Tools.Bash.Bin,
			Args: []string{
				"-c",
				"gem update --system",
			},
		}
		if err := helpers.RunCommand(updateCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error updating Ruby Gems: %v", err)
		}
		log.Infof("RubyGems installed successfully")
	} else if pm.Action == "remove" {
		removeCmd := types.Command{
			Exec: osInfo.Tools.Bash.Bin,
			Args: []string{
				"-c",
				"gem uninstall rubygems-update",
			},
		}
		if err := helpers.RunCommand(removeCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error removing Ruby Gems: %v", err)
		}
		log.Infof("RubyGems removed successfully")
	}
	return nil
}

func processCargo(pm types.PackageManagerInfo, osInfo *types.OSInfo, initConfig *types.InitConfig) error {

	if pm.Action == "install" {
		if helpers.FindTool("cargo").Exists {
			log.Infof("Cargo is already installed")
			return nil
		}

		log.Infof("Installing Cargo")

		// Create a temporary file for the installation script
		tmpFile, err := os.CreateTemp("", "cargo-install-*.sh")
		if err != nil {
			return fmt.Errorf("error creating temporary file: %v", err)
		}
		defer func(name string) {
			err := os.Remove(name)
			if err != nil {
				log.Warnf("Error removing temporary file %s: %v", name, err)
			}
		}(tmpFile.Name())

		// Download the cargo installation script
		downloadCmd := types.Command{
			Exec: osInfo.Tools.Curl.Bin,
			Args: []string{
				"-fsSLf",
				"-o",
				tmpFile.Name(),
				"--proto '=https' --tlsv1.2 https://sh.rustup.rs",
			},
		}
		if err := helpers.RunCommand(downloadCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error downloading cargo installation script: %v", err)
		}

		// Make the installation script executable
		chmodCmd := types.Command{
			Exec: "chmod",
			Args: []string{
				"+x",
				tmpFile.Name(),
			},
		}
		if err := helpers.RunCommand(chmodCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error making cargo installation script executable: %v", err)
		}

		log.Debugf("Running Cargo Install Command")
		// Run the installation script
		installCmd := types.Command{
			Exec: osInfo.Tools.Bash.Bin,
			Args: []string{
				tmpFile.Name(), "-y",
			},
		}
		err = helpers.RunCommand(installCmd, initConfig.Variables.Flags.Debug)
		if err != nil {
			return fmt.Errorf("error installing cargo: %v", err)
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
		err = os.Setenv("PATH", fmt.Sprintf("%s:%s", cargoPath, os.Getenv("PATH")))
		if err != nil {
			return err
		}

		// Install cargo-update
		cargoUpdateCmd := types.Command{
			Exec:   filepath.Join(cargoPath, "cargo"),
			Args:   []string{"install", "cargo-update", "--features", "vendored-openssl"},
			AsUser: pm.AsUser,
		}
		if err := helpers.RunCommand(cargoUpdateCmd, initConfig.Variables.Flags.Debug); err != nil {
			log.Warnf("Error installing cargo-update: %v", err)
			// Continue execution even if cargo-update installation fails
		}

		// Install cargo-cache
		cargoCacheCmd := types.Command{
			Exec:   filepath.Join(cargoPath, "cargo"),
			Args:   []string{"install", "cargo-cache"},
			AsUser: pm.AsUser,
		}
		if err := helpers.RunCommand(cargoCacheCmd, initConfig.Variables.Flags.Debug); err != nil {
			log.Warnf("Error installing cargo-cache: %v", err)
			// Continue execution even if cargo-cache installation fails
		}
		log.Infof("Cargo installed successfully")

		osInfo.PackageManager.Cargo = types.PackageManagerInfo{
			Name:     "cargo",
			Bin:      filepath.Join(cargoPath, "cargo"),
			List:     filepath.Join(cargoPath, "cargo install --list"),
			Search:   filepath.Join(cargoPath, "cargo search"),
			Install:  filepath.Join(cargoPath, "cargo install"),
			Remove:   filepath.Join(cargoPath, "cargo uninstall"),
			Update:   filepath.Join(cargoPath, "cargo install --force"),
			Clean:    filepath.Join(cargoPath, "cargo cache --autoclean"),
			Elevated: false,
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
		cargoUpdateCmd := types.Command{
			Exec:   filepath.Join(cargoPath, "cargo"),
			Args:   []string{"uninstall", "cargo-update"},
			AsUser: pm.AsUser,
		}
		if err := helpers.RunCommand(cargoUpdateCmd, initConfig.Variables.Flags.Debug); err != nil {
			log.Warnf("Error uninstalling cargo-update: %v", err)
			// Continue execution even if cargo-update uninstallation fails
		}

		// Uninstall Rust and Cargo
		uninstallCmd := types.Command{
			Exec: osInfo.Tools.Bash.Bin,
			Args: []string{
				"-c",
				"rustup self uninstall -y",
			},
			AsUser: pm.AsUser,
		}
		if err := helpers.RunCommand(uninstallCmd, initConfig.Variables.Flags.Debug); err != nil {
			return fmt.Errorf("error removing Cargo: %v", err)
		}
		log.Infof("Cargo removed successfully")
	}
	return nil
}

func processGnomeExtensionsCLI(pm types.PackageManagerInfo, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	if pm.Action == "install" {
		if helpers.FindTool("gnome-extensions").Exists || helpers.FindTool("gext").Exists {
			log.Infof("GNOME Extensions CLI is already installed")
			return nil
		}

		log.Infof("Installing GNOME Extensions CLI")
		if osInfo.PackageManager.Pip.Bin != "" {
			installCmd := types.Command{
				Exec:     osInfo.PackageManager.Default.Install,
				Args:     []string{"gnome-extensions-cli"},
				Elevated: osInfo.PackageManager.Default.Elevated,
			}
			err := helpers.RunCommand(installCmd, initConfig.Variables.Flags.Debug)
			if err != nil {
				return fmt.Errorf("error installing GNOME Extensions CLI: %v", err)
			}
		} else {
			log.Warn("pip not installed, cannot install gnome-extensions")
		}

		log.Infof("GNOME Extensions CLI installed successfully")

		if helpers.FindTool("gnome-extensions").Exists {

			osInfo.PackageManager.GnomeExtensions = types.PackageManagerInfo{
				Name:     "gnome-extensions",
				Bin:      "gnome-extensions",
				List:     "gnome-extensions list --user --enabled",
				Search:   "gnome-extensions search",
				Install:  "gnome-extensions install",
				Remove:   "gnome-extensions uninstall",
				Update:   "",
				Clean:    "",
				Elevated: false,
			}
		} else if helpers.FindTool("gext").Exists {
			osInfo.PackageManager.GnomeExtensions = types.PackageManagerInfo{
				Name:     "gext",
				Bin:      "gext",
				List:     "gext list --user --enabled",
				Search:   "gext search",
				Install:  "gext install",
				Remove:   "gext uninstall",
				Update:   "gext update",
				Clean:    "",
				Elevated: false,
			}
		}
	} else if pm.Action == "remove" {
		log.Infof("Removing GNOME Extensions CLI")
		removeCmd := types.Command{
			Exec:     osInfo.PackageManager.Default.Remove,
			Args:     []string{"gnome-extensions-cli"},
			Elevated: osInfo.PackageManager.Default.Elevated,
		}
		err := helpers.RunCommand(removeCmd, initConfig.Variables.Flags.Debug)
		if err != nil {
			return fmt.Errorf("error removing GNOME Extensions CLI: %v", err)
		}
		log.Infof("GNOME Extensions CLI removed successfully")
	}

	return nil
}
