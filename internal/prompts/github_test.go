package prompts

import (
	"strings"
	"testing"

	"charm.land/huh/v2"
)

func TestValidateGitHubToken(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr string
	}{
		// The one that mattered: fine-grained PATs are what GitHub's
		// "Generate new token" offers first, so this was the common case
		// being refused.
		{name: "fine-grained personal access token", token: "github_pat_11ABCDEFG0abcdef1234567890"},
		{name: "classic personal access token", token: "ghp_abcdef1234567890"},
		{name: "oauth token", token: "gho_abcdef1234567890"},
		{name: "user-to-server token", token: "ghu_abcdef1234567890"},
		// A GitHub App installation token, which is what a CI job hands to
		// rwr. Previously refused; that was characterisation of the
		// implementation rather than a decision about App tokens.
		{name: "server-to-server token", token: "ghs_abcdef1234567890"},
		{name: "empty", token: "", wantErr: "cannot be empty"},
		{name: "no recognised prefix", token: "abcdef1234567890", wantErr: "does not look like a GitHub token"},
		{name: "prefix must lead", token: "xghp_abcdef", wantErr: "does not look like a GitHub token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGitHubToken(tt.token)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateGitHubToken(%q) = %v, want nil", tt.token, err)
				}
				return
			}

			if err == nil {
				t.Fatalf("validateGitHubToken(%q) = nil, want error containing %q", tt.token, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("validateGitHubToken(%q) = %q, want it to contain %q", tt.token, err, tt.wantErr)
			}
		})
	}
}

// The huh v2 upgrade came with a module rename (github.com/charmbracelet/huh ->
// charm.land/huh/v2), so it is worth exercising the builder API rather than only
// compiling against it. These construct the same forms the prompts build, without
// running them - Run needs a terminal.
func TestFormsBuildAgainstHuhV2(t *testing.T) {
	t.Run("select", func(t *testing.T) {
		var choice string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("GitHub authentication required to upload SSH key").
					Description("How would you like to authenticate?").
					Options(
						huh.NewOption("Authenticate with OAuth (recommended)", "oauth"),
						huh.NewOption("Enter GitHub token manually", "manual"),
						huh.NewOption("Skip (don't upload to GitHub)", "skip"),
					).
					Value(&choice),
			),
		)
		if form == nil {
			t.Fatal("huh.NewForm returned nil for the auth method select")
		}
	})

	t.Run("password input with validation", func(t *testing.T) {
		var token string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Enter your GitHub personal access token").
					EchoMode(huh.EchoModePassword).
					Value(&token).
					Validate(validateGitHubToken),
			),
		)
		if form == nil {
			t.Fatal("huh.NewForm returned nil for the token input")
		}
	})

	t.Run("confirm", func(t *testing.T) {
		var confirm bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("A GitHub token already exists in your config").
					Affirmative("Yes, replace it").
					Negative("No, keep existing").
					Value(&confirm),
			),
		)
		if form == nil {
			t.Fatal("huh.NewForm returned nil for the confirm")
		}
	})
}

// The rejection has to name what would be accepted. "invalid GitHub token
// format" left an operator holding a real, correctly-scoped token with
// nothing to act on.
func TestValidateGitHubTokenErrorNamesTheAcceptedPrefixes(t *testing.T) {
	err := validateGitHubToken("not-a-token")
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, prefix := range gitHubTokenPrefixes {
		if !strings.Contains(err.Error(), prefix) {
			t.Errorf("error does not mention %q: %v", prefix, err)
		}
	}
}

// The message must not echo the value back: it is a credential, and the
// operator already knows what they pasted.
func TestValidateGitHubTokenErrorOmitsTheValue(t *testing.T) {
	const pasted = "definitely-a-secret-value"

	err := validateGitHubToken(pasted)
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), pasted) {
		t.Errorf("error repeats the pasted credential: %v", err)
	}
}
