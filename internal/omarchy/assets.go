package omarchy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fynxlabs/rwr/internal/types"
)

type ownership struct {
	Source string `json:"source"`
	Hash   string `json:"hash"`
}

func (c *Client) ownership() (map[string]ownership, error) {
	raw, err := c.read(".local/state/rwr/omarchy/ownership.json")
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]ownership{}, nil
	}
	if err != nil {
		return nil, err
	}
	var state map[string]ownership
	err = json.Unmarshal(raw, &state)
	if state == nil && err == nil {
		err = fmt.Errorf("invalid Omarchy ownership record")
	}
	return state, err
}
func (c *Client) remember(id, source, sum string) error {
	state, err := c.ownership()
	if err != nil {
		return err
	}
	state[id] = ownership{source, sum}
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = c.write(".local/state/rwr/omarchy/ownership.json", raw, 0600)
	return err
}
func treeFiles(path string) (map[string][]byte, map[string]fs.FileMode, error) {
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, nil, err
	}
	defer closeRoot(root)
	files := map[string][]byte{}
	modes := map[string]fs.FileMode{}
	err = fs.WalkDir(root.FS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("asset symlink %s is unsupported; supply regular files", p)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported asset type")
		}
		raw, err := root.ReadFile(p)
		if err != nil {
			return err
		}
		files[p] = raw
		modes[p] = info.Mode().Perm()
		return nil
	})
	return files, modes, err
}
func treeHash(path string) (string, error) {
	files, modes, err := treeFiles(path)
	if err != nil {
		return "", err
	}
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte(0)
		b.WriteString(hash(files[k]))
		fmt.Fprint(&b, modes[k])
		b.WriteByte(0)
	}
	return hash([]byte(b.String())), nil
}
func (c *Client) stage(ctx context.Context, source *types.OmarchySource) (string, func(), error) {
	base := ".local/state/rwr/omarchy/staging/" + fmt.Sprint(time.Now().UnixNano())
	root, err := os.OpenRoot(c.Home)
	if err != nil {
		return "", nil, err
	}
	defer closeRoot(root)
	cleanup := func() {
		r, err := os.OpenRoot(c.Home)
		if err == nil {
			defer closeRoot(r)
			if err := r.RemoveAll(base); err != nil {
				return
			}
		}
	}
	if err := root.MkdirAll(filepath.Dir(base), 0700); err != nil {
		return "", nil, err
	}
	dest := filepath.Join(c.Home, base)
	if source.Git != "" {
		if err := c.Run(types.Command{Exec: "git", Args: []string{"-c", "core.hooksPath=/dev/null", "clone", "--", source.Git, dest}, Variables: map[string]string{"GIT_TERMINAL_PROMPT": "0", "GIT_SSH_COMMAND": "ssh -oBatchMode=yes"}}); err != nil {
			cleanup()
			return "", nil, err
		}
		if source.Ref != "" {
			if err := c.command("git", "-C", dest, "-c", "core.hooksPath=/dev/null", "checkout", "--detach", source.Ref, "--"); err != nil {
				cleanup()
				return "", nil, err
			}
		}
	} else {
		files, modes, err := treeFiles(source.Path)
		if err != nil {
			cleanup()
			return "", nil, err
		}
		if err := root.MkdirAll(base, 0700); err != nil {
			cleanup()
			return "", nil, err
		}
		for p, raw := range files {
			target := filepath.Join(base, p)
			if err := root.MkdirAll(filepath.Dir(target), 0700); err != nil {
				cleanup()
				return "", nil, err
			}
			if err := root.WriteFile(target, raw, modes[p]); err != nil {
				cleanup()
				return "", nil, err
			}
		}
	}
	return dest, cleanup, nil
}
func manifestAt(path string) (map[string]any, error) {
	raw, err := readExternal(filepath.Join(path, "manifest.json"))
	if err != nil {
		return nil, err
	}
	var m map[string]any
	err = json.Unmarshal(raw, &m)
	return m, err
}
func (c *Client) sourceMatches(ctx context.Context, path string, s *types.OmarchySource) error {
	if s.Git != "" {
		raw, err := c.Read(ctx, "git", "-C", path, "remote", "get-url", "origin")
		if err != nil || strings.TrimSuffix(trim(raw), ".git") != strings.TrimSuffix(s.Git, ".git") {
			return fmt.Errorf("installed source differs from declared repository")
		}
	}
	if s.Path != "" {
		a, err := treeHash(path)
		if err != nil {
			return err
		}
		b, err := treeHash(s.Path)
		if err != nil {
			return err
		}
		if a != b {
			return fmt.Errorf("local assets differ; only unchanged owned assets may be replaced")
		}
	}
	return nil
}
func (c *Client) cleanOwned(ctx context.Context, id, path string) error {
	state, err := c.ownership()
	if err != nil {
		return err
	}
	record, ok := state[id]
	if !ok {
		return fmt.Errorf("%s predates RWR or was adopted; refusing removal/replacement", id)
	}
	sum, err := treeHash(path)
	if err != nil {
		return err
	}
	if sum != record.Hash {
		return fmt.Errorf("%s has local modifications", id)
	}
	if record.Source != "" && strings.HasPrefix(record.Source, "git:") {
		remote, err := c.Read(ctx, "git", "-C", path, "remote", "get-url", "origin")
		if err != nil || trim(remote) != strings.TrimPrefix(record.Source, "git:") {
			return fmt.Errorf("%s repository origin changed", id)
		}
		raw, err := c.Read(ctx, "git", "-C", path, "status", "--porcelain")
		if err != nil || trim(raw) != "" {
			return fmt.Errorf("%s checkout is dirty or unavailable", id)
		}
	}
	return nil
}
func (c *Client) plugin(ctx context.Context, o Operation, snap *Snapshot) (bool, error) {
	p := o.Plugin
	m, present := snap.Plugins[p.ID]
	dest := ".config/omarchy/plugins/" + p.ID
	absolute := filepath.Join(c.Home, dest)
	if p.State == "absent" {
		if !present {
			return false, nil
		}
		if m.FirstParty {
			return false, fmt.Errorf("packaged plugin cannot be removed")
		}
		if err := c.cleanOwned(ctx, o.ID, absolute); err != nil {
			return false, err
		}
		return true, c.command("omarchy", "plugin", "remove", p.ID, "--yes")
	}
	if present && p.Source == nil {
		return false, nil
	}
	if !present && p.Source == nil {
		return false, fmt.Errorf("plugin %s missing; provide its source", p.ID)
	}
	s := p.Source
	if s.Clone != "" {
		expected := c.cloneID(s.Clone)
		if p.ID != expected {
			return false, fmt.Errorf("native clone on this user requires ID %s", expected)
		}
		if present {
			if m.Clone != s.Clone {
				return false, fmt.Errorf("clone source mismatch")
			}
			return false, nil
		}
		if err := c.Run(types.Command{Exec: "omarchy", Args: []string{"plugin", "clone", s.Clone}, Variables: map[string]string{"USER": strings.TrimSuffix(c.cloneID(s.Clone), "."+strings.TrimPrefix(s.Clone, "omarchy."))}}); err != nil {
			return true, err
		}
		sum, err := treeHash(absolute)
		if err != nil {
			return true, err
		}
		return true, c.remember(o.ID, "clone:"+s.Clone, sum)
	}
	if present {
		matchErr := c.sourceMatches(ctx, absolute, s)
		if matchErr != nil && s.Git != "" {
			return false, matchErr
		}
		if s.Git != "" && s.Ref != "" {
			head, err := c.Read(ctx, "git", "-C", absolute, "rev-parse", "HEAD")
			if err != nil {
				return false, err
			}
			want, err := c.Read(ctx, "git", "-C", absolute, "rev-parse", "--verify", s.Ref+"^{commit}")
			if err == nil && trim(head) == trim(want) {
				return false, nil
			}
		} else if matchErr == nil && !p.Update {
			return false, nil
		}
		if err := c.cleanOwned(ctx, o.ID, absolute); err != nil {
			return false, err
		}
	}
	stage, cleanup, err := c.preparedSource(ctx, o.ID, s)
	if err != nil {
		return false, err
	}
	defer cleanup()
	manifest, err := manifestAt(stage)
	if err != nil {
		return false, err
	}
	if manifest["id"] != p.ID {
		return false, fmt.Errorf("staged plugin manifest ID does not match %s", p.ID)
	}
	if _, err := c.Read(ctx, "omarchy", "plugin", "validate", stage); err != nil {
		return false, fmt.Errorf("staged plugin failed native manifest validation")
	}
	r, err := os.OpenRoot(c.Home)
	if err != nil {
		return false, err
	}
	defer closeRoot(r)
	if err := r.MkdirAll(filepath.Dir(dest), 0700); err != nil {
		return false, err
	}
	prior := filepath.Join(filepath.Dir(dest), ".rwr-prior-"+p.ID)
	if present {
		if _, err := r.Stat(prior); err == nil {
			return false, fmt.Errorf("previous plugin recovery directory exists")
		}
		if err := r.Rename(dest, prior); err != nil {
			return false, err
		}
	}
	relative, err := filepath.Rel(c.Home, stage)
	if err != nil {
		return false, err
	}
	if err := r.Rename(relative, dest); err != nil {
		if present {
			if restoreErr := r.Rename(prior, dest); restoreErr != nil {
				return false, errors.Join(err, restoreErr)
			}
		}
		return false, err
	}
	if err := c.command("omarchy-shell", "shell", "rescanPlugins"); err != nil {
		return true, err
	}
	sum, err := treeHash(absolute)
	if err != nil {
		return true, err
	}
	origin := "path:" + s.Path
	if s.Git != "" {
		origin = "git:" + s.Git
	}
	if err := c.remember(o.ID, origin, sum); err != nil {
		return true, err
	}
	if present {
		if err := r.RemoveAll(prior); err != nil {
			return true, err
		}
	}
	return true, nil
}
func (c *Client) themeSource(ctx context.Context, o Operation) (bool, error) {
	t := o.Theme
	s := t.Source
	dest := ".config/omarchy/themes/" + t.Name
	absolute := filepath.Join(c.Home, dest)
	if _, err := os.Stat(absolute); err == nil {
		if c.prepared[o.ID] == "" {
			if err := c.sourceMatches(ctx, absolute, s); err == nil {
				return false, nil
			}
		}
		if err := c.cleanOwned(ctx, o.ID, absolute); err != nil {
			return false, err
		}
	}
	stage, cleanup, err := c.preparedSource(ctx, o.ID, s)
	if err != nil {
		return false, err
	}
	defer cleanup()
	// Keep .git for native theme-set's executable-content filtering.
	if _, err := readExternal(filepath.Join(stage, "colors.toml")); err != nil {
		if _, stockErr := readExternal(filepath.Join(c.Distribution, "themes", t.Name, "colors.toml")); stockErr != nil {
			return false, fmt.Errorf("theme requires colors.toml or a matching stock overlay")
		}
	}
	root, err := os.OpenRoot(c.Home)
	if err != nil {
		return false, err
	}
	defer closeRoot(root)
	if err := root.MkdirAll(filepath.Dir(dest), 0700); err != nil {
		return false, err
	}
	prior := filepath.Join(filepath.Dir(dest), ".rwr-prior-"+t.Name)
	hadPrior := false
	if _, err := root.Stat(dest); err == nil {
		if _, err := root.Stat(prior); err == nil {
			return false, fmt.Errorf("previous theme recovery directory exists")
		}
		if err := root.Rename(dest, prior); err != nil {
			return false, err
		}
		hadPrior = true
	}
	rel, err := filepath.Rel(c.Home, stage)
	if err != nil {
		return false, err
	}
	if err := root.Rename(rel, dest); err != nil {
		if hadPrior {
			if restoreErr := root.Rename(prior, dest); restoreErr != nil {
				return false, errors.Join(err, restoreErr)
			}
		}
		return false, err
	}
	if hadPrior {
		if err := root.RemoveAll(prior); err != nil {
			return true, err
		}
	}
	sum, err := treeHash(absolute)
	if err != nil {
		return true, err
	}
	source := "path:" + s.Path
	if s.Git != "" {
		source = "git:" + s.Git
	}
	return true, c.remember(o.ID, source, sum)
}

