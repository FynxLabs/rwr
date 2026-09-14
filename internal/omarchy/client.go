package omarchy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

type Client struct {
	hooks          []types.OmarchyHook
	widgets        []Operation
	prepared       map[string]string
	Home           string
	User           string
	Distribution   string
	Read           func(context.Context, string, ...string) ([]byte, error)
	Run            func(types.Command) error
	LookPath       func(string) (string, error)
	RequireSession bool
}
type Manifest struct {
	ID           string   `json:"id"`
	Kinds        []string `json:"kinds"`
	FirstParty   bool     `json:"firstParty"`
	SourceDir    string   `json:"sourceDir"`
	ManifestPath string   `json:"manifestPath"`
	Enabled      bool     `json:"enabled"`
	Clone        string   `json:"-"`
}
type Snapshot struct {
	Plugins map[string]Manifest
	Config  map[string]any
}

func NewClient(debug bool) (*Client, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dist := os.Getenv("OMARCHY_PATH")
	if dist == "" {
		dist = "/usr/share/omarchy"
	}
	account, err := user.Current()
	if err != nil {
		return nil, err
	}
	return &Client{Home: home, User: account.Username, Distribution: dist, RequireSession: true, LookPath: exec.LookPath, Run: func(cmd types.Command) error { return system.RunCommand(cmd, debug) }, Read: func(ctx context.Context, name string, args ...string) ([]byte, error) {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return exec.CommandContext(ctx, name, args...).Output() // #nosec G204 -- discrete argv, read-only capability probes
	}}, nil
}
func (c *Client) command(name string, args ...string) error {
	return c.Run(types.Command{Exec: name, Args: args})
}
func (c *Client) Discover(ctx context.Context) (*Snapshot, error) {
	if c.RequireSession {
		if runtime.GOOS != "linux" || os.Geteuid() == 0 {
			return nil, fmt.Errorf("omarchy requires the logged-in Linux desktop user, without sudo")
		}
		if os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") == "" || os.Getenv("XDG_RUNTIME_DIR") == "" {
			return nil, fmt.Errorf("omarchy requires the active Hyprland session")
		}
		if _, err := c.Read(ctx, "hyprctl", "-j", "monitors"); err != nil {
			return nil, fmt.Errorf("hyprland IPC unavailable")
		}
	}
	for _, name := range []string{"omarchy", "omarchy-shell"} {
		if _, err := c.LookPath(name); err != nil {
			return nil, fmt.Errorf("required Omarchy capability %s unavailable", name)
		}
	}
	raw, err := c.Read(ctx, "omarchy", "plugin", "catalog")
	if err != nil {
		return nil, fmt.Errorf("omarchy plugin catalog unavailable: %w", err)
	}
	var catalog []Manifest
	if err := json.Unmarshal(raw, &catalog); err != nil || catalog == nil {
		return nil, fmt.Errorf("invalid Omarchy catalog response")
	}
	raw, err = c.Read(ctx, "omarchy", "plugin", "list", "--json")
	if err != nil {
		return nil, fmt.Errorf("omarchy shell IPC unavailable: %w", err)
	}
	var live []Manifest
	if err := json.Unmarshal(raw, &live); err != nil || live == nil {
		return nil, fmt.Errorf("invalid Omarchy plugin list response")
	}
	enabled := map[string]bool{}
	for _, p := range live {
		if !validID(p.ID) {
			return nil, fmt.Errorf("invalid runtime plugin identity")
		}
		enabled[p.ID] = p.Enabled
	}
	snap := &Snapshot{Plugins: map[string]Manifest{}}
	for _, p := range catalog {
		if !validID(p.ID) || p.SourceDir == "" || len(p.Kinds) == 0 {
			return nil, fmt.Errorf("invalid catalog manifest")
		}
		if _, ok := snap.Plugins[p.ID]; ok {
			return nil, fmt.Errorf("duplicate catalog identity %s", p.ID)
		}
		if _, ok := enabled[p.ID]; !ok {
			return nil, fmt.Errorf("plugin %s catalog/runtime disagree", p.ID)
		}
		p.Enabled = enabled[p.ID]
		if p.ManifestPath != "" {
			m, err := readExternal(p.ManifestPath)
			if err != nil {
				return nil, err
			}
			var metadata struct {
				Omarchy struct {
					ClonedFrom string `json:"clonedFrom"`
				} `json:"omarchy"`
			}
			if err := json.Unmarshal(m, &metadata); err != nil {
				return nil, err
			}
			p.Clone = metadata.Omarchy.ClonedFrom
		}
		snap.Plugins[p.ID] = p
	}
	snap.Config, _, err = c.config()
	return snap, err
}
func closeRoot(r *os.Root) {
	if err := r.Close(); err != nil {
		log.Warn("Closing Omarchy filesystem", "error", err)
	}
}
func readExternal(path string) ([]byte, error) {
	r, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	defer closeRoot(r)
	return r.ReadFile(filepath.Base(path))
}
func (c *Client) read(path string) ([]byte, error) {
	r, err := os.OpenRoot(c.Home)
	if err != nil {
		return nil, err
	}
	defer closeRoot(r)
	return r.ReadFile(path)
}
func (c *Client) config() (map[string]any, []byte, error) {
	raw, err := c.read(".config/omarchy/shell.json")
	original := raw
	if errors.Is(err, fs.ErrNotExist) {
		raw, err = readExternal(filepath.Join(c.Distribution, "config/omarchy/shell.json"))
		original = nil
	}
	if err != nil {
		return nil, nil, err
	}
	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil || config == nil {
		return nil, nil, fmt.Errorf("invalid shell.json; refusing to reset it")
	}
	for _, k := range []string{"plugins", "disabledPlugins", "cloneSourceRestores"} {
		if v, ok := config[k]; ok {
			if _, ok := v.([]any); !ok {
				return nil, nil, fmt.Errorf("invalid shell.json %s", k)
			}
		}
	}
	if _, ok := config["bar"].(map[string]any); !ok {
		return nil, nil, fmt.Errorf("shell.json requires a bar object")
	}
	return config, original, nil
}
func hash(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

// replace compares against the version observed by the planner before committing.
// GUI writers do not take RWR's lock, so a changed file causes a clean retry.
func (c *Client) replace(path string, old, data []byte, mode fs.FileMode) (bool, error) {
	if bytes.Equal(old, data) {
		return false, nil
	}
	if system.IsDryRun() {
		return false, fmt.Errorf("attempted Omarchy filesystem mutation during dry-run")
	}
	r, err := os.OpenRoot(c.Home)
	if err != nil {
		return false, err
	}
	defer closeRoot(r)
	if info, err := r.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return false, fmt.Errorf("refusing to replace non-regular file %s", path)
		}
		mode = info.Mode().Perm()
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	if err := r.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return false, err
	}
	temporary := path + ".rwr-" + fmt.Sprint(time.Now().UnixNano())
	f, err := r.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return false, err
	}
	defer func() {
		if err := r.Remove(temporary); err != nil && !errors.Is(err, fs.ErrNotExist) {
			log.Warn("Removing temporary Omarchy file", "error", err)
		}
	}()
	_, writeErr := f.Write(data)
	syncErr := f.Sync()
	closeErr := f.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return false, err
	}
	current, err := r.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	if !bytes.Equal(current, old) {
		return false, fmt.Errorf("%s changed concurrently; rerun to rebase declared changes", path)
	}
	if len(old) > 0 {
		backup := ".local/state/rwr/omarchy/backups/" + hash([]byte(path)) + "-" + hash(old)
		if err := r.MkdirAll(filepath.Dir(backup), 0700); err != nil {
			return false, err
		}
		if err := r.WriteFile(backup, old, 0600); err != nil {
			return false, err
		}
	}
	if err := r.Rename(temporary, path); err != nil {
		return false, err
	}
	actual, err := r.ReadFile(path)
	if err != nil || !bytes.Equal(actual, data) {
		return true, fmt.Errorf("%s did not retain declared content", path)
	}
	return true, nil
}
func (c *Client) write(path string, data []byte, mode fs.FileMode) (bool, error) {
	old, err := c.read(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	return c.replace(path, old, data, mode)
}
func (c *Client) settings(o Operation, snap *Snapshot) (bool, error) {
	config, original, err := c.config()
	if err != nil {
		return false, err
	}
	target := config
	if o.Plugin != nil && o.Kind != "widget" {
		m, ok := snap.Plugins[o.Plugin.ID]
		if !ok {
			return false, fmt.Errorf("plugin %s is unavailable", o.Plugin.ID)
		}
		if !m.Enabled {
			// A settings entry must not implicitly activate a disabled third-party plugin.
			disabled := array(config["disabledPlugins"])
			found := false
			for _, v := range disabled {
				if v == m.ID {
					found = true
				}
			}
			if !found {
				config["disabledPlugins"] = append(disabled, m.ID)
			}
		}
		target, err = pluginEntry(config, m.ID, true)
		if err != nil {
			return false, err
		}
	}
	if o.Kind == "widget" {
		if len(c.widgets) > 0 {
			err = reconcileWidgets(config, c.widgets, snap)
		} else {
			err = placeWidget(config, *o.Plugin, snap)
		}
	} else {
		err = setPath(target, o.Path, o.Value, o.Unset)
	}
	if err != nil {
		return false, err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return false, err
	}
	data = append(data, '\n')
	// Formatting alone is not a desired-state change.
	var old map[string]any
	if len(original) > 0 {
		if err := json.Unmarshal(original, &old); err != nil {
			return false, err
		}
	}
	if equal(old, config) {
		return false, nil
	}
	return c.replace(".config/omarchy/shell.json", original, data, 0600)
}
func equal(a, b any) bool {
	x, err := json.Marshal(a)
	if err != nil {
		return false
	}
	y, err := json.Marshal(b)
	return err == nil && bytes.Equal(x, y)
}
func stringValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
func array(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}
func mapping(v any) map[string]any {
	if s, ok := v.(map[string]any); ok {
		return s
	}
	return nil
}
func number(v any) float64 {
	if n, ok := v.(float64); ok {
		return n
	}
	return 0
}
func cloneConfig(config map[string]any) (map[string]any, error) {
	raw, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	err = json.Unmarshal(raw, &result)
	return result, err
}
func object(m map[string]any, key string) (map[string]any, error) {
	v, ok := m[key]
	if !ok {
		r := map[string]any{}
		m[key] = r
		return r, nil
	}
	r, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an object", key)
	}
	return r, nil
}
func setPath(m map[string]any, path []string, value any, unset bool) error {
	if len(path) == 0 {
		return fmt.Errorf("empty setting path")
	}
	for _, k := range path[:len(path)-1] {
		if unset && m[k] == nil {
			return nil
		}
		next, err := object(m, k)
		if err != nil {
			return err
		}
		m = next
	}
	if unset {
		delete(m, path[len(path)-1])
	} else {
		m[path[len(path)-1]] = value
	}
	return nil
}
func pluginEntry(config map[string]any, id string, create bool) (map[string]any, error) {
	var found map[string]any
	visit := func(entries any) error {
		if entries == nil {
			return nil
		}
		list, ok := entries.([]any)
		if !ok {
			return fmt.Errorf("plugin entries must be an array")
		}
		for _, v := range list {
			entry, ok := v.(map[string]any)
			if !ok {
				return fmt.Errorf("plugin entry must be an object")
			}
			if entry["id"] == id {
				if found != nil {
					return fmt.Errorf("duplicate plugin %s in shell.json", id)
				}
				found = entry
			}
		}
		return nil
	}
	if err := visit(config["plugins"]); err != nil {
		return nil, err
	}
	bar, err := object(config, "bar")
	if err != nil {
		return nil, err
	}
	layout, err := object(bar, "layout")
	if err != nil {
		return nil, err
	}
	for _, section := range []string{"left", "center", "right"} {
		if err := visit(layout[section]); err != nil {
			return nil, err
		}
	}
	if found != nil || !create {
		return found, nil
	}
	entry := map[string]any{"id": id}
	list := array(config["plugins"])
	config["plugins"] = append(list, entry)
	return entry, nil
}
func hasKind(m Manifest, k string) bool {
	for _, v := range m.Kinds {
		if v == k {
			return true
		}
	}
	return false
}
func placeWidget(config map[string]any, p types.OmarchyPlugin, snap *Snapshot) error {
	m, ok := snap.Plugins[p.ID]
	if !ok || !hasKind(m, "bar-widget") {
		return fmt.Errorf("%s has no bar-widget capability", p.ID)
	}
	bar, err := object(config, "bar")
	if err != nil {
		return err
	}
	layout, err := object(bar, "layout")
	if err != nil {
		return err
	}
	entry, err := pluginEntry(config, p.ID, false)
	if err != nil {
		return err
	}
	if entry == nil {
		entry = map[string]any{"id": p.ID}
	}
	oldSection := ""
	oldIndex := 0
	for _, s := range []string{"left", "center", "right"} {
		items := array(layout[s])
		var kept []any
		for i, v := range items {
			if mapping(v)["id"] == p.ID {
				oldSection = s
				oldIndex = i
			} else {
				kept = append(kept, v)
			}
		}
		if kept == nil {
			kept = []any{}
		}
		layout[s] = kept
	}
	plugins := array(config["plugins"])
	kept := []any{}
	for _, v := range plugins {
		if mapping(v)["id"] != p.ID {
			kept = append(kept, v)
		}
	}
	config["plugins"] = kept
	w := p.Widget
	if w.Visible != nil && !*w.Visible {
		// Preserve overlay/service activation when removing its optional widget.
		if m.Enabled && !m.FirstParty {
			config["plugins"] = append(kept, entry)
		}
		return nil
	}
	if !m.Enabled {
		return fmt.Errorf("cannot place disabled plugin %s", p.ID)
	}
	section := w.Section
	if section == "" {
		section = oldSection
	}
	if section == "" {
		section = "center"
	}
	anchor := w.Before
	if anchor == "" {
		anchor = w.After
	}
	if anchor != "" && w.Section == "" {
		for _, s := range []string{"left", "center", "right"} {
			for _, v := range array(layout[s]) {
				if mapping(v)["id"] == anchor {
					section = s
				}
			}
		}
	}
	items := array(layout[section])
	index := len(items)
	if oldSection == section {
		index = oldIndex
	}
	if w.Index != nil {
		index = *w.Index
	}
	if anchor != "" {
		index = -1
		for i, v := range items {
			if mapping(v)["id"] == anchor {
				index = i
				if w.After != "" {
					index++
				}
				break
			}
		}
		if index < 0 {
			return fmt.Errorf("widget anchor %s not found in %s", anchor, section)
		}
	}
	if index > len(items) {
		return fmt.Errorf("widget index %d exceeds section length %d", index, len(items))
	}
	items = append(items, nil)
	copy(items[index+1:], items[index:])
	items[index] = entry
	layout[section] = items
	return nil
}
func trim(data []byte) string { return strings.TrimSpace(string(data)) }
