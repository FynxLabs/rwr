package helpers

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fynxlabs/rwr/internal/types"

	"github.com/charmbracelet/log"
)

// DetectOS detects the operating system and package managers, returns an OSInfo struct.
// Can be used to make decisions based on the user's system.
func DetectOS() *types.OSInfo {
	log.Debug("Detecting operating system.")
	osInfo := &types.OSInfo{} // Create a new instance of types.OSInfo
	err := SetPaths()
	if err != nil {
		log.Fatalf("Error setting PATH: %v", err)
	}

	osInfo.Tools = findCommonTools()

	osInfo.OS = runtime.GOOS
	osInfo.System = types.System{
		OS:        runtime.GOOS,
		OSFamily:  getOSFamily(),
		OSVersion: getOSVersion(),
		OSArch:    runtime.GOARCH,
	}

	switch runtime.GOOS {
	case "linux":
		log.Debug("Linux detected.")
		err := SetLinuxDetails(osInfo)
		if err != nil {
			log.Fatalf("Error setting Linux details: %v", err)
		}
	case "darwin":
		log.Debug("macOS detected.")
		SetMacOSDetails(osInfo)
	case "windows":
		log.Debug("Windows detected.")
		SetWindowsDetails(osInfo)
	default:
		log.Fatal("This setup only supports macOS, Linux, and Windows.")
	}

	return osInfo
}

// findCommonTools finds if the listed common tools are installed and returns their information.
func findCommonTools() types.ToolList {
	var tools types.ToolList

	tools.Git = FindTool("git")
	tools.Go = FindTool("go")
	tools.Rust = FindTool("rust")
	tools.Bun = FindTool("bun")
	tools.Docker = FindTool("docker")
	tools.Curl = FindTool("curl")
	tools.Wget = FindTool("wget")
	tools.Make = FindTool("make")
	tools.Clang = FindTool("clang")
	tools.Python = FindTool("python")
	tools.Ruby = FindTool("ruby")
	tools.Java = FindTool("java")
	tools.Bash = FindTool("bash")
	tools.Zsh = FindTool("zsh")
	tools.PowerShell = FindTool("powershell")
	tools.Perl = FindTool("perl")
	tools.Lua = FindTool("lua")
	tools.Gpg = FindTool("gpg")
	tools.Rpm = FindTool("rpm")
	tools.Dpkg = FindTool("dpkg")
	tools.Cat = FindTool("cat")
	tools.Ls = FindTool("ls")
	tools.Lsof = FindTool("lsof")

	return tools
}

// FindTool checks if a tool exists and returns its information.
func FindTool(name string) types.ToolInfo {
	log.Debugf("Checking for %s", name)

	// Add common paths to the PATH environment variable
	updatedPath := AddCommonPaths()

	// Save the original PATH environment variable
	originalPath := os.Getenv("PATH")
	defer func(key, value string) {
		err := os.Setenv(key, value)
		if err != nil {
			log.Warnf("Error setting %s: %v", key, err)
		}
	}("PATH", originalPath)

	// Set the updated PATH environment variable
	err := os.Setenv("PATH", updatedPath)
	if err != nil {
		log.Warnf("Error setting PATH: %v", err)
	}

	path, err := exec.LookPath(name)
	if err != nil {
		log.Debugf("%s not found", name)
		return types.ToolInfo{Exists: false}
	}
	log.Debugf("%s found at %s", name, path)
	return types.ToolInfo{Exists: true, Bin: path}
}

