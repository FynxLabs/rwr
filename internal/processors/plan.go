package processors

import (
	"fmt"
	"os"
	"time"

	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
)

// Plan is what a run knows before it executes: the resolved blueprint tree.
// Stage 1 (parse, imports, schema, template resolution, diagnostics) needs
// only SetPaths and DetectOS, so `rwr validate` consumes it pre-init; stage 2
// (provider states, resource enumeration) runs after init and bootstrap,
// because bootstrap can install the package manager later blueprints depend
// on — detecting providers earlier produces wrong lanes.
type Plan struct {
	Init      *types.InitConfig
	Order     []string
	Files     map[string][]ResolvedFile
	Providers []ProviderState
	Resources []Resource
	Diags     []Diagnostic
}

// ResolvedFile is one blueprint file after stage 1: routed, read, and
// template-resolved, so no consumer reads or renders it a second time.
type ResolvedFile struct {
	Path      string
	Processor string
	Format    string
	Raw       []byte
	Resolved  []byte
}

// ProviderState is one detected provider and the lane it will run.
type ProviderState struct {
	Name      string
	Available bool
	Elevated  bool
}

// Resource is one unit of work a run performs (a package, a file, a service).
type Resource struct {
	Processor string
	Provider  string // empty for files, services, git, scripts
	Name      string // "neovim", "~/.config/nvim/"
	Action    string // install, copy, enable, clone
	Status    Status
	Detail    string
	Dur       time.Duration
}

// Status is a resource's outcome. `unknown` is required: rwr shells out and
// several providers do not report per-package results reliably — forcing
// those into ok or failed would be a lie in either direction.
type Status string

const (
	StatusOK      Status = "ok"
	StatusFailed  Status = "failed"
	StatusSkipped Status = "skipped"
	StatusUnknown Status = "unknown"
	StatusPlanned Status = "planned"
	StatusPresent Status = "present"
)

// Severity classifies a diagnostic.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Diagnostic is one stage-1 finding, positioned when the source position is
// known.
type Diagnostic struct {
	Severity  Severity
	Processor string
	File      string
	Line      int
	Msg       string
}

// ResolveStage1 reads and resolves the blueprint tree without executing
// anything: file routing (path- and content-based), template resolution, and
// strict decode per blueprint type. Problems become Diagnostics rather than
// errors — one bad file must not hide the rest of the tree from a validator
// or a progress display. The returned error is reserved for the tree being
// unreadable at all.
func ResolveStage1(initConfig *types.InitConfig) (*Plan, error) {
	plan := &Plan{
		Init:  initConfig,
		Files: map[string][]ResolvedFile{},
	}

	order, err := GetBlueprintRunOrder(initConfig)
	if err != nil {
		return nil, fmt.Errorf("error getting blueprint run order: %w", err)
	}
	plan.Order = order

	location := initConfig.Init.Location
	fileOrder, err := GetBlueprintFileOrder(location, initConfig.Init.Order, initConfig.Init.RunOnlyListed, initConfig)
	if err != nil {
		return nil, fmt.Errorf("error getting blueprint file order: %w", err)
	}

	for processor, files := range fileOrder {
		for _, rel := range files {
			path := location + string(os.PathSeparator) + rel

			format, formatErr := helpers.FormatForPath(path)
			if formatErr != nil {
				plan.Diags = append(plan.Diags, Diagnostic{Severity: SeverityError, Processor: processor, File: path, Msg: formatErr.Error()})
				continue
			}

			raw, readErr := os.ReadFile(path) // #nosec G304 -- operator's own blueprint tree
			if readErr != nil {
				plan.Diags = append(plan.Diags, Diagnostic{Severity: SeverityError, Processor: processor, File: path, Msg: readErr.Error()})
				continue
			}

			// Strict for the fixed namespaces, lenient only for UserDefined —
			// the same contract validate enforces.
			for _, ref := range helpers.UnknownTemplateReferences(raw, initConfig.Variables) {
				plan.Diags = append(plan.Diags, Diagnostic{
					Severity: SeverityError, Processor: processor, File: path,
					Msg: fmt.Sprintf("Template references %s, which does not exist", ref),
				})
			}

			resolved, resolveErr := helpers.ResolveTemplateForValidation(raw, initConfig.Variables)
			if resolveErr != nil {
				plan.Diags = append(plan.Diags, Diagnostic{Severity: SeverityError, Processor: processor, File: path, Msg: resolveErr.Error()})
				continue
			}

			// A multi-type file is cut down to this processor's sections,
			// exactly as the run loop does before dispatch; single-type files
			// pass through and keep strict decode's typo protection.
			subset, subsetFormat, subsetErr := subsetForProcessor(resolved, format, processor)
			if subsetErr != nil {
				plan.Diags = append(plan.Diags, Diagnostic{Severity: SeverityError, Processor: processor, File: path, Msg: subsetErr.Error()})
				continue
			}

			plan.Files[processor] = append(plan.Files[processor], ResolvedFile{
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
