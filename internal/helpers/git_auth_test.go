package helpers

import (
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

const testGHToken = "ghp_testtoken"

func testInitConfig() *types.InitConfig {
	return &types.InitConfig{
		Variables: types.Variables{
			Flags: types.Flags{GHAPIToken: testGHToken},
		},
	}
}

// TestGetAuthMethodTokenScope pins the token to GitHub. repo.URL comes from a
// blueprint, so any host it names would otherwise be handed the user's PAT.
func TestGetAuthMethodTokenScope(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantToken  bool
		wantErr    bool
		errFragmnt string
	}{
		{name: "github attaches token", url: "https://github.com/fynxlabs/rwr.git", wantToken: true},
		{name: "github raw host attaches token", url: "https://raw.githubusercontent.com/fynxlabs/rwr/main/f", wantToken: true},
		{name: "other https host gets no token", url: "https://attacker.tld/r.git"},
		{name: "http is refused", url: "http://attacker.tld/r.git", wantErr: true, errFragmnt: "cleartext"},
		{name: "http github is refused too", url: "http://github.com/fynxlabs/rwr.git", wantErr: true, errFragmnt: "cleartext"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth, err := getAuthMethod(tt.url, testInitConfig())

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error for %s, got auth %v", tt.url, auth)
				}
				if !strings.Contains(err.Error(), tt.errFragmnt) {
					t.Errorf("error %q does not mention %q", err, tt.errFragmnt)
				}
				if auth != nil {
					t.Errorf("credential returned alongside an error: %v", auth)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.url, err)
			}

			basic, isBasic := auth.(*http.BasicAuth)
			if !tt.wantToken {
				if auth != nil {
					t.Fatalf("token sent to %s: %v", tt.url, auth)
				}
				return
			}
			if !isBasic || basic.Password != testGHToken {
				t.Fatalf("expected the GitHub token for %s, got %v", tt.url, auth)
			}
		})
	}
}