// AddCommonPaths checks for common paths and appends them to the existing PATH environment variable.
func AddCommonPaths() string {
	var paths []string
	existingPath := os.Getenv("PATH")
	if existingPath != "" {
		paths = append(paths, existingPath)
	}

	var commonPaths []string

	switch runtime.GOOS {
	case "windows":
		commonPaths = []string{
			"%USERPROFILE%\\AppData\\Local\\Microsoft\\WindowsApps", // Path for Windows Store apps
			"%USERPROFILE%\\scoop\\shims",                           // Path for Scoop package manager
			"%PROGRAMFILES%\\Git\\bin",                              // Path for Git
			"%PROGRAMFILES%\\Go\\bin",                               // Path for Go
			"%PROGRAMFILES%\\nodejs",                                // Path for Node.js
			"%PROGRAMFILES%\\Rust\\.cargo\\bin",                     // Path for Cargo (Rust package manager)
		}
	default: // Unix-like systems (macOS, Linux)
		currentUser, err := user.Current()
		if err != nil {
			log.Warnf("Error getting current user: %v", err)
		} else {
			homeDir := currentUser.HomeDir
			commonPaths = []string{
				"/usr/local/bin",                     // Common system path
				"/usr/local/sbin",                    // Common system path
				filepath.Join(homeDir, ".brew/bin"),  // Path for Homebrew
				filepath.Join(homeDir, ".cargo/bin"), // Path for Cargo
				"/nix/var/nix/profiles/default/bin",  // Common path for Nix
				"/usr/bin",                           // Common system path
				"/usr/sbin",                          // Common system path
				"/bin",                               // Common system path
				"/sbin",                              // Common system path
				"/usr/local/go/bin",                  // Common path for Go
				"/usr/local/cargo/bin",               // Common path for Cargo (Rust package manager)
				"/home/linuxbrew/.linuxbrew/bin",     // Common path for Linuxbrew (Homebrew on Linux)
				"/home/linuxbrew/.linuxbrew/sbin",    // Common path for Linuxbrew (Homebrew on Linux)
				"/snap/bin",                          // Common path for Snap packages
				"/var/lib/flatpak/exports/bin",       // Common path for Flatpak
			}
		}
	}

	for _, p := range commonPaths {
		path, err := filepath.EvalSymlinks(p)
		if err != nil {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			paths = append(paths, path)
		} else if os.IsNotExist(err) {
			log.Debugf("Path %s does not exist", path)
		} else {
			continue
		}
	}

	return strings.Join(paths, string(os.PathListSeparator))
}

// SetPaths sets the PATH environment variable with the common paths appended.
func SetPaths() error {
	newPath := AddCommonPaths()
	switch runtime.GOOS {
	case "windows":
		return os.Setenv("Path", newPath)
	default:
		return os.Setenv("PATH", newPath)
	}
}

func getOSFamily() string {
	switch runtime.GOOS {
	case "linux":
		return getLinuxDistro()
	case "darwin":
		return "Darwin"
	case "windows":
		return "Windows"
	default:
		return "Unknown"
	}
}

func getOSVersion() string {
	switch runtime.GOOS {
	case "linux":
		return getLinuxVersion()
	case "darwin":
		return getDarwinVersion()
	case "windows":
		return getWindowsVersion()
	default:
		return "Unknown"
	}
}

func getLinuxDistro() string {
	if fileExists("/etc/os-release") {
		content, err := os.ReadFile("/etc/os-release")
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "ID=") {
					return strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
				}
			}
		}
	}

	if fileExists("/etc/lsb-release") {
		content, err := os.ReadFile("/etc/lsb-release")
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "DISTRIB_ID=") {
					return strings.Trim(strings.TrimPrefix(line, "DISTRIB_ID="), "\"")
				}
			}
		}
	}

	return "Unknown Linux"
}

func getLinuxVersion() string {
	if fileExists("/etc/os-release") {
		content, err := os.ReadFile("/etc/os-release")
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "VERSION_ID=") {
					return strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
				}
			}
		}
	}

	if fileExists("/etc/lsb-release") {
		content, err := os.ReadFile("/etc/lsb-release")
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "DISTRIB_RELEASE=") {
					return strings.Trim(strings.TrimPrefix(line, "DISTRIB_RELEASE="), "\"")
				}
			}
		}
	}

	return "Unknown Version"
}
