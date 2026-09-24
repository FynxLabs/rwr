package omarchy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/freehold-digital/rwr/internal/system"
)

type Result struct {
	Operation Operation
	Changed   bool
	Err       error
	Blocked   bool
}

func (c *Client) Preflight(ctx context.Context, ops []Operation, snap *Snapshot) error {
	planned := map[string]bool{}
	replacements := map[string]string{}
	enabled := map[string]bool{}
	for _, o := range ops {
		if o.Kind == "enabled" {
			enabled[o.Plugin.ID] = *o.Plugin.Enabled
		}
	}
	for _, o := range ops {
		if o.Kind != "plugin" || o.Plugin.State == "absent" {
			continue
		}
		planned[o.Plugin.ID] = true
		manifest, present := snap.Plugins[o.Plugin.ID]
		source := manifest.Clone
		if o.Plugin.Source != nil && o.Plugin.Source.Clone != "" {
			source = o.Plugin.Source.Clone
		}
		active := manifest.Enabled
		if !present && source != "" {
			active = true
		}
		if desired, ok := enabled[o.Plugin.ID]; ok {
			active = desired
		}
		if source != "" && active {
			if previous := replacements[source]; previous != "" && previous != o.Plugin.ID {
				return fmt.Errorf("competing clones for %s", source)
			}
			replacements[source] = o.Plugin.ID
		}
	}
	for _, o := range ops {
		if o.Kind == "screensaver" && o.Screensaver.Mode == "external" {
			idle, exists := snap.Plugins["omarchy.idle"]
			active := idle.Enabled
			requested, explicit := enabled["omarchy.idle"]
			if explicit {
				active = requested
			}
			if !exists || !idle.FirstParty || !active {
				return fmt.Errorf("external screensaver routing requires the stock omarchy.idle plugin enabled")
			}
			for id, plugin := range snap.Plugins {
				if plugin.Clone != "omarchy.idle" {
					continue
				}
				active := plugin.Enabled
				if want, declared := enabled[id]; declared {
					active = want
				}
				if active {
					return fmt.Errorf("external screensaver routing cannot use active idle clone %s; explicitly restore stock idle", id)
				}
			}
		}
		if o.Plugin != nil && o.Kind != "plugin" {
			if _, ok := snap.Plugins[o.Plugin.ID]; !ok && !planned[o.Plugin.ID] {
				return fmt.Errorf("%s requires unavailable plugin %s", o.ID, o.Plugin.ID)
			}
		}
		if o.Kind == "enabled" && *o.Plugin.Enabled {
			if clone := replacements[o.Plugin.ID]; clone != "" {
				return fmt.Errorf("cannot enable %s and replacement %s together", o.Plugin.ID, clone)
			}
		}
		if o.Kind == "default" {
			app := applications[o.Path[0]][stringValue(o.Value)]
			if _, err := c.LookPath(app); err != nil {
				return fmt.Errorf("default %s requires installed application %s", o.Path[0], app)
			}
		}
		if o.Kind == "hook" && o.Hook.State != "absent" {
			if _, err := readExternal(o.Hook.Source); err != nil {
				return err
			}
		}
		if o.Kind == "background" {
			if _, err := readExternal(stringValue(o.Value)); err != nil {
				return err
			}
		}
	}
	return nil
}
func (c *Client) Apply(ctx context.Context, ops []Operation, emit func(Result)) error {
	var err error
	ops, err = Merge(ops)
	if err != nil {
		return err
	}
	if system.IsDryRun() {
		for _, o := range ops {
			emit(Result{Operation: o})
		}
		return nil
	}
	snap, err := c.Discover(ctx)
	if err != nil {
		return err
	}
	if err := c.Preflight(ctx, ops, snap); err != nil {
		return err
	}
	cleanup, err := c.prepareSources(ctx, ops, snap)
	if err != nil {
		return err
	}
	defer cleanup()
	if err := c.validatePlacement(ops, snap); err != nil {
		return err
	}
	c.widgets = nil
	for _, o := range ops {
		if o.Kind == "widget" {
			c.widgets = append(c.widgets, o)
		}
	}
	c.hooks = nil
	for _, o := range ops {
		if o.Kind == "hook" && o.Hook.Event == "theme-set" && o.Hook.State != "absent" {
			c.hooks = append(c.hooks, *o.Hook)
		}
	}
	failed := map[string]bool{}
	var failures []error
	themeChanged := false
	for _, o := range ops {
		result := Result{Operation: o}
		if snap == nil {
			result.Err = fmt.Errorf("desktop discovery lost after mutation")
			result.Blocked = true
		} else if err := ctx.Err(); err != nil {
			result.Err = err
			result.Blocked = true
		} else if (o.Plugin != nil && failed["plugin/"+o.Plugin.ID]) || ((o.Kind == "theme-active" || o.Kind == "background") && failed["theme"]) {
			result.Err = fmt.Errorf("prerequisite failed")
			result.Blocked = true
		} else {
			result.Changed, result.Err = c.applyOne(ctx, o, snap, themeChanged)
			if result.Err == nil && (o.Kind == "plugin" || o.Kind == "enabled" || o.Kind == "setting" || o.Kind == "widget") {
				if result.Changed {
					snap, result.Err = c.waitForResource(ctx, o)
				} else {
					var satisfied bool
					satisfied, result.Err = c.satisfied(ctx, o, snap)
					if result.Err == nil && !satisfied {
						result.Err = fmt.Errorf("%s differs from desired state", o.ID)
					}
				}
			}

		}
		if o.Kind == "theme-source" && result.Changed {
			themeChanged = true
		}
		if result.Err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", o.ID, result.Err))
			if o.Plugin != nil {
				failed["plugin/"+o.Plugin.ID] = true
			}
			if strings.HasPrefix(o.Kind, "theme") || o.Kind == "hook" {
				failed["theme"] = true
			}
		}
		emit(result)

	}
	return errors.Join(failures...)
}
func (c *Client) applyOne(ctx context.Context, o Operation, snap *Snapshot, themeChanged bool) (bool, error) {
	switch o.Kind {
	case "plugin":
		return c.plugin(ctx, o, snap)
	case "enabled":
		p := snap.Plugins[o.Plugin.ID]
		if p.Enabled == *o.Plugin.Enabled {
			return false, nil
		}
		action := "disable"
		if *o.Plugin.Enabled {
			action = "enable"
		}
		if err := c.command("omarchy", "plugin", action, p.ID); err != nil {
			return true, err
		}
		if !*o.Plugin.Enabled && p.FirstParty && hasKind(p, "bar-widget") {
			config, original, err := c.config()
			if err != nil {
				return true, err
			}
			disabled := array(config["disabledPlugins"])
			exists := false
			for _, id := range disabled {
				if id == p.ID {
					exists = true
				}
			}
			if !exists {
				config["disabledPlugins"] = append(disabled, p.ID)
				raw, err := json.MarshalIndent(config, "", "  ")
				if err != nil {
					return true, err
				}
				if _, err := c.replace(".config/omarchy/shell.json", original, append(raw, '\n'), 0600); err != nil {
					return true, err
				}
			}
		}
		return true, nil
	case "setting", "widget":
		return c.settings(o, snap)
	case "theme-source":
		return c.themeSource(ctx, o)
	case "theme-active":
		current, err := c.read(".local/state/omarchy/current/theme.name")
		if err == nil && trim(current) == stringValue(o.Value) && !themeChanged {
			return false, nil
		}
		if err := c.activateTheme(stringValue(o.Value)); err != nil {
			return true, err
		}
		current, err = c.read(".local/state/omarchy/current/theme.name")
		if err != nil || trim(current) != stringValue(o.Value) {
			return true, fmt.Errorf("active theme did not persist")
		}
		return true, nil
	case "background":
		same, err := c.satisfied(ctx, o, snap)
		if err == nil && same {
			return false, nil
		}
		if err := c.command("omarchy", "theme", "bg", "set", stringValue(o.Value)); err != nil {
			return true, err
		}
		same, err = c.satisfied(ctx, o, snap)
		if err == nil && !same {
			err = fmt.Errorf("background did not persist")
		}
		return true, err
	case "default":
		kind := o.Path[0]
		value := stringValue(o.Value)
		raw, err := c.Read(ctx, "omarchy", "default", kind)
		if err != nil {
			return false, fmt.Errorf("default %s query unavailable", kind)
		}
		if trim(raw) == value || (value == "zed" && trim(raw) == "zeditor") {
			return false, nil
		}
		applyErr := c.command("omarchy", "default", kind, value)
		raw, err = c.Read(ctx, "omarchy", "default", kind)
		if err != nil || (trim(raw) != value && (value != "zed" || trim(raw) != "zeditor")) {
			return true, errors.Join(fmt.Errorf("default %s did not persist", kind), applyErr, err)
		}
		// Native setters may persist successfully and then fail to notify the shell.
		return true, nil
	case "hook":
		return c.hook(o)
	case "screensaver":
		return c.screensaver(ctx, o.Screensaver)
	}
	return false, fmt.Errorf("unknown Omarchy resource kind %s", o.Kind)
}
func (c *Client) hook(o Operation) (bool, error) {
	h := o.Hook
	path := ".config/omarchy/hooks/" + h.Event + ".d/" + h.Name
	old, err := c.read(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	state, err := c.ownership()
	if err != nil {
		return false, err
	}
	record, owned := state[o.ID]
	if owned {
		existing, err := c.read(hookPayload(*h))
		if err != nil || record.Source != "sha256:"+hash(existing) {
			return false, fmt.Errorf("hook payload was modified or is unavailable")
		}
	}
	if h.State == "absent" {
		if old == nil {
			return false, nil
		}
		if !owned || record.Hash != hash(old) {
			return false, fmt.Errorf("hook is unowned or modified")
		}
		root, err := os.OpenRoot(c.Home)
		if err != nil {
			return false, err
		}
		defer closeRoot(root)
		return true, root.Remove(path)
	}
	payload, err := readExternal(h.Source)
	if err != nil {
		return false, err
	}
	data := c.hookWrapper(*h)
	if equal(string(old), string(data)) {
		previous, err := c.read(hookPayload(*h))
		if err == nil && equal(string(previous), string(payload)) {
			return false, nil
		}
	}
	if old != nil && (!owned || record.Hash != hash(old)) {
		return false, fmt.Errorf("existing hook is unowned or modified")
	}
	payloadChanged, err := c.write(hookPayload(*h), payload, 0700)
	if err != nil {
		return false, err
	}
	changed, err := c.replace(path, old, data, 0700)
	if err != nil {
		return changed, err
	}
	return changed || payloadChanged, c.remember(o.ID, "sha256:"+hash(payload), hash(data))
}
func (c *Client) Satisfied(ctx context.Context, o Operation) (bool, error) {
	snap, err := c.Discover(ctx)
	if err != nil {
		return false, err
	}
	return c.satisfied(ctx, o, snap)
}

// SatisfiedWithSnapshot evaluates an operation without repeating desktop discovery.
func (c *Client) SatisfiedWithSnapshot(ctx context.Context, o Operation, snap *Snapshot) (bool, error) {
	return c.satisfied(ctx, o, snap)
}
func (c *Client) satisfied(ctx context.Context, o Operation, snap *Snapshot) (bool, error) {
	switch o.Kind {
	case "plugin":
		p, ok := snap.Plugins[o.Plugin.ID]
		if o.Plugin.State == "absent" {
			return !ok, nil
		}
		if !ok {
			return false, nil
		}
		if s := o.Plugin.Source; s != nil {
			if s.Clone != "" {
				return p.Clone == s.Clone, nil
			}
			if err := c.sourceMatches(ctx, p.SourceDir, s); err != nil {
				return false, err
			}
			if s.Ref != "" {
				a, err := c.Read(ctx, "git", "-C", p.SourceDir, "rev-parse", "HEAD")
				if err != nil {
					return false, err
				}
				b, err := c.Read(ctx, "git", "-C", p.SourceDir, "rev-parse", "--verify", s.Ref+"^{commit}")
				return trim(a) == trim(b), err
			}
		}
		return true, nil
	case "enabled":
		p, ok := snap.Plugins[o.Plugin.ID]
		return ok && p.Enabled == *o.Plugin.Enabled, nil
	case "setting", "widget":
		config, err := cloneConfig(snap.Config)
		if err != nil {
			return false, err
		}
		if o.Kind == "widget" {
			if err := placeWidget(config, *o.Plugin, snap); err != nil {
				return false, err
			}
			return equal(config, snap.Config), nil
		}
		target := config
		if o.Plugin != nil {
			var err error
			target, err = pluginEntry(config, o.Plugin.ID, false)
			if err != nil {
				return false, err
			}
			if target == nil {
				return o.Unset, nil
			}
		}
		var value any = target
		exists := true
		for _, k := range o.Path {
			m, ok := value.(map[string]any)
			if !ok {
				exists = false
				break
			}
			value, exists = m[k]
			if !exists {
				break
			}
		}
		if o.Unset {
			return !exists, nil
		}
		return exists && equal(value, o.Value), nil
	case "theme-active":
		raw, err := c.read(".local/state/omarchy/current/theme.name")
		return trim(raw) == o.Value, err
	case "theme-source":
		path := filepath.Join(c.Home, ".config/omarchy/themes", o.Theme.Name)
		if err := c.sourceMatches(ctx, path, o.Theme.Source); err != nil {
			return false, err
		}
		if ref := o.Theme.Source.Ref; ref != "" {
			a, err := c.Read(ctx, "git", "-C", path, "rev-parse", "HEAD")
			if err != nil {
				return false, err
			}
			b, err := c.Read(ctx, "git", "-C", path, "rev-parse", "--verify", ref+"^{commit}")
			return trim(a) == trim(b), err
		}
		return true, nil
	case "background":
		root, err := os.OpenRoot(c.Home)
		if err != nil {
			return false, err
		}
		defer closeRoot(root)
		path, err := root.Readlink(".local/state/omarchy/current/background")
		return path == o.Value, err
	case "default":
		raw, err := c.Read(ctx, "omarchy", "default", o.Path[0])
		return trim(raw) == o.Value || (o.Value == "zed" && trim(raw) == "zeditor"), err
	case "hook":
		raw, err := c.read(".config/omarchy/hooks/" + o.Hook.Event + ".d/" + o.Hook.Name)
		if o.Hook.State == "absent" {
			return errors.Is(err, fs.ErrNotExist), nil
		}
		if err != nil {
			return false, err
		}
		want := c.hookWrapper(*o.Hook)
		payload, err := c.read(hookPayload(*o.Hook))
		if err != nil {
			return false, err
		}
		source, err := readExternal(o.Hook.Source)
		return equal(string(raw), string(want)) && equal(string(payload), string(source)), err
	case "screensaver":
		_, profile, err := c.profile()
		if err != nil {
			return false, err
		}
		if o.Screensaver.Mode == "stock" {
			return !strings.Contains(string(profile), routeStart), nil
		}
		route, err := c.read(routePath)
		return equal(string(route), string(c.routeContent(o.Screensaver))) && strings.Contains(string(profile), string(c.routeBlock())), err
	}
	return false, fmt.Errorf("unsupported status resource")
}

func (c *Client) validatePlacement(ops []Operation, snap *Snapshot) error {
	config, err := cloneConfig(snap.Config)
	if err != nil {
		return err
	}
	simulated := &Snapshot{Plugins: map[string]Manifest{}, Config: config}
	for id, m := range snap.Plugins {
		simulated.Plugins[id] = m
	}
	for _, o := range ops {
		if o.Kind == "plugin" {
			if o.Plugin.State == "absent" {
				delete(simulated.Plugins, o.Plugin.ID)
				continue
			}
			if _, ok := simulated.Plugins[o.Plugin.ID]; !ok {
				var m Manifest
				if path := c.prepared[o.ID]; path != "" {
					raw, err := readExternal(filepath.Join(path, "manifest.json"))
					if err != nil {
						return err
					}
					if err := json.Unmarshal(raw, &m); err != nil {
						return err
					}
				} else if o.Plugin.Source != nil && o.Plugin.Source.Clone != "" {
					m = simulated.Plugins[o.Plugin.Source.Clone]
					m.FirstParty = false
					m.Enabled = true
				} else {
					return fmt.Errorf("plugin %s unavailable", o.Plugin.ID)
				}
				m.ID = o.Plugin.ID
				simulated.Plugins[m.ID] = m
			}
		}
		if o.Kind == "enabled" {
			m := simulated.Plugins[o.Plugin.ID]
			m.Enabled = *o.Plugin.Enabled
			simulated.Plugins[m.ID] = m
		}
	}
	widgets := []Operation{}
	for _, o := range ops {
		if o.Kind == "widget" {
			widgets = append(widgets, o)
		}
	}
	return reconcileWidgets(config, widgets, simulated)
}
func reconcileWidgets(config map[string]any, widgets []Operation, snap *Snapshot) error {
	// First establish visibility, then apply ordering against the complete desired
	// set. This permits an anchor installed by another declaration in the same run.
	for _, o := range widgets {
		if o.Kind == "widget" {
			p := *o.Plugin
			w := *p.Widget
			w.Before = ""
			w.After = ""
			w.Index = nil
			p.Widget = &w
			if err := placeWidget(config, p, snap); err != nil {
				return err
			}
		}
	}
	// A bounded fixed point detects cycles and contradictory index declarations.
	for pass := 0; pass <= len(widgets); pass++ {
		before, err := cloneConfig(config)
		if err != nil {
			return err
		}
		for _, o := range widgets {
			if err := placeWidget(config, *o.Plugin, snap); err != nil {
				return err
			}
		}
		if equal(before, config) {
			break
		}
	}
	for _, o := range widgets {
		candidate, err := cloneConfig(config)
		if err != nil {
			return err
		}
		if err := placeWidget(candidate, *o.Plugin, snap); err != nil {
			return err
		}
		if !equal(candidate, config) {
			return fmt.Errorf("conflicting widget placement for %s", o.Plugin.ID)
		}
	}
	return nil
}

func (c *Client) waitForResource(ctx context.Context, o Operation) (*Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var last error
	for {
		snap, err := c.Discover(ctx)
		if err == nil {
			var satisfied bool
			satisfied, err = c.satisfied(ctx, o, snap)
			if err == nil && satisfied {
				return snap, nil
			}
			if err == nil {
				err = fmt.Errorf("%s did not converge", o.ID)
			}
		}
		last = err
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, errors.Join(last, ctx.Err())
		case <-timer.C:
		}
	}
}
