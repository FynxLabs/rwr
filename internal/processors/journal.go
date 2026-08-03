package processors

import (
	"sync"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/state"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
)

// The run journal mirrors the failure ledger's seam: package state the
// per-item emission points write through, opened by All()/RunBootstrap and
// finalized when the run ends. A nil journal (dry-run, or no config dir)
// swallows appends.
var (
	journalMu sync.Mutex
	journal   *state.Writer
)

// openJournal starts a run record; a dry-run opens nothing by design.
func openJournal(location string) {
	writer, err := state.NewWriter(viper.GetString("rwr.configdir"), location, system.IsDryRun())
	if err != nil {
		// The journal observes the run; failing to open it must not stop the
		// run itself.
		log.Warnf("Run journal unavailable: %v", err)
		return
	}
	journalMu.Lock()
	journal = writer
	journalMu.Unlock()
}

// closeJournal finalizes the record and points `latest` at it.
func closeJournal() {
	journalMu.Lock()
	writer := journal
	journal = nil
	journalMu.Unlock()
	if err := writer.Finalize(); err != nil {
		log.Warnf("Finalizing the run journal: %v", err)
	}
}

// fileJournalIdentity resolves a file entry's on-disk destination and its
// post-apply content hash — what a later uninstall needs for a hash-guarded
// delete. Best effort: the identity observes the apply, it never fails it.
func fileJournalIdentity(file types.File, blueprintDir string) map[string]string {
	_, target, err := determineSourceAndTargetPaths(file, blueprintDir)
	if err != nil || target == "" {
		return nil
	}
	identity := map[string]string{"dest": target}
	if sum, hashErr := system.HashFileSHA256(target); hashErr == nil {
		identity["sha256"] = sum
	}
	return identity
}

// journalAppend records one applied unit. Planned (dry-run) and skipped
// items are not applies and leave no entry.
func journalAppend(processor, provider, name, action string, status types.Status, detail string, identity map[string]string) {
	journalMu.Lock()
	writer := journal
	journalMu.Unlock()
	if writer == nil {
		return
	}
	switch status {
	case types.StatusPlanned, types.StatusSkipped:
		return
	case types.StatusOK, types.StatusFailed, types.StatusUnknown, types.StatusPresent:
	}

	merged := map[string]string{}
	for key, value := range identity {
		merged[key] = value
	}
	if provider != "" {
		merged["provider"] = provider
	}
	if name != "" {
		merged["name"] = name
	}
	writer.Append(state.Entry{
		Processor: processor,
		Action:    action,
		Identity:  merged,
		Detail:    detail,
		Outcome:   string(status),
		OK:        status == types.StatusOK,
	})
}
