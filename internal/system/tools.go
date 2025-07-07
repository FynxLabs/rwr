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
	log.Debugf("FindTool: Checking for binary '%s'", name)

	// Get current PATH before enhancement
	originalPath := os.Getenv("PATH")
	log.Debugf("FindTool: Original PATH: %s", originalPath)

	// Use the enhanced PATH from AddCommonPaths
	enhancedPath := AddCommonPaths()
	log.Debugf("FindTool: Enhanced PATH: %s", enhancedPath)

	if err := os.Setenv("PATH", enhancedPath); err != nil {
		log.Errorf("FindTool: Error setting enhanced PATH: %v", err)
		// Try with original PATH as fallback
		if path, err := exec.LookPath(name); err == nil {
			log.Debugf("FindTool: %s found at %s (using original PATH)", name, path)
			return types.ToolInfo{Exists: true, Bin: path}
		}
	} else {
		if path, err := exec.LookPath(name); err == nil {
			log.Debugf("FindTool: %s found at %s (using enhanced PATH)", name, path)
			return types.ToolInfo{Exists: true, Bin: path}
		} else {
			log.Debugf("FindTool: exec.LookPath failed for %s: %v", name, err)
		}
	}

	log.Debugf("FindTool: %s not found in any PATH", name)
	return types.ToolInfo{Exists: false}
}
