// Package omarchy reconciles explicitly declared desktop resources.
package omarchy

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
)

type Operation struct {
	ID          string
	Kind        string
	Origin      string
	Plugin      *types.OmarchyPlugin
	Theme       *types.OmarchyTheme
	Hook        *types.OmarchyHook
	Screensaver *types.OmarchyScreensaver
	Path        []string
	Value       any
	Unset       bool
}

var identifier = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9._+-]*$`)

func validID(s string) bool { return identifier.MatchString(s) && s != "." && s != ".." }
func validSource(s *types.OmarchySource, plugin bool) error {
	if s == nil {
		return nil
	}
	n := 0
	for _, v := range []string{s.Git, s.Path, s.Clone} {
		if v != "" {
			n++
		}
	}
	if n != 1 {
		return fmt.Errorf("source requires exactly one of git, path, clone")
	}
	if s.Ref != "" && (s.Git == "" || strings.HasPrefix(s.Ref, "-")) {
		return fmt.Errorf("ref requires git and cannot be an option")
	}
	if s.Clone != "" && (!plugin || !validID(s.Clone) || !strings.HasPrefix(s.Clone, "omarchy.")) {
		return fmt.Errorf("clone requires a built-in plugin ID")
	}
	if s.Git != "" {
		u, err := url.Parse(s.Git)
		if err != nil || (u.Scheme != "https" && u.Scheme != "ssh") || u.Host == "" || u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("git requires an HTTPS or SSH URL without query or fragment")
		}
		if u.User != nil {
			if _, hasPassword := u.User.Password(); hasPassword {
				return fmt.Errorf("git URLs cannot contain passwords; use configured Git authentication")
			}
		}
	}
	return nil
}
func Load(raw []byte, format, origin string, cfg *types.InitConfig) ([]Operation, error) {
	return load(raw, format, origin, cfg, map[string]bool{})
}
func load(raw []byte, format, origin string, cfg *types.InitConfig, visiting map[string]bool) ([]Operation, error) {
	origin, err := filepath.Abs(origin)
	if err != nil {
		return nil, err
	}
	if visiting[origin] {
		return nil, fmt.Errorf("circular Omarchy import at %s", origin)
	}
	visiting[origin] = true
	defer delete(visiting, origin)
	var d types.ConfigData
	if err := helpers.DecodeBlueprintInto(raw, format, types.BlueprintTypeConfiguration, helpers.TreeSchemaVersion(cfg), &d); err != nil {
		return nil, err
	}
	var ops []Operation
	for _, config := range helpers.FilterByProfiles(d.Configurations, cfg.Variables.Flags.Profiles) {
		entry := config.AsOmarchySetup()
		if config.Tool != "omarchy" {
			if config.Import != "" {
				return nil, fmt.Errorf("configuration %q uses import with tool %q; import is only supported for omarchy configurations", config.Name, config.Tool)
			}
			if hasOmarchyResources(entry) {
				return nil, fmt.Errorf("configuration %q contains omarchy fields but uses tool %q", config.Name, config.Tool)
			}
			continue
		}
		if config.Import != "" {
			if config.Action != "" || entry.Name != "" || hasOmarchyResources(entry) || hasForeignConfigurationFields(config) {
				return nil, fmt.Errorf("import entry cannot also declare resources")
			}
			path := filepath.Join(filepath.Dir(origin), config.Import)
			data, err := os.ReadFile(path) // #nosec G304 -- explicitly selected blueprint import, not a mutation target
			if err != nil {
				return nil, err
			}
			f, err := helpers.FormatForPath(path)
			if err != nil {
				return nil, err
			}
			data, err = helpers.ResolveTemplateForValidation(data, cfg.Variables)
			if err != nil {
				return nil, err
			}
			imported, err := load(data, f, path, cfg, visiting)
			if err != nil {
				return nil, err
			}
			ops = append(ops, imported...)
			continue
		}
		if config.Action != "" && config.Action != types.ConfigurationActionSet {
			return nil, fmt.Errorf("%s (%s): unsupported action %q: the only supported action is %q", entry.Name, origin, config.Action, types.ConfigurationActionSet)
		}
		if hasForeignConfigurationFields(config) {
			return nil, fmt.Errorf("%s (%s): omarchy configuration contains fields for another configuration tool", entry.Name, origin)
		}
		if entry.Name == "" {
			return nil, fmt.Errorf("omarchy configuration requires name")
		}
		current, err := entryOperations(entry, origin)
		if err != nil {
			return nil, fmt.Errorf("%s (%s): %w", entry.Name, origin, err)
		}
		ops = append(ops, current...)
	}
	return Merge(ops)
}

func hasOmarchyResources(entry types.OmarchySetup) bool {
	return len(entry.Plugins) > 0 || entry.Shell != nil || entry.Theme != nil ||
		entry.Defaults != nil || len(entry.Hooks) > 0 || entry.Integrations != nil
}

func hasForeignConfigurationFields(config types.Configuration) bool {
	return len(config.Names) > 0 || config.Elevated || config.RunOnce || config.File != "" ||
		config.Schema != "" || config.Path != "" || config.Key != "" || config.Value != nil ||
		config.Domain != "" || config.Kind != "" || config.Type != "" || len(config.Settings) > 0
}
func entryOperations(e types.OmarchySetup, origin string) ([]Operation, error) {
	var ops []Operation
	add := func(o Operation) { o.Origin = origin; ops = append(ops, o) }
	absolute := func(p string) string {
		if p == "" || filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(filepath.Dir(origin), p)
	}
	for _, p := range e.Plugins {
		if !validID(p.ID) {
			return nil, fmt.Errorf("invalid plugin ID %q", p.ID)
		}
		if err := validSource(p.Source, true); err != nil {
			return nil, err
		}
		if p.State != "" && p.State != "present" && p.State != "absent" {
			return nil, fmt.Errorf("invalid plugin state")
		}
		if strings.HasPrefix(p.ID, "omarchy.") && (p.Source != nil || p.State == "absent") {
			return nil, fmt.Errorf("built-in %s cannot be installed or removed; use enabled", p.ID)
		}
		if p.Update && (p.Source == nil || p.Source.Git == "" || p.Source.Ref != "") {
			return nil, fmt.Errorf("update requires an unpinned git source")
		}
		if p.State == "absent" && (p.Source != nil || p.Enabled != nil || p.Widget != nil || len(p.Settings) > 0 || len(p.Unset) > 0) {
			return nil, fmt.Errorf("absent plugin cannot declare configuration")
		}
		if p.Source != nil {
			p.Source.Path = absolute(p.Source.Path)
		}
		presence := p
		presence.Enabled = nil
		presence.Settings = nil
		presence.Unset = nil
		presence.Widget = nil
		if p.State == "" {
			presence.State = "present"
		}
		{
			add(Operation{ID: "plugin/" + p.ID, Kind: "plugin", Plugin: &presence})
		}
		if p.Enabled != nil {
			q := types.OmarchyPlugin{ID: p.ID, Enabled: p.Enabled}
			add(Operation{ID: "plugin/" + p.ID + "/enabled", Kind: "enabled", Plugin: &q})
		}
		options, err := settingOperations(p.ID, p.Settings, nil)
		if err != nil {
			return nil, err
		}
		for _, o := range options {
			add(o)
		}
		for _, k := range p.Unset {
			if !validSettingPath(k) {
				return nil, fmt.Errorf("invalid unset path")
			}
			add(Operation{ID: "plugin/" + p.ID + "/settings/" + k, Kind: "setting", Plugin: &types.OmarchyPlugin{ID: p.ID}, Path: strings.Split(k, "."), Unset: true})
		}
		if w := p.Widget; w != nil {
			n := 0
			for _, v := range []bool{w.Before != "", w.After != "", w.Index != nil} {
				if v {
					n++
				}
			}
			if n > 1 || (w.Index != nil && *w.Index < 0) || (w.Section != "" && w.Section != "left" && w.Section != "center" && w.Section != "right") {
				return nil, fmt.Errorf("invalid widget placement for %s", p.ID)
			}
			if (w.Before != "" && (!validID(w.Before) || w.Before == p.ID)) || (w.After != "" && (!validID(w.After) || w.After == p.ID)) {
				return nil, fmt.Errorf("invalid widget anchor")
			}
			if w.Visible != nil && !*w.Visible && (n > 0 || w.Section != "") {
				return nil, fmt.Errorf("hidden widget cannot declare placement")
			}
			if p.Enabled != nil && !*p.Enabled && (w.Visible == nil || *w.Visible) {
				return nil, fmt.Errorf("disabled plugin cannot request a visible widget")
			}
			q := types.OmarchyPlugin{ID: p.ID, Widget: w}
			add(Operation{ID: "plugin/" + p.ID + "/widget", Kind: "widget", Plugin: &q})
		}
	}
	if e.Shell != nil {
		b, err := json.Marshal(e.Shell)
		if err != nil {
			return nil, err
		}
		var settings map[string]any
		if err := json.Unmarshal(b, &settings); err != nil {
			return nil, err
		}
		for section, v := range settings {
			for k, val := range mapping(v) {
				if section == "idle" && number(val) < 0 {
					return nil, fmt.Errorf("idle time cannot be negative")
				}
				if k == "position" && val != "top" && val != "bottom" && val != "left" && val != "right" {
					return nil, fmt.Errorf("invalid bar position")
				}
				add(Operation{ID: "shell/" + section + "/" + k, Kind: "setting", Path: []string{section, k}, Value: val})
			}
		}
	}
	if t := e.Theme; t != nil {
		if !validID(t.Name) {
			return nil, fmt.Errorf("invalid theme name")
		}
		if err := validSource(t.Source, false); err != nil {
			return nil, err
		}
		if t.Source != nil {
			t.Source.Path = absolute(t.Source.Path)
			q := *t
			q.Active = nil
			q.Background = ""
			add(Operation{ID: "theme/" + t.Name, Kind: "theme-source", Theme: &q})
		}
		if t.Active != nil && *t.Active {
			add(Operation{ID: "theme/active", Kind: "theme-active", Value: t.Name})
		}
		if t.Background != "" {
			add(Operation{ID: "theme/background", Kind: "background", Value: absolute(t.Background)})
		}
	}
	if d := e.Defaults; d != nil {
		for k, v := range map[string]string{"browser": d.Browser, "terminal": d.Terminal, "editor": d.Editor} {
			if v != "" {
				if _, ok := applications[k][v]; !ok {
					return nil, fmt.Errorf("unsupported default %s %q", k, v)
				}
				add(Operation{ID: "default/" + k, Kind: "default", Path: []string{k}, Value: v})
			}
		}
	}
	for _, h := range e.Hooks {
		if !validID(h.Name) || !knownEvent(h.Event) || (h.State != "" && h.State != "present" && h.State != "absent") {
			return nil, fmt.Errorf("invalid hook event/name/state")
		}
		if h.State != "absent" && h.Source == "" {
			return nil, fmt.Errorf("hook requires source")
		}
		h.Source = absolute(h.Source)
		add(Operation{ID: "hook/" + h.Event + "/" + h.Name, Kind: "hook", Hook: &h})
	}
	if e.Integrations != nil && e.Integrations.Screensaver != nil {
		s := e.Integrations.Screensaver
		if s.Mode != "stock" && s.Mode != "external" {
			return nil, fmt.Errorf("screensaver mode must be stock or external")
		}
		if s.Mode == "stock" && (s.Launch != nil || s.Check != nil || s.WindowClass != "") {
			return nil, fmt.Errorf("stock mode cannot declare external commands")
		}
		if s.Mode == "external" {
			if s.WindowClass != "org.omarchy.screensaver" {
				return nil, fmt.Errorf("external screensaver must declare windowClass org.omarchy.screensaver and implement its dismissal/lock lifecycle")
			}
			for _, c := range []*types.OmarchyExec{s.Launch, s.Check} {
				if c == nil || !filepath.IsAbs(c.Exec) || strings.ContainsAny(c.Exec, "\x00\n") {
					return nil, fmt.Errorf("external screensaver requires absolute launch and check executables")
				}
			}
		}
		add(Operation{ID: "integration/screensaver", Kind: "screensaver", Screensaver: s})
	}
	return ops, nil
}
func validSettingPath(s string) bool {
	for _, k := range strings.Split(s, ".") {
		if !validID(k) || k == "id" || k == "__proto__" || k == "constructor" || k == "omarchy" {
			return false
		}
	}
	return true
}
func knownEvent(s string) bool {
	switch s {
	case "theme-set", "font-set", "post-boot", "post-update", "battery-low", "pre-refresh-pacman":
		return true
	}
	return false
}
func Merge(ops []Operation) ([]Operation, error) {
	seen := map[string]Operation{}
	var result []Operation
	for _, o := range ops {
		if prev, ok := seen[o.ID]; ok {
			a, b := prev, o
			a.Origin = ""
			b.Origin = ""
			if a.Kind == "plugin" && b.Kind == "plugin" && a.Plugin.State == b.Plugin.State {
				if a.Plugin.Source == nil {
					a.Plugin.Source = b.Plugin.Source
				}
				if b.Plugin.Source == nil {
					b.Plugin.Source = a.Plugin.Source
				}
				a.Plugin.Update = a.Plugin.Update || b.Plugin.Update
				b.Plugin.Update = a.Plugin.Update
			}
			if !reflect.DeepEqual(a, b) {
				return nil, fmt.Errorf("conflicting %s in %s and %s", o.ID, prev.Origin, o.Origin)
			}
			seen[o.ID] = a
			for i := range result {
				if result[i].ID == o.ID {
					a.Origin = prev.Origin
					result[i] = a
				}
			}
			continue
		}
		seen[o.ID] = o
		result = append(result, o)
	}
	for _, o := range result {
		if o.Plugin != nil {
			if p, ok := seen["plugin/"+o.Plugin.ID]; ok && p.Plugin.State == "absent" && o.Kind != "plugin" {
				return nil, fmt.Errorf("absent plugin %s also has configuration", o.Plugin.ID)
			}
		}
	}
	for _, a := range result {
		if a.Kind != "setting" {
			continue
		}
		for _, b := range result {
			if a.ID != b.ID && strings.HasPrefix(b.ID, a.ID+".") {
				return nil, fmt.Errorf("overlapping settings %s (%s) and %s (%s)", a.ID, a.Origin, b.ID, b.Origin)
			}
		}
	}

	priority := map[string]int{"plugin": 0, "theme-source": 1, "hook": 2, "enabled": 3, "setting": 4, "widget": 5, "default": 6, "theme-active": 7, "background": 8, "screensaver": 9}
	sort.SliceStable(result, func(i, j int) bool {
		a, b := priority[result[i].Kind], priority[result[j].Kind]
		if a != b {
			return a < b
		}
		return result[i].ID < result[j].ID
	})
	return result, nil
}

var applications = map[string]map[string]string{
	"browser":  {"chromium": "chromium", "chrome": "google-chrome-stable", "brave": "brave", "brave-origin": "brave-origin", "edge": "microsoft-edge-stable", "firefox": "firefox", "zen": "zen-browser"},
	"terminal": {"alacritty": "alacritty", "foot": "foot", "ghostty": "ghostty", "kitty": "kitty"},
	"editor":   {"code": "code", "cursor": "cursor", "zed": "zeditor", "sublime_text": "sublime_text", "helix": "hx", "vim": "vim", "emacs": "emacs", "nvim": "nvim"},
}

func settingOperations(id string, settings map[string]any, prefix []string) ([]Operation, error) {
	var ops []Operation
	for k, v := range settings {
		if !validSettingPath(k) || strings.Contains(k, ".") {
			return nil, fmt.Errorf("invalid plugin setting key %q; use nested maps", k)
		}
		path := append(append([]string(nil), prefix...), k)
		if nested, ok := v.(map[string]any); ok && len(nested) > 0 {
			children, err := settingOperations(id, nested, path)
			if err != nil {
				return nil, err
			}
			ops = append(ops, children...)
		} else {
			ops = append(ops, Operation{ID: "plugin/" + id + "/settings/" + strings.Join(path, "."), Kind: "setting", Plugin: &types.OmarchyPlugin{ID: id}, Path: path, Value: v})
		}
	}
	return ops, nil
}