// Prepare every changed source before any installed resource is changed. A bad
// manifest in the last declaration cannot leave the first plugin installed.
func (c *Client) prepareSources(ctx context.Context, ops []Operation, snap *Snapshot) (func(), error) {
	c.prepared = map[string]string{}
	var cleanups []func()
	cleanup := func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
		c.prepared = nil
	}
	for _, o := range ops {
		var source *types.OmarchySource
		var dest string
		if o.Kind == "plugin" {
			source = o.Plugin.Source
			if source == nil {
				continue
			}
			if source.Clone != "" {
				expected := c.cloneID(source.Clone)
				if o.Plugin.ID != expected {
					cleanup()
					return nil, fmt.Errorf("native clone requires ID %s", expected)
				}
				if m, ok := snap.Plugins[source.Clone]; !ok || !m.FirstParty {
					cleanup()
					return nil, fmt.Errorf("clone source is not a discovered built-in")
				}
				continue
			}
			if p, ok := snap.Plugins[o.Plugin.ID]; ok {
				dest = p.SourceDir
			}
		} else if o.Kind == "theme-source" {
			source = o.Theme.Source
			dest = filepath.Join(c.Home, ".config/omarchy/themes", o.Theme.Name)
			if _, err := os.Stat(dest); errors.Is(err, fs.ErrNotExist) {
				dest = ""
			}
		} else {
			continue
		}
		if dest != "" {
			matches := c.sourceMatches(ctx, dest, source) == nil
			if source.Git != "" && !matches {
				cleanup()
				return nil, fmt.Errorf("%s source mismatch", o.ID)
			}
			pinned := false
			if source.Ref != "" {
				a, err := c.Read(ctx, "git", "-C", dest, "rev-parse", "HEAD")
				if err != nil {
					cleanup()
					return nil, err
				}
				b, err := c.Read(ctx, "git", "-C", dest, "rev-parse", "--verify", source.Ref+"^{commit}")
				pinned = err == nil && trim(a) == trim(b)
			}
			update := o.Plugin != nil && o.Plugin.Update
			if matches && !update && (source.Ref == "" || pinned) {
				continue
			}
			if err := c.cleanOwned(ctx, o.ID, dest); err != nil {
				cleanup()
				return nil, err
			}
		}
		stage, remove, err := c.stage(ctx, source)
		if err != nil {
			cleanup()
			return nil, err
		}
		cleanups = append(cleanups, remove)
		if o.Kind == "plugin" {
			m, err := manifestAt(stage)
			if err != nil || m["id"] != o.Plugin.ID {
				cleanup()
				return nil, fmt.Errorf("%s staged manifest identity mismatch", o.ID)
			}
			if _, err := c.Read(ctx, "omarchy", "plugin", "validate", stage); err != nil {
				cleanup()
				return nil, fmt.Errorf("%s manifest validation failed", o.ID)
			}
		} else {
			if _, err := readExternal(filepath.Join(stage, "colors.toml")); err != nil {
				if _, err := readExternal(filepath.Join(c.Distribution, "themes", o.Theme.Name, "colors.toml")); err != nil {
					cleanup()
					return nil, fmt.Errorf("%s needs colors.toml or a stock overlay", o.ID)
				}
			}
		}
		c.prepared[o.ID] = stage
	}
	return cleanup, nil
}
func (c *Client) preparedSource(ctx context.Context, id string, source *types.OmarchySource) (string, func(), error) {
	if path, ok := c.prepared[id]; ok {
		return path, func() {}, nil
	}
	return c.stage(ctx, source)
}

func (c *Client) cloneID(source string) string {
	username := c.User
	if username == "" {
		username = filepath.Base(c.Home)
	}
	return username + "." + strings.TrimPrefix(source, "omarchy.")
}
