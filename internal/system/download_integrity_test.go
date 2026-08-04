package system

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Everything rwr downloads is installed with the operator's privileges -
// package-signing keys included - so plain http from a real host is refused.
// Loopback stays allowed for local mirrors and these tests.
func TestValidateDownloadURL(t *testing.T) {
	for _, tt := range []struct {
		url string
		ok  bool
	}{
		{"https://example.com/key.gpg", true},
		{"http://localhost:8080/x", true},
		{"http://127.0.0.1:9999/x", true},
		{"http://[::1]/x", true},
		{"http://example.com/key.gpg", false},
		{"http://10.0.0.5/x", false},
		{"ftp://example.com/x", false},
		{"file:///etc/passwd", false},
		{"://not a url", false},
	} {
		err := ValidateDownloadURL(tt.url)
		if tt.ok && err != nil {
			t.Errorf("ValidateDownloadURL(%q) = %v, want nil", tt.url, err)
		}
		if !tt.ok && err == nil {
			t.Errorf("ValidateDownloadURL(%q) = nil, want refusal", tt.url)
		}
	}
}

// The refusal happens before any request is made: a plain-http URL to a dead
// host errors on the scheme, not the network.
func TestDownloadFile_RefusesPlainHTTP(t *testing.T) {
	target := filepath.Join(t.TempDir(), "out")
	err := DownloadFile("http://example.invalid/key.gpg", target, false)
	if err == nil || !strings.Contains(err.Error(), "plain http") {
		t.Fatalf("err = %v, want a plain-http refusal", err)
	}
	if _, statErr := os.Stat(target); statErr == nil {
		t.Fatal("target file exists despite the refusal")
	}
}

// Validating only the initial URL is worthless if a redirect can hop to plain
// http: the default client follows it silently and the content is once again
// substitutable in transit. Every hop is re-validated by DownloadClient's
// CheckRedirect. The redirect target here is a non-loopback http URL - the
// refusal happens in CheckRedirect before any request is made to it.
func TestDownloadFile_RefusesRedirectToPlainHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://example.invalid/key.gpg", http.StatusFound)
	}))
	defer server.Close()

	target := filepath.Join(t.TempDir(), "out")
	err := DownloadFile(server.URL, target, false)
	if err == nil || !strings.Contains(err.Error(), "plain http") {
		t.Fatalf("err = %v, want a plain-http refusal on the redirect hop", err)
	}
	if _, statErr := os.Stat(target); statErr == nil {
		t.Fatal("target file exists despite the refusal")
	}
}

// A same-origin redirect to another validated URL still works.
func TestDownloadFile_FollowsValidatedRedirect(t *testing.T) {
	content := []byte("redirected content")
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()
	mux.HandleFunc("/moved", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, server.URL+"/real", http.StatusFound)
	})
	mux.HandleFunc("/real", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(content)
	})

	target := filepath.Join(t.TempDir(), "out")
	if err := DownloadFile(server.URL+"/moved", target, false); err != nil {
		t.Fatalf("DownloadFile: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != string(content) {
		t.Fatalf("target = %q (%v), want the redirected content", got, err)
	}
}

func TestDownloadFileWithChecksum(t *testing.T) {
	content := []byte("the artifact")
	sum := sha256.Sum256(content)
	good := hex.EncodeToString(sum[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(content)
	}))
	defer server.Close()

	t.Run("matching digest installs", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "out")
		if err := DownloadFileWithChecksum(server.URL, target, false, good); err != nil {
			t.Fatalf("DownloadFileWithChecksum: %v", err)
		}
		got, err := os.ReadFile(target)
		if err != nil || string(got) != string(content) {
			t.Fatalf("target = %q (%v), want the artifact", got, err)
		}
	})

	t.Run("digest case is ignored", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "out")
		if err := DownloadFileWithChecksum(server.URL, target, false, strings.ToUpper(good)); err != nil {
			t.Fatalf("DownloadFileWithChecksum: %v", err)
		}
	})

	t.Run("mismatch discards the download", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "out")
		err := DownloadFileWithChecksum(server.URL, target, false, strings.Repeat("0", 64))
		if err == nil || !strings.Contains(err.Error(), "sha256 mismatch") {
			t.Fatalf("err = %v, want a sha256 mismatch", err)
		}
		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if len(entries) != 0 {
			t.Fatalf("directory not empty after refused install: %v", entries)
		}
	})
}
