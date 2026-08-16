package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"runtime"
	godebug "runtime/debug"
	"strconv"
	"strings"
	"time"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// BuildInfo carries the build metadata injected at link time by goreleaser.
// main owns the ldflag variables; it hands them here via SetVersionInfo.
type BuildInfo struct {
	Version   string
	Commit    string
	Date      string
	BuiltBy   string
	TreeState string
}

// buildInfo holds the build metadata for the running binary. It defaults to a
// dev build so that `rwr version` still says something useful when the binary
// was built with a plain `go build`.
var buildInfo = BuildInfo{Version: "dev"}

// latestReleaseURL is the GitHub API endpoint consulted by the startup version
// check. It is a variable so tests can point it at a local server.
var latestReleaseURL = "https://api.github.com/repos/fynxlabs/rwr/releases/latest"

// SetVersionInfo records the build metadata injected at link time. Empty
// fields fall back to Go's embedded build info, which is what `go install`
// builds have instead of ldflags. NewRootCmd wires it into cobra so that
// `rwr --version` works alongside `rwr version` - buildInfo is link-time
// constant data, set once from main before any tree is built.
func SetVersionInfo(info BuildInfo) {
	buildInfo = fillFromBuildInfo(info)
}

// fillFromBuildInfo backfills missing build metadata from the module and VCS
// information the Go toolchain embeds in the binary.
func fillFromBuildInfo(info BuildInfo) BuildInfo {
	if info.Version == "" {
		info.Version = "dev"
	}

	bi, ok := godebug.ReadBuildInfo()
	if !ok {
		return info
	}

	if info.Version == "dev" && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		info.Version = strings.TrimPrefix(bi.Main.Version, "v")
	}

	for _, setting := range bi.Settings {
		switch setting.Key {
		case "vcs.revision":
			if info.Commit == "" {
				info.Commit = setting.Value
			}
		case "vcs.time":
			if info.Date == "" {
				info.Date = setting.Value
			}
		case "vcs.modified":
			if info.TreeState == "" && setting.Value == "true" {
				info.TreeState = "dirty"
			}
		}
	}

	if info.BuiltBy == "" {
		info.BuiltBy = "go"
	}
	return info
}

// pseudoVersion matches the module pseudo-versions the Go toolchain synthesizes
// for an untagged build, e.g. "0.5.2-0.20260801212055-24872aab978e".
var pseudoVersion = regexp.MustCompile(`\d{14}-[0-9a-f]{12}`)

// isDevBuild reports whether this binary carries no real release version. Dev
// builds skip the update check: there is nothing meaningful to compare against,
// and a plain `go build` in a checkout must never reach out to the network.
func isDevBuild() bool {
	v := buildInfo.Version
	return v == "" ||
		v == "dev" ||
		strings.Contains(v, "+dirty") ||
		pseudoVersion.MatchString(v)
}

func newVersionCmd(app *AppConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version, commit, and build metadata of this binary",
		Long: `Print build information for this rwr binary.

Release and nightly binaries carry the commit they were built from, so a binary
on disk can be traced back to the source that produced it.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			var b strings.Builder
			fmt.Fprintf(&b, "rwr %s\n", buildInfo.Version)
			if buildInfo.Commit != "" {
				fmt.Fprintf(&b, "commit:     %s\n", buildInfo.Commit)
			}
			if buildInfo.Date != "" {
				fmt.Fprintf(&b, "built:      %s\n", buildInfo.Date)
			}
			if buildInfo.BuiltBy != "" {
				fmt.Fprintf(&b, "built by:   %s\n", buildInfo.BuiltBy)
			}
			if buildInfo.TreeState != "" {
				fmt.Fprintf(&b, "tree state: %s\n", buildInfo.TreeState)
			}
			fmt.Fprintf(&b, "go:         %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)

			_, err := io.WriteString(out, b.String())
			return err
		},
	}
}

// checkForNewVersion asks GitHub for the latest release and prints a one-line
// notice to stderr when a newer version exists. It is strictly advisory: every
// failure path is a debug log and a silent return, so a missing network, a rate
// limit, or a malformed response can never affect the run.
func checkForNewVersion(app *AppConfig) {
	// Read through viper, not the flag variable: the flag is bound to
	// rwr.skipVersionCheck, but that binding is one-way. Checking the variable
	// alone meant the config key `rwr config --create` writes, and its env form,
	// were both inert - only the command-line flag did anything.
	if app.SkipVersionCheck || viper.GetBool("rwr.skipVersionCheck") {
		log.Debugf("Version check skipped (rwr.skipVersionCheck)")
		return
	}
	if isDevBuild() {
		log.Debugf("Version check skipped for dev build")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		log.Debugf("Version check skipped: %v", err)
		return
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "rwr/"+buildInfo.Version)

	resp, err := system.NewHTTPClient(2 * time.Second).Do(req)
	if err != nil {
		log.Debugf("Version check failed: %v", err)
		return
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Debugf("Version check: closing response body: %v", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		log.Debugf("Version check failed: unexpected status %s", resp.Status)
		return
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		log.Debugf("Version check failed: %v", err)
		return
	}

	latest := strings.TrimPrefix(strings.TrimSpace(release.TagName), "v")
	if latest == "" {
		log.Debugf("Version check failed: empty tag in response")
		return
	}

	if compareVersions(buildInfo.Version, latest) < 0 {
		fmt.Fprintf(os.Stderr, "rwr: a newer version is available: %s (you have %s)\n", latest, buildInfo.Version)
		fmt.Fprintf(os.Stderr, "rwr: https://github.com/fynxlabs/rwr/releases/latest (silence with --skip-version-check)\n")
	}
}

// compareVersions compares two dotted numeric versions, returning -1, 0 or 1.
// Non-numeric suffixes (a "-master-abc1234" nightly, an "-rc1" prerelease) sort
// before the same version without one, matching semver's ordering closely enough
// for an advisory notice. Anything unparseable compares as equal, so a version
// string we do not understand never produces a spurious notice.
func compareVersions(a, b string) int {
	aNums, aPre := splitVersion(a)
	bNums, bPre := splitVersion(b)
	if aNums == nil || bNums == nil {
		return 0
	}

	for i := 0; i < len(aNums) || i < len(bNums); i++ {
		var an, bn int
		if i < len(aNums) {
			an = aNums[i]
		}
		if i < len(bNums) {
			bn = bNums[i]
		}
		if an != bn {
			if an < bn {
				return -1
			}
			return 1
		}
	}

	switch {
	case aPre == bPre:
		return 0
	case aPre != "" && bPre == "":
		return -1
	case aPre == "" && bPre != "":
		return 1
	case aPre < bPre:
		return -1
	default:
		return 1
	}
}

// splitVersion splits "1.2.3-rc1" into ([1 2 3], "rc1"). It returns a nil slice
// when the leading component is not numeric.
func splitVersion(v string) ([]int, string) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")

	pre := ""
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		pre = v[i+1:]
		v = v[:i]
	}

	var nums []int
	for _, part := range strings.Split(v, ".") {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, pre
		}
		nums = append(nums, n)
	}
	return nums, pre
}
