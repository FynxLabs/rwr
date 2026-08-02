package system

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/types"
)

// FindTool checks if a tool exists and returns its information.
// It prioritizes system paths over third-party package manager paths.
//
// The enhanced path is searched directly rather than written into the process
// environment: the old os.Setenv permanently rewrote PATH for the whole
// process (and every child it spawned) as a side effect of an existence
// check, dozens of times per run.
func FindTool(name string) types.ToolInfo {
	log.Debugf("FindTool: Checking for binary '%s'", name)

	for _, dir := range filepath.SplitList(AddCommonPaths()) {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, name)
		if runtime.GOOS == "windows" && filepath.Ext(candidate) == "" {
			candidate += ".exe"
		}
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
			continue
		}
		log.Debugf("FindTool: %s found at %s", name, candidate)
		return types.ToolInfo{Exists: true, Bin: candidate}
	}

	// Fall back to the real PATH via LookPath, which also understands
	// platform quirks (PATHEXT on windows) the loop above does not.
	if path, err := exec.LookPath(name); err == nil {
		log.Debugf("FindTool: %s found at %s (via PATH)", name, path)
		return types.ToolInfo{Exists: true, Bin: path}
	}

	log.Debugf("FindTool: %s not found in any PATH", name)
	return types.ToolInfo{Exists: false}
}
