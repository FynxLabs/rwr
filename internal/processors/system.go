package processors

import (
	"github.com/thefynx/rwr/internal/helpers"
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
		helpers.SetLinuxDetails(&osInfo)
	case "darwin":
		log.Debug("macOS detected.")
		osInfo.OS = "macos"
		helpers.SetMacOSDetails(&osInfo)
	case "windows":
		log.Debug("Windows detected.")
		osInfo.OS = "windows"
		helpers.SetWindowsDetails(&osInfo)
	default:
		log.Fatal("This setup only supports macOS, Linux, and Windows.")
	}

	osInfo.Tools = findCommonTools()

	return osInfo
}

// findCommonTools finds if the listed common tools are installed and returns their information.
func findCommonTools() types.ToolList {
	var tools types.ToolList

	tools.Git = findTool("git")
	tools.Pip = findTool("pip")
	tools.Gem = findTool("gem")
	tools.Npm = findTool("npm")
	tools.Yarn = findTool("yarn")
	tools.Pnpm = findTool("pnpm")
	tools.Bun = findTool("bun")
	tools.Cargo = findTool("cargo")
	tools.Docker = findTool("docker")
	tools.Curl = findTool("curl")
	tools.Wget = findTool("wget")
	tools.Make = findTool("make")
	tools.Clang = findTool("clang")
	tools.Python = findTool("python")
	tools.Ruby = findTool("ruby")
	tools.Java = findTool("java")

	return tools
}

// findTool checks if a tool exists and returns its information.
func findTool(name string) types.ToolInfo {
	log.Debugf("Checking for %s", name)
	path, err := exec.LookPath(name)
	if err != nil {
		log.Debugf("%s not found", name)
		return types.ToolInfo{Exists: false}
	}
	log.Debugf("%s found at %s", name, path)
	return types.ToolInfo{Exists: true, Bin: path}
}
