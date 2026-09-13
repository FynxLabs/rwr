package credentials

import (
	"context"
	"os"
	"testing"
)

func TestBitwardenServerRequiresHTTPS(t *testing.T) {
	t.Parallel()
	for _, server := range []string{"https://vault.bitwarden.com", "https://vault.bitwarden.eu", "https://vault.example.com:8443/api"} {
		if err := validateBitwardenServer(server); err != nil {
			t.Errorf("%s: %v", server, err)
		}
	}
	for _, server := range []string{"", "http://vault.example.com", "vault.example.com", "https:///missing", "https://user:password@vault.example.com"} {
		if err := validateBitwardenServer(server); err == nil {
			t.Errorf("accepted %q", server)
		}
	}
}

func TestProtectedPasswordIsChildOnly(t *testing.T) {
	t.Setenv("RWR_BW_PASSWORD", "")
	value, err := ProtectedCommand(context.Background(), "/bin/sh", []string{"-c", "printf '%s' \"$RWR_BW_PASSWORD\""}, map[string]string{"RWR_BW_PASSWORD": "master-secret"}, nil)
	if err != nil || string(value) != "master-secret" {
		t.Fatalf("child password unavailable: %v", err)
	}
	if os.Getenv("RWR_BW_PASSWORD") != "" {
		t.Fatal("password leaked to parent")
	}
}
