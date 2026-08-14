package system

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Every hop is re-validated, not just the URL the caller passed in.
//
// Go strips Authorization when a redirect crosses to a different host, but not
// when it merely drops https for http on the same one. A bare client following
// such a hop puts whatever the request carried - the GitHub token rwr sends to
// api.github.com, the device code it exchanges for one - on the wire in
// cleartext.
func TestHTTPClientRefusesADowngradeRedirect(t *testing.T) {
	t.Parallel()

	// Stands in for a host answering 302 to plain http.
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://example.invalid/token", http.StatusFound)
	}))
	defer origin.Close()

	client := NewHTTPClient(5 * time.Second)
	resp, err := client.Get(origin.URL) //nolint:bodyclose // the request must not succeed
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("a redirect to plain http was followed")
	}
	if !strings.Contains(err.Error(), "refusing to download") {
		t.Errorf("refused for the wrong reason: %v", err)
	}
}

// A redirect that stays on https is ordinary and must still work, or every
// GitHub API call that redirects breaks.
func TestHTTPClientFollowsAnHTTPSRedirect(t *testing.T) {
	t.Parallel()

	final := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer final.Close()

	origin := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer origin.Close()

	client := NewHTTPClient(5 * time.Second)
	// The test servers use self-signed certs; trust them for this exchange.
	client.Transport = origin.Client().Transport

	resp, err := client.Get(origin.URL)
	if err != nil {
		t.Fatalf("an https redirect was refused: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

// The redirect chain is bounded, so a server looping forever fails the step
// instead of hanging the run.
func TestHTTPClientStopsAfterTenRedirects(t *testing.T) {
	t.Parallel()

	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, server.URL+"/again", http.StatusFound)
	}))
	defer server.Close()

	client := NewHTTPClient(5 * time.Second)
	client.Transport = server.Client().Transport

	resp, err := client.Get(server.URL) //nolint:bodyclose // the request must not succeed
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("an endless redirect chain was followed")
	}
	if !strings.Contains(err.Error(), "stopped after 10 redirects") {
		t.Errorf("stopped for the wrong reason: %v", err)
	}
}

// DownloadClient keeps its own long timeout while sharing the policy.
func TestDownloadClientKeepsItsTimeout(t *testing.T) {
	t.Parallel()

	if DownloadClient.Timeout != 15*time.Minute {
		t.Errorf("DownloadClient timeout = %v, want 15m", DownloadClient.Timeout)
	}
	if DownloadClient.CheckRedirect == nil {
		t.Error("DownloadClient lost its redirect policy")
	}
}
