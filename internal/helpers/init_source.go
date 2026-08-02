package helpers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"charm.land/log/v2"
)

// initFileNames are the file names probed, in order, when the init source is a
// directory (local or a repository root). Derived from the format registry so
// a new format is discoverable without touching this file.
var initFileNames = CandidateFilenames("init")

// shorthandPattern matches GitHub repository shorthands: owner/repo, with an
// optional /path/to/file and an optional @ref suffix. Owner and repo are a
// single path segment each; anything with a scheme or an absolute path never
// reaches this check.
var shorthandPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+(/[^@]+)?(@[A-Za-z0-9_./-]+)?$`)

// probeClient answers "does this raw URL exist" during shorthand resolution.
// A var so tests can point probes at their own server without waiting 10s.
var probeClient = &http.Client{Timeout: 10 * time.Second}

// ResolveInitSource turns whatever the operator gave as the init source into
// the one form Initialize consumes: an existing local file path, or an
// https:// URL to download. One documented precedence, top wins:
//
//  1. "" — probe the current directory for init.{yaml,yml,json,toml}.
//  2. An https:// URL. A GitHub /blob/ page URL is rewritten to its
//     raw.githubusercontent.com form.
//  3. An http:// URL — refused: the init file drives everything rwr runs.
//  4. An existing local file.
//  5. An existing local directory — probed like the current directory.
//  6. A GitHub shorthand: owner/repo, owner/repo@ref, owner/repo/path/to/file
//     (with optional @ref). Resolved against raw.githubusercontent.com; a bare
//     owner/repo probes the repository root for the init file names, @ref
//     defaults to HEAD (the default branch).
//
// The local-before-shorthand order means a directory literally named
// "owner/repo" keeps working — existence on disk wins over interpretation.
func ResolveInitSource(ref string) (string, error) {
	if ref == "" {
		return probeInitDir(".")
	}

	if strings.HasPrefix(ref, "https://") {
		return rewriteBlobURL(ref)
	}
	if strings.HasPrefix(ref, "http://") {
		return "", fmt.Errorf("refusing to fetch the init file over http://: it is served in cleartext and drives everything rwr runs; use https:// instead (%s)", ref)
	}
	if strings.Contains(ref, "://") {
		return "", fmt.Errorf("unsupported init source scheme in %q: use a local path, an https:// URL, or a GitHub owner/repo shorthand", ref)
	}

	if info, err := os.Stat(ref); err == nil {
		if info.IsDir() {
			return probeInitDir(ref)
		}
		return ref, nil
	}

	if shorthandPattern.MatchString(ref) {
		return resolveShorthand(ref)
	}

	return "", fmt.Errorf("init source %q is not an existing path, an https:// URL, or an owner/repo shorthand", ref)
}

// probeInitDir returns the first init file name that exists in dir.
func probeInitDir(dir string) (string, error) {
	for _, name := range initFileNames {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			log.Debugf("Found init file: %s", candidate)
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no init file (%s) found in %s", strings.Join(initFileNames, ", "), dir)
}

// rewriteBlobURL turns a GitHub /blob/ page URL into its raw content URL and
// passes every other https URL through. The old in-place rewrite indexed the
// split unchecked, so a malformed URL containing "/blob/" was an index panic.
func rewriteBlobURL(url string) (string, error) {
	if !strings.Contains(url, "/blob/") {
		return url, nil
	}
	parts := strings.Split(url, "/")
	blobSplit := strings.Split(url, "/blob/")
	if len(parts) < 5 || len(blobSplit) != 2 || blobSplit[1] == "" {
		return "", fmt.Errorf("cannot rewrite %q as a GitHub blob URL: expected https://github.com/<owner>/<repo>/blob/<ref>/<path>", url)
	}
	raw := "https://raw.githubusercontent.com/" + parts[3] + "/" + parts[4] + "/" + blobSplit[1]
	log.Debugf("Rewrote GitHub blob URL to raw: %s", raw)
	return raw, nil
}

// rawGitHubBase is a var so tests can resolve shorthands against their own
// server.
var rawGitHubBase = "https://raw.githubusercontent.com"

// resolveShorthand resolves owner/repo[/path][@ref] to a raw content URL.
func resolveShorthand(ref string) (string, error) {
	gitRef := "HEAD"
	if at := strings.LastIndex(ref, "@"); at >= 0 {
		gitRef = ref[at+1:]
		ref = ref[:at]
	}

	segments := strings.SplitN(ref, "/", 3)
	base := fmt.Sprintf("%s/%s/%s/%s", rawGitHubBase, segments[0], segments[1], gitRef)

	// An explicit file path needs no probing.
	if len(segments) == 3 {
		return base + "/" + segments[2], nil
	}

	// A bare owner/repo names the repository root: probe the init file names,
	// the same contract as pointing rwr at a local directory.
	var tried []string
	for _, name := range initFileNames {
		candidate := base + "/" + name
		resp, err := probeClient.Head(candidate)
		if err != nil {
			return "", fmt.Errorf("probing %s: %w", candidate, err)
		}
		if cerr := resp.Body.Close(); cerr != nil {
			log.Debugf("closing probe response for %s: %v", candidate, cerr)
		}
		if resp.StatusCode == http.StatusOK {
			log.Debugf("Resolved shorthand %s@%s to %s", ref, gitRef, candidate)
			return candidate, nil
		}
		tried = append(tried, name)
	}
	return "", fmt.Errorf("no init file found in %s at %s: tried %s", ref, gitRef, strings.Join(tried, ", "))
}
