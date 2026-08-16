package types

import "time"

// The Plan data model is shared by the resolver (internal/processors), the
// reporters (internal/reporting), and validate - it lives here so none of
// them import each other.

// ResolvedFile is one blueprint file after stage 1: routed, read, and
// template-resolved (multi-type files already subset per processor), so no
// consumer reads or renders it a second time.
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

// Status is a resource's outcome. `unknown` is required: rwr shells out and
// several providers do not report per-package results reliably - forcing
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

// Resource is one unit of work a run performs (a package, a file, a service).
type Resource struct {
	Processor string
	Provider  string // empty for files, services, git, scripts
	Name      string // "neovim", "~/.config/nvim/"
	// Location identifies resources whose name is not unique: the destination
	// of a file/directory or the target of a git checkout. Empty for resources
	// such as packages and services that are identified by name.
	Location string
	Action   string // install, copy, enable, clone
	Status   Status
	Detail   string
	Dur      time.Duration
}

// Severity classifies a diagnostic.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Diagnostic is one stage-1 finding, positioned when the position is known.
type Diagnostic struct {
	Severity  Severity
	Processor string
	File      string
	Line      int
	Msg       string
}

// StepError is one processor failure collected by a push-through run.
type StepError struct {
	Processor string
	Err       error
}

// Plan is what a run knows before it executes: the resolved blueprint tree.
// Stage 1 (parse, imports, schema, template resolution, diagnostics) needs
// only SetPaths and DetectOS, so `rwr validate` consumes it pre-init; stage 2
// (provider states, resource enumeration) runs after init and bootstrap,
// because bootstrap can install the package manager later blueprints depend
// on - detecting providers earlier produces wrong lanes.
type Plan struct {
	Init      *InitConfig
	Order     []string
	Files     map[string][]ResolvedFile
	Providers []ProviderState
	Resources []Resource
	Diags     []Diagnostic
}
