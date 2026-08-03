// Package status answers "does this machine match the tree" without
// applying anything. Queries are read-only and never elevate: a status
// check that mutates, or asks for root, is a status check nobody trusts.
package status

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	"charm.land/log/v2"
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
		installed = listInstalled(provider)
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

// listInstalled runs the provider's list command read-only and returns the
// set of first-column package names, nil when the command fails. Some list
// values name a different binary entirely (apt's is `dpkg
// --get-selections`); the first field decides what actually runs.
func listInstalled(provider *types.Provider) map[string]bool {
	fields := strings.Fields(provider.Commands.List)
	bin := provider.BinPath
	args := fields
	if len(fields) > 0 {
		if path, err := exec.LookPath(fields[0]); err == nil && fields[0] != provider.Name {
			bin, args = path, fields[1:]
		}
	}
	out, err := exec.Command(bin, args...).Output() // #nosec G204 -- provider definitions are rwr's own vetted data; list verbs are read-only
	if err != nil {
		log.Debugf("status: %s list failed: %v", provider.Name, err)
		return nil
	}
	installed := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := fields[0]
		// dpkg emits "name:arch"; strip the qualifier so identity matches.
		if i := strings.IndexByte(name, ':'); i > 0 {
			name = name[:i]
		}
		installed[name] = true
	}
	return installed
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

// ServiceState queries a service read-only on the current platform; other
// platforms and query failures are Unknown.
func ServiceState(name string) Presence {
	if name == "" || runtime.GOOS != "linux" {
		return Unknown
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		return Unknown
	}
	// is-enabled exits non-zero for disabled AND for unknown units; the
	// output tells them apart.
	out, _ := exec.Command("systemctl", "is-enabled", name).CombinedOutput() //nolint:errcheck // exit code is part of the answer
	answer := strings.TrimSpace(string(out))
	switch answer {
	case "enabled", "enabled-runtime", "static", "alias", "linked":
		return Present
	case "disabled":
		return Absent
	}
	return Unknown
}
