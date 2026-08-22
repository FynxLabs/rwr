package helpers

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
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
// probeClient re-validates redirects like every other fetch rwr makes: an
// init source resolves by following the server's answer, and a 302 to plain
// http would otherwise decide where the blueprint tree comes from over an
// unauthenticated connection.
var probeClient = system.NewHTTPClient(10 * time.Second)

// ResolveInitSource turns whatever the operator gave as the init source into
// the one form Initialize consumes: an existing local file path, or an
// https:// URL to download. One documented precedence, top wins:
//
//  1. "" - probe the current directory for init.{yaml,yml,json,toml}.
//  2. An https:// URL. A GitHub /blob/ page URL is rewritten to its
//     raw.githubusercontent.com form.
//  3. An http:// URL - refused: the init file drives everything rwr runs.
//  4. An existing local file.
//  5. An existing local directory - probed like the current directory.
//  6. A GitHub shorthand: owner/repo, owner/repo@ref, owner/repo/path/to/file
//     (with optional @ref). Resolved against raw.githubusercontent.com; a bare
//     owner/repo probes the repository root for the init file names, @ref
//     defaults to HEAD (the default branch).
//
// The local-before-shorthand order means a directory literally named
// "owner/repo" keeps working - existence on disk wins over interpretation.
func ResolveInitSource(ref string) (string, error) {
	if ref == "" {
		return probeInitDir(".")
	}

	if strings.HasPrefix(ref, "https://") {
		// A GitHub repository page is a container, not an init file. Passing it
		// through made Initialize download GitHub's HTML and parse the page's own
		// {{ ... }} expressions as blueprint templates. Resolve repository roots
		// by cloning the tree: even a root init file may refer to sibling files,
		// and downloading that init alone would turn its relative location into a
		// temporary directory that disappears after initialization.
		if shorthand, ok := githubRepositoryShorthand(ref); ok {
			parts := strings.SplitN(shorthand, "/", 2)
			return cloneManifestRepo(parts[0], parts[1])
		}
		// A raw GitHub init still belongs to a repository. Clone the tree so
		// relative blueprint paths resolve beside the init instead of under the
		// short-lived download directory.
		normalized, err := rewriteBlobURL(ref)
		if err != nil {
			return "", err
		}
		if owner, repo, ok := rawGitHubRepository(normalized); ok {
			return cloneManifestRepo(owner, repo)
		}
		return normalized, nil
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

func rawGitHubRepository(ref string) (string, string, bool) {
	u, err := url.Parse(ref)
	if err != nil || !strings.EqualFold(u.Hostname(), "raw.githubusercontent.com") {
		return "", "", false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 {
		return "", "", false
	}
	base := strings.TrimSuffix(parts[len(parts)-1], filepath.Ext(parts[len(parts)-1]))
	if base != "init" && base != "manifest" {
		return "", "", false
	}
	return parts[0], strings.TrimSuffix(parts[1], ".git"), true
}

// githubRepositoryShorthand recognizes only a GitHub repository root. Blob
// URLs remain individual files, and tree/issue/action URLs must not silently
// acquire repository-root semantics.
func githubRepositoryShorthand(ref string) (string, bool) {
	u, err := url.Parse(ref)
	if err != nil || !strings.EqualFold(u.Hostname(), "github.com") || u.RawQuery != "" || u.Fragment != "" {
		return "", false
	}
	parts := strings.Split(strings.Trim(u.EscapedPath(), "/"), "/")
	if len(parts) != 2 {
		return "", false
	}
	owner, err := url.PathUnescape(parts[0])
	if err != nil {
		return "", false
	}
	repo, err := url.PathUnescape(parts[1])
	if err != nil {
		return "", false
	}
	repo = strings.TrimSuffix(repo, ".git")
	shorthand := owner + "/" + repo
	return shorthand, shorthandPattern.MatchString(shorthand)
}

// probeInitDir returns the first init file name that exists in dir. A
// directory with no init file but a manifest at its root is a
// multi-configuration repo: the manifest path is returned and the caller
// runs entry selection (the manifest path only activates when there is no
// init file, so plain trees behave exactly as before).
func probeInitDir(dir string) (string, error) {
	for _, name := range initFileNames {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			log.Debugf("Found init file: %s", candidate)
			return candidate, nil
		}
	}
	if manifest := FindManifest(dir); manifest != "" {
		log.Debugf("Found manifest: %s", manifest)
		return manifest, nil
	}
	return "", fmt.Errorf("no init file (%s) or manifest found in %s", strings.Join(initFileNames, ", "), dir)
}

// probeRepositoryDir adds the conventional per-OS directory used by older
// multi-machine repositories that predate manifests.
func probeRepositoryDir(dir string) (string, error) {
	if resolved, err := probeInitDir(dir); err == nil {
		return resolved, nil
	}
	var osDirs []string
	switch runtime.GOOS {
	case "darwin":
		osDirs = []string{"macOS", "macos", "darwin"}
	case "windows":
		osDirs = []string{"Windows", "windows"}
	case "linux":
		osDirs = []string{"Linux", "linux"}
	}
	for _, osDir := range osDirs {
		candidate := filepath.Join(dir, osDir)
		if resolved, err := probeInitDir(candidate); err == nil {
			log.Infof("Using conventional %s blueprint directory", osDir)
			return resolved, nil
		}
	}
	return probeInitDir(dir)
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

	// No init file at the repo root: a manifest there marks a
	// multi-configuration repo, which needs the whole tree locally - entry
	// init files reference siblings - so it is cloned and probed as a local
	// root. Only reached when no init file exists, same as the local rule.
	for _, name := range CandidateFilenames("manifest") {
		candidate := base + "/" + name
		resp, err := probeClient.Head(candidate)
		if err != nil {
			return "", fmt.Errorf("probing %s: %w", candidate, err)
		}
		if cerr := resp.Body.Close(); cerr != nil {
			log.Debugf("closing probe response for %s: %v", candidate, cerr)
		}
		if resp.StatusCode == http.StatusOK {
			log.Debugf("Shorthand %s@%s is a manifest repo", ref, gitRef)
			return cloneManifestRepo(segments[0], segments[1])
		}
	}
	return "", fmt.Errorf("no init file or manifest found in %s at %s: tried %s", ref, gitRef, strings.Join(tried, ", "))
}

// cloneManifestRepo is a seam so tests can fake the clone.
var cloneManifestRepo = func(owner, repo string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error finding home directory: %w", err)
	}
	target := filepath.Join(homeDir, ".config", "rwr", "blueprints", owner+"-"+repo)

	if _, err := os.Stat(filepath.Join(target, ".git")); err != nil {
		opts := types.GitOptions{
			URL:    "https://github.com/" + owner + "/" + repo + ".git",
			Target: target,
		}
		if err := HandleGitClone(opts, &types.InitConfig{}); err != nil {
			return "", fmt.Errorf("error cloning manifest repo %s/%s: %w", owner, repo, err)
		}
	}
	return probeRepositoryDir(target)
}
