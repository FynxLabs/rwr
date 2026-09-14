package processors

import (
	"encoding/json"
	"fmt"
	"time"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/omarchy"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

func omarchyOperations(files []types.ResolvedFile, cfg *types.InitConfig) ([]omarchy.Operation, error) {
	var ops []omarchy.Operation
	for _, file := range files {
		current, err := omarchy.Load(file.Resolved, file.Format, file.Path, cfg)
		if err != nil {
			return nil, err
		}
		ops = append(ops, current...)
	}
	return omarchy.Merge(ops)
}
func ProcessOmarchy(files []types.ResolvedFile, cfg *types.InitConfig) error {
	ops, err := omarchyOperations(files, cfg)
	if err != nil {
		return err
	}
	track := newProgress(types.BlueprintTypeOmarchy)
	track.expect("", len(ops))
	if system.IsDryRun() {
		for _, o := range ops {
			log.Infof("[DRY-RUN] Would reconcile Omarchy %s", o.ID)
			track.item("", o.ID, "reconcile", types.StatusPlanned, "live state not queried", 0)
		}
		return nil
	}
	c, err := omarchy.NewClient(cfg.Variables.Flags.Debug)
	if err != nil {
		return err
	}
	started := time.Now()
	done := map[string]bool{}
	err = c.Apply(system.RunContext(), ops, func(r omarchy.Result) {
		status := types.StatusPresent
		detail := "unchanged"
		if r.Changed {
			status = types.StatusOK
			detail = "applied"
		}
		if r.Err != nil {
			status = types.StatusFailed
			detail = r.Err.Error()
		}
		if r.Blocked {
			status = types.StatusSkipped
		}
		done[r.Operation.ID] = true
		track.itemIdentity("", r.Operation.ID, "reconcile", status, detail, time.Since(started), map[string]string{"resource": r.Operation.ID, "reversal": "explicit desired-state change required"})
	})
	if err != nil {
		for _, o := range ops {
			if !done[o.ID] {
				track.item("", o.ID, "reconcile", types.StatusFailed, "Omarchy preflight or discovery failed", 0)
			}
		}
		recordFailure(types.BlueprintTypeOmarchy, "desktop", err)
		return fmt.Errorf("omarchy setup: %w", err)
	}
	return nil
}
func omarchyResources(files []types.ResolvedFile, cfg *types.InitConfig) []types.Resource {
	ops, err := omarchyOperations(files, cfg)
	if err != nil {
		return nil
	}
	var result []types.Resource
	for _, o := range ops {
		raw, err := json.Marshal(o)
		if err != nil {
			continue
		}
		result = append(result, types.Resource{Processor: types.BlueprintTypeOmarchy, Name: o.ID, Action: "reconcile", Status: types.StatusPlanned, DesiredState: raw})
	}
	return result
}
