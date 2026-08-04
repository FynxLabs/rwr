// Package scan answers "what did the operator put on this machine" - the
// shared harvest layer under `rwr diff` and `rwr capture`. Scanners are
// read-only and never elevate: they execute only provider list verbs and
// read files and unit states.
package scan

import (
	"os/exec"
	"sort"
	"strings"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/types"
)

// PackageResult is one provider's operator-chosen package set. Unfiltered
// marks a provider without a list_explicit verb: the full list, dependency
// noise included, because pretending otherwise would be a lie.
type PackageResult struct {
	Provider   string
	Names      []string
	Unfiltered bool
}

// Packages scans every given provider, preferring the explicitly-installed
// query over the full list.
func Packages(providers map[string]*types.Provider) []PackageResult {
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	sort.Strings(names)

	var results []PackageResult
	for _, name := range names {
		provider := providers[name]
		command := provider.Commands.ListExplicit
		unfiltered := false
		if command == "" {
			command = provider.Commands.List
			unfiltered = true
		}
		if command == "" {
			continue
		}
		packages := RunListCommand(provider, command)
		if packages == nil {
			continue
		}
		results = append(results, PackageResult{Provider: name, Names: packages, Unfiltered: unfiltered})
	}
	return results
}

// RunListCommand executes a provider's list-class verb read-only and returns
// the first-column names, nil on failure. Some verbs name a different binary
// entirely (apt's explicit query is `apt-mark showmanual`); the first field
// decides what actually runs - the same dispatch the status querier uses.
func RunListCommand(provider *types.Provider, command string) []string {
	fields := strings.Fields(command)
	bin := provider.BinPath
	args := fields
	if len(fields) > 0 {
		if path, err := exec.LookPath(fields[0]); err == nil && fields[0] != provider.Name {
			bin, args = path, fields[1:]
		}
	}
	out, err := exec.Command(bin, args...).Output() // #nosec G204 -- provider definitions are rwr's own vetted data; list verbs are read-only
	if err != nil {
		log.Debugf("scan: %s list failed: %v", provider.Name, err)
		return nil
	}
	var names []string
	seen := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		// Indented lines are detail under an entry (cargo lists each crate's
		// binaries indented), not entries themselves.
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := fields[0]
		// snap (and friends) print a header row; "Name" first is a header,
		// not a package.
		if name == "Name" {
			continue
		}
		// dpkg emits "name:arch"; cargo emits "name v1.2.3:"; strip both.
		if i := strings.IndexByte(name, ':'); i > 0 {
			name = name[:i]
		}
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	return names
}
