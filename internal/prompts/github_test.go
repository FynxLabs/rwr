package prompts

import (
	"strings"
	"testing"

	"charm.land/huh/v2"
)

func TestValidateGitHubToken(t *testing.T) {
	t.Parallel()

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
		{name: "github app user access token", token: "ghu_abcdef1234567890"},
		// A GitHub App *installation* token acts as the installation, not as a
		// person, so /user/keys has no authenticated user to add a key to.
		// Accepting it would only move the failure to the upload, where it
		// arrives as a bare 403.
		{name: "installation token is the wrong kind", token: "ghs_abcdef1234567890",
			wantErr: "has no authenticated user"},
		{name: "empty", token: "", wantErr: "cannot be empty"},
		{name: "no recognised prefix", token: "abcdef1234567890", wantErr: "does not look like a GitHub token"},
		{name: "prefix must lead", token: "xghp_abcdef", wantErr: "does not look like a GitHub token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

	const pasted = "definitely-a-secret-value"

	err := validateGitHubToken(pasted)
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), pasted) {
		t.Errorf("error repeats the pasted credential: %v", err)
	}
}

// A real GitHub token of the wrong kind is told which kind it is, and pointed
// at one that works. Falling back to "does not look like a GitHub token" would
// be untrue, and leaves an operator staring at a token they can see is valid.
func TestValidateGitHubTokenExplainsAWrongKindOfToken(t *testing.T) {
	t.Parallel()

	err := validateGitHubToken("ghs_abcdef1234567890")
	if err == nil {
		t.Fatal("an installation token was accepted for a /user/keys upload")
	}
	if strings.Contains(err.Error(), "does not look like a GitHub token") {
		t.Errorf("a real token was called unrecognisable: %v", err)
	}
	// It has to name the alternative, or the operator has nothing to do next.
	if !strings.Contains(err.Error(), "ghu_") {
		t.Errorf("error does not point at a token type that works: %v", err)
	}
}

// Nothing may appear in both lists: a prefix that is accepted and explained as
// a rejection would resolve by list order rather than by intent.
func TestGitHubTokenPrefixesAndRejectionsAreDisjoint(t *testing.T) {
	t.Parallel()

	for _, accepted := range gitHubTokenPrefixes {
		if _, rejected := gitHubTokenRejections[accepted]; rejected {
			t.Errorf("%q is both accepted and rejected", accepted)
		}
	}
}
