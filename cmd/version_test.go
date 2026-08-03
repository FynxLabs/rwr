package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.2.3", "1.2.3", 0},
		{"v1.2.3", "1.2.3", 0},
		{"1.2.3", "1.2.4", -1},
		{"1.3.0", "1.2.9", 1},
		{"1.2", "1.2.0", 0},
		{"1.2.3", "2.0.0", -1},
		// A nightly built from master sorts before the release it precedes.
		{"1.2.4-master-abc1234", "1.2.4", -1},
		{"1.2.4", "1.2.4-rc1", 1},
		// Unparseable versions never produce a notice.
		{"dev", "1.2.3", 0},
		{"1.2.3", "not-a-version", 0},
	}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestVersionCommandPrintsBuildInfo(t *testing.T) {
	original := buildInfo
	t.Cleanup(func() { buildInfo = original })

	buildInfo = BuildInfo{
		Version:   "1.2.3",
		Commit:    "abc1234",
		Date:      "2026-01-01T00:00:00Z",
		BuiltBy:   "goreleaser",
		TreeState: "false",
	}

	var out bytes.Buffer
	versionCmd := newVersionCmd(NewAppConfig())
	versionCmd.SetOut(&out)
	if err := versionCmd.RunE(versionCmd, nil); err != nil {
		t.Fatalf("version command: %v", err)
	}

	got := out.String()
	for _, want := range []string{"rwr 1.2.3", "abc1234", "goreleaser"} {
		if !strings.Contains(got, want) {
			t.Errorf("version output missing %q:\n%s", want, got)
		}
	}
}

func TestVersionCheckSkipsDevBuildsAndFlag(t *testing.T) {
	originalInfo, originalURL := buildInfo, latestReleaseURL
	t.Cleanup(func() {
		buildInfo, latestReleaseURL = originalInfo, originalURL
	})
	app := NewAppConfig()

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, _ = w.Write([]byte(`{"tag_name":"v99.0.0"}`))
	}))
	t.Cleanup(server.Close)
	latestReleaseURL = server.URL

	app.SkipVersionCheck = false
	// "dev", and the pseudo-versions a local `go build` embeds, are all dev
	// builds: none of them may reach the network.
	for _, v := range []string{"dev", "0.5.2-0.20260801212055-24872aab978e", "0.5.2+dirty"} {
		buildInfo = BuildInfo{Version: v}
		checkForNewVersion(app)
		if called {
			t.Errorf("version check contacted the network for dev build %q", v)
		}
	}

	buildInfo = BuildInfo{Version: "1.0.0"}
	app.SkipVersionCheck = true
	checkForNewVersion(app)
	if called {
		t.Error("version check ran despite --skip-version-check")
	}

	app.SkipVersionCheck = false
	checkForNewVersion(app)
	if !called {
		t.Error("version check did not run for a release build")
	}
}

func TestVersionCheckIgnoresFailures(t *testing.T) {
	originalInfo, originalURL := buildInfo, latestReleaseURL
	t.Cleanup(func() {
		buildInfo, latestReleaseURL = originalInfo, originalURL
	})
	app := NewAppConfig()

	buildInfo = BuildInfo{Version: "1.0.0"}
	app.SkipVersionCheck = false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(server.Close)

	// Each of these must return without panicking or failing the run.
	latestReleaseURL = server.URL
	checkForNewVersion(app)
	latestReleaseURL = "http://127.0.0.1:1/nope"
	checkForNewVersion(app)
	latestReleaseURL = "://malformed"
	checkForNewVersion(app)
}
