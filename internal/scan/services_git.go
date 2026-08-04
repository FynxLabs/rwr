package scan

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Services reports enabled unit names on the current platform; elsewhere it
// reports nothing — an empty answer is honest, a guessed one is not.
func Services() []string {
	if runtime.GOOS != "linux" {
		return nil
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		return nil
	}
	out, err := exec.Command("systemctl", "list-unit-files", "--state=enabled", "--type=service", "--no-legend", "--plain").Output()
	if err != nil {
		return nil
	}
	return parseEnabledUnits(string(out))
}

// parseEnabledUnits keeps the units the operator enabled: a unit whose
// vendor preset column already says enabled would be on without them.
func parseEnabledUnits(out string) []string {
	var services []string
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) >= 3 && fields[2] == "enabled" {
			continue
		}
		services = append(services, strings.TrimSuffix(fields[0], ".service"))
	}
	sort.Strings(services)
	return services
}

// GitCheckout is one discovered clone.
type GitCheckout struct {
	Path string
	URL  string
}

// GitCheckouts finds clones under the given roots (two levels deep — the
// host/org/repo layout most people keep).
func GitCheckouts(roots []string) []GitCheckout {
	var checkouts []GitCheckout
	for _, root := range roots {
		for _, depth := range []string{"*", "*/*"} {
			matches, err := filepath.Glob(filepath.Join(root, depth, ".git"))
			if err != nil {
				continue
			}
			for _, match := range matches {
				dir := filepath.Dir(match)
				if _, err := os.Stat(match); err != nil {
					continue
				}
				url := ""
				if out, err := exec.Command("git", "-C", dir, "remote", "get-url", "origin").Output(); err == nil { // #nosec G204 -- dir comes from the operator's own filesystem glob; read-only query
					url = strings.TrimSpace(string(out))
				}
				checkouts = append(checkouts, GitCheckout{Path: dir, URL: url})
			}
		}
	}
	sort.Slice(checkouts, func(i, j int) bool { return checkouts[i].Path < checkouts[j].Path })
	return checkouts
}
