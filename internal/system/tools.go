package system

import (
	"os"
	"os/exec"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/types"
)

// FindTool checks if a tool exists and returns its information.
// It prioritizes system paths over third-party package manager paths.
func FindTool(name string) types.ToolInfo {
	log.Debugf("Checking for %s", name)

	// Use the enhanced PATH from AddCommonPaths
	enhancedPath := AddCommonPaths()
	if err := os.Setenv("PATH", enhancedPath); err == nil {
		if path, err := exec.LookPath(name); err == nil {
			log.Debugf("%s found at %s", name, path)
			return types.ToolInfo{Exists: true, Bin: path}
		}
	}

	log.Debugf("%s not found in any path", name)
	return types.ToolInfo{Exists: false}
}
