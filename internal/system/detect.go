package system

import (
	"runtime"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/types"
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

	osInfo.System = types.System{
		OS:        strings.ToLower(runtime.GOOS),
		OSFamily:  strings.ToLower(getOSFamily()),
		OSVersion: strings.ToLower(getOSVersion()),
		OSArch:    strings.ToLower(runtime.GOARCH),
	}

	switch runtime.GOOS {
	case "linux":
		log.Debug("Linux detected.")
		if err := SetLinuxDetails(osInfo); err != nil {
			log.Errorf("Error setting Linux details: %v", err)
			return nil
		}
	case "darwin":
		log.Debug("macOS detected.")
		if err := SetMacOSDetails(osInfo); err != nil {
			log.Errorf("Error setting macOS details: %v", err)
			return nil
		}
	case "windows":
		log.Debug("Windows detected.")
		if err := SetWindowsDetails(osInfo); err != nil {
			log.Errorf("Error setting Windows details: %v", err)
			return nil
		}
	default:
		log.Fatal("This setup only supports macOS, Linux, and Windows.")
	}

	log.Debug("Returning osInfo")
	log.Debugf("osInfo System: %s", osInfo.System)
	log.Debugf("osInfo Package Mangers: %v", osInfo.PackageManager)
	log.Debugf("osInfo Tools: %v", osInfo.Tools)

	return osInfo
}

// findCommonTools finds if the listed common tools are installed and returns their information.
func findCommonTools() types.ToolList {
	var tools types.ToolList

	tools.Git = FindTool("git")
	tools.Go = FindTool("go")
	// Check for Rust using cargo (package manager) first, then rustc (compiler)
	if cargo := FindTool("cargo"); cargo.Exists {
		tools.Rust = cargo
	} else {
		tools.Rust = FindTool("rustc")
	}
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
