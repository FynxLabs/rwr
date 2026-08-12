package processors

import (
	"fmt"
	"os"

	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
)

// ResolveStage1 reads and resolves the blueprint tree without executing
// anything: file routing (path- and content-based), template resolution, and
// strict decode per blueprint type. Problems become Diagnostics rather than
// errors - one bad file must not hide the rest of the tree from a validator
// or a progress display. The returned error is reserved for the tree being
// unreadable at all.
func ResolveStage1(initConfig *types.InitConfig) (*types.Plan, error) {
	plan := &types.Plan{
		Init:  initConfig,
		Files: map[string][]types.ResolvedFile{},
	}

	order, err := GetBlueprintRunOrder(initConfig)
	if err != nil {
		return nil, fmt.Errorf("error getting blueprint run order: %w", err)
	}

	location := initConfig.Init.Location
	fileOrder, err := GetBlueprintFileOrder(location, initConfig.Init.Order, initConfig.Init.RunOnlyListed, initConfig)
	if err != nil {
		return nil, fmt.Errorf("error getting blueprint file order: %w", err)
	}

	// The plan's order carries only the processors this tree configures. The
	// executor already skips processors with no files; keeping them in the
	// plan just rendered phantom rows (a tree with no ssh_keys blueprints
	// showed an ssh_keys processor pending forever).
	for _, processor := range order {
		if _, ok := fileOrder[processor]; ok {
			plan.Order = append(plan.Order, processor)
		}
	}

	for processor, files := range fileOrder {
		for _, rel := range files {
			path := location + string(os.PathSeparator) + rel

			format, formatErr := helpers.FormatForPath(path)
			if formatErr != nil {
				plan.Diags = append(plan.Diags, types.Diagnostic{Severity: types.SeverityError, Processor: processor, File: path, Msg: formatErr.Error()})
				continue
			}

			raw, readErr := os.ReadFile(path) // #nosec G304 -- operator's own blueprint tree
			if readErr != nil {
				plan.Diags = append(plan.Diags, types.Diagnostic{Severity: types.SeverityError, Processor: processor, File: path, Msg: readErr.Error()})
				continue
			}

			// Strict for the fixed namespaces, lenient only for UserDefined -
			// the same contract validate enforces.
			for _, ref := range helpers.UnknownTemplateReferences(raw, initConfig.Variables) {
				plan.Diags = append(plan.Diags, types.Diagnostic{
					Severity: types.SeverityError, Processor: processor, File: path,
					Msg: fmt.Sprintf("Template references %s, which does not exist", ref),
				})
			}

			resolved, resolveErr := helpers.ResolveTemplateForValidation(raw, initConfig.Variables)
			if resolveErr != nil {
				plan.Diags = append(plan.Diags, types.Diagnostic{Severity: types.SeverityError, Processor: processor, File: path, Msg: resolveErr.Error()})
				continue
			}

			// A multi-type file is cut down to this processor's sections,
			// exactly as the run loop does before dispatch; single-type files
			// pass through and keep strict decode's typo protection.
			subset, subsetFormat, subsetErr := subsetForProcessor(resolved, format, processor)
			if subsetErr != nil {
				plan.Diags = append(plan.Diags, types.Diagnostic{Severity: types.SeverityError, Processor: processor, File: path, Msg: subsetErr.Error()})
				continue
			}

			plan.Files[processor] = append(plan.Files[processor], types.ResolvedFile{
				Path:      path,
				Processor: processor,
				Format:    subsetFormat,
				Raw:       raw,
				Resolved:  subset,
			})
		}
	}

	return plan, nil
}
