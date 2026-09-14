package omarchy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/fynxlabs/rwr/internal/types"
)

const routePath = ".config/rwr/omarchy/bin/omarchy-launch-screensaver"
const routeStart = "# BEGIN RWR OMARCHY SCREENSAVER"
const routeEnd = "# END RWR OMARCHY SCREENSAVER"

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
func (c *Client) profile() (string, []byte, error) {
	for _, name := range []string{".bash_profile", ".bash_login", ".profile"} {
		raw, err := c.read(name)
		if err == nil {
			return name, raw, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", nil, err
		}
	}
	return ".bash_profile", nil, nil
}
func removeRouteBlock(raw []byte) ([]byte, error) {
	s := string(raw)
	start := strings.Index(s, routeStart)
	end := strings.Index(s, routeEnd)
	if start < 0 && end < 0 {
		return raw, nil
	}
	if start < 0 || end < start || strings.Count(s, routeStart) != 1 || strings.Count(s, routeEnd) != 1 {
		return nil, fmt.Errorf("malformed RWR launch route block")
	}
	end += len(routeEnd)
	if end < len(s) && s[end] == '\n' {
		end++
	}
	return []byte(s[:start] + s[end:]), nil
}
func (c *Client) routeContent(s *types.OmarchyScreensaver) []byte {
	stock := filepath.Join(c.Distribution, "bin/omarchy-launch-screensaver")
	var b strings.Builder
	b.WriteString("#!/bin/bash\n# Managed by RWR: launch routing only.\n")
	b.WriteString("if [[ ${RWR_OMARCHY_STOCK_SCREENSAVER:-0} == 1 ]]; then exec " + shellQuote(stock) + " \"$@\"; fi\n")
	b.WriteString("exec " + shellQuote(s.Launch.Exec))
	for _, a := range s.Launch.Args {
		b.WriteByte(' ')
		b.WriteString(shellQuote(a))
	}
	b.WriteString(" \"$@\"\n")
	return []byte(b.String())
}
func (c *Client) routeBlock() []byte {
	dir := filepath.Join(c.Home, filepath.Dir(routePath))
	return []byte(routeStart + "\nexport PATH=" + shellQuote(dir) + ":\"$PATH\"\n" + routeEnd + "\n")
}
func (c *Client) screensaver(ctx context.Context, s *types.OmarchyScreensaver) (bool, error) {
	name, oldProfile, err := c.profile()
	if err != nil {
		return false, err
	}
	oldRoute, err := c.read(routePath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	state, err := c.ownership()
	if err != nil {
		return false, err
	}
	record, owned := state["integration/screensaver"]
	if len(oldRoute) > 0 && (!owned || record.Hash != hash(oldRoute)) {
		return false, fmt.Errorf("screensaver route is unowned or modified")
	}
	clean, err := removeRouteBlock(oldProfile)
	if err != nil {
		return false, err
	}
	if bytes.Contains(oldProfile, []byte(routeStart)) && !bytes.Contains(oldProfile, c.routeBlock()) {
		return false, fmt.Errorf("screensaver profile route was modified")
	}
	if s.Mode == "stock" {
		if !owned && !bytes.Contains(oldProfile, []byte(routeStart)) {
			return false, nil
		}
		changed, err := c.replace(name, oldProfile, clean, 0600)
		if err != nil {
			return false, err
		}
		// Keep the dormant owned dispatcher for guarded reactivation; the profile no
		// longer selects it, and stock resolution must be checked in a fresh shell.
		resolved, err := c.Read(ctx, "bash", "-lc", "command -v omarchy-launch-screensaver")
		if err != nil || !sameFilePath(trim(resolved), filepath.Join(c.Distribution, "bin/omarchy-launch-screensaver")) {
			_, restoreErr := c.replace(name, clean, oldProfile, 0600)
			return changed, errors.Join(fmt.Errorf("stock launcher route could not be restored"), restoreErr)
		}
		return changed, nil
	}
	for _, cmd := range []*types.OmarchyExec{s.Launch, s.Check} {
		if _, err := c.LookPath(cmd.Exec); err != nil {
			return false, fmt.Errorf("screensaver executable unavailable")
		}
	}
	idle, err := readExternal(filepath.Join(c.Distribution, "shell/plugins/services/idle/Service.qml"))
	if err != nil {
		return false, fmt.Errorf("unsupported Omarchy idle capability")
	}
	for _, required := range []string{"omarchy-launch-screensaver", "org.omarchy.screensaver", "screensaver-dismissed", "screensaverWindowCount"} {
		if !bytes.Contains(idle, []byte(required)) {
			return false, fmt.Errorf("unsupported stock idle lifecycle: missing %s", required)
		}
	}
	if _, err := readExternal(filepath.Join(c.Distribution, "bin/omarchy-launch-screensaver")); err != nil {
		return false, err
	}
	if err := c.Run(types.Command{Exec: s.Check.Exec, Args: s.Check.Args}); err != nil {
		return false, fmt.Errorf("external screensaver readiness check failed: %w", err)
	}
	route := c.routeContent(s)
	profile := append(bytes.TrimRight(clean, "\n"), '\n')
	profile = append(profile, c.routeBlock()...)
	routeChanged, err := c.replace(routePath, oldRoute, route, 0700)
	if err != nil {
		return false, err
	}
	profileChanged, err := c.replace(name, oldProfile, profile, 0600)
	if err != nil {
		return routeChanged, errors.Join(err, c.restoreRoute(route, oldRoute))
	}
	resolved, probeErr := c.Read(ctx, "bash", "-lc", "command -v omarchy-launch-screensaver")
	if probeErr != nil || trim(resolved) != filepath.Join(c.Home, routePath) {
		_, restoreErr := c.replace(name, profile, oldProfile, 0600)
		return true, errors.Join(fmt.Errorf("login shell did not select RWR screensaver route; activation reverted"), restoreErr, c.restoreRoute(route, oldRoute))
	}
	if err := c.remember("integration/screensaver", name, hash(route)); err != nil {
		return true, err
	}
	return routeChanged || profileChanged, nil
}

func sameFilePath(a, b string) bool {
	if a == b {
		return true
	}
	x, err := filepath.EvalSymlinks(a)
	if err != nil {
		return false
	}
	y, err := filepath.EvalSymlinks(b)
	return err == nil && x == y
}
func (c *Client) restoreRoute(current, previous []byte) error {
	if previous != nil {
		_, err := c.replace(routePath, current, previous, 0700)
		return err
	}
	actual, err := c.read(routePath)
	if err != nil {
		return err
	}
	if !bytes.Equal(actual, current) {
		return fmt.Errorf("route changed during recovery")
	}
	root, err := os.OpenRoot(c.Home)
	if err != nil {
		return err
	}
	defer closeRoot(root)
	return root.Remove(routePath)
}
