package helpers

import (
	"github.com/thefynx/rwr/internal/processors/types"
	"os/exec"
	"runtime"

	"github.com/charmbracelet/log"
)

// DetectOS detects the operating system and package managers, returns an OSInfo struct.
// Can be used to make decisions based on the user's system.
func DetectOS() types.OSInfo {
	log.Debug("Detecting operating system.")
	var osInfo types.OSInfo

	switch runtime.GOOS {
	case "linux":
		log.Debug("Linux detected.")
		osInfo.OS = "linux"
		err := SetLinuxDetails(&osInfo)
		if err != nil {
			log.Fatalf("Error setting Linux details: %v", err)
		}
	case "darwin":
		log.Debug("macOS detected.")
		osInfo.OS = "macos"
		SetMacOSDetails(&osInfo)
	case "windows":
		log.Debug("Windows detected.")
		osInfo.OS = "windows"
		SetWindowsDetails(&osInfo)
	default:
		log.Fatal("This setup only supports macOS, Linux, and Windows.")
	}

	osInfo.Tools = findCommonTools()

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

	return tools
}

// FindTool checks if a tool exists and returns its information.
func FindTool(name string) types.ToolInfo {
	log.Debugf("Checking for %s", name)
	path, err := exec.LookPath(name)
	if err != nil {
		log.Debugf("%s not found", name)
		return types.ToolInfo{Exists: false}
	}
	log.Debugf("%s found at %s", name, path)
	return types.ToolInfo{Exists: true, Bin: path}
}
