// Package status answers "does this machine match the tree" without
// applying anything. Queries are read-only and never elevate: a status
// check that mutates, or asks for root, is a status check nobody trusts.
package status

import (
	"errors"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/fynxlabs/rwr/internal/scan"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// Presence is one queried actual state.
type Presence string

const (
	Present  Presence = "present"
	Absent   Presence = "absent"
	Modified Presence = "modified"
	Unknown  Presence = "unknown"
)

// Querier caches per-provider package listings for one status run.
type Querier struct {
	packageLists map[string]map[string]bool // provider → installed set; nil = list failed
}

func NewQuerier() *Querier {
	return &Querier{packageLists: map[string]map[string]bool{}}
}

// PackagePresent reports whether a provider's list output names the package.
// A provider without a usable list command yields Unknown, never a guess.
func (q *Querier) PackagePresent(provider *types.Provider, name string) Presence {
	if provider == nil || provider.Commands.List == "" {
		return Unknown
	}
	installed, cached := q.packageLists[provider.Name]
	if !cached {
		installed = toSet(scan.RunListCommand(provider, provider.Commands.List))
		q.packageLists[provider.Name] = installed
	}
	if installed == nil {
		return Unknown
	}
	if installed[name] {
		return Present
	}
	return Absent
}

// toSet indexes list output for presence checks; nil stays nil (the list
// failed and the answer is unknown, not empty).
func toSet(names []string) map[string]bool {
	if names == nil {
		return nil
	}
	set := make(map[string]bool, len(names))
	for _, name := range names {
		set[name] = true
	}
	return set
}

// FileState compares a recorded file against disk: missing, still matching
// its recorded hash, or modified since the apply.
func FileState(dest, recordedSHA string) Presence {
	if dest == "" {
		return Unknown
	}
	if _, err := os.Stat(dest); err != nil {
		return Absent
	}
	if recordedSHA == "" {
		return Present
	}
	sum, err := system.HashFileSHA256(dest)
	if err != nil {
		return Unknown
	}
	if strings.EqualFold(sum, recordedSHA) {
		return Present
	}
	return Modified
}

// PathPresent is the existence check fonts and git checkouts get.
func PathPresent(path string) Presence {
	if path == "" {
		return Unknown
	}
	if _, err := os.Stat(path); err != nil {
		return Absent
	}
	return Present
}

// validUnitName accepts systemd unit-name characters only. The query is
// argv-exec'd so a shell never sees the name, but a name starting with "-"
// would be read as a systemctl option - refused here.
var validUnitName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.@:\\-]*$`)

// ServiceState queries a service read-only on the current platform; other
// platforms and query failures are Unknown.
func ServiceState(name string) Presence {
	if name == "" || !validUnitName.MatchString(name) {
		return Unknown
	}
	switch runtime.GOOS {
	case types.OSLinux:
		return systemdState(name)
	case types.OSDarwin:
		return launchdState(name)
	}
	return Unknown
}

func systemdState(name string) Presence {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return Unknown
	}
	// is-enabled exits non-zero for disabled AND for unknown units; the
	// output tells them apart.
	query := exec.Command("systemctl", "is-enabled", name) // #nosec G204 -- argv-exec'd read-only query; name validated against the unit-name pattern above
	out, _ := query.CombinedOutput()                       //nolint:errcheck // exit code is part of the answer
	answer := strings.TrimSpace(string(out))
	switch answer {
	case "enabled", "enabled-runtime", "static", "alias", "linked":
		return Present
	case "disabled":
		return Absent
	}
	return Unknown
}

// launchdState asks launchctl whether a label is loaded.
//
// `launchctl list <label>` prints the job's dictionary and exits zero when the
// label is loaded, non-zero when it is not. That is the same verb the services
// processor's own "status" action uses, so status and apply agree about what
// "this service is here" means.
//
// Loaded, not enabled: launchd has no single is-enabled equivalent, and a job
// that is loaded is the state `rwr services` produces with `launchctl load`.
// Reporting Absent for a label launchctl does not know is honest; anything
// less certain stays Unknown.
func launchdState(name string) Presence {
	if _, err := exec.LookPath("launchctl"); err != nil {
		return Unknown
	}
	query := exec.Command("launchctl", "list", name) // #nosec G204 -- argv-exec'd read-only query; name validated against the unit-name pattern above
	if err := query.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return Absent
		}
		return Unknown
	}
	return Present
}
