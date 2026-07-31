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
		{name: "personal access token", token: "ghp_abcdef1234567890"},
		{name: "oauth token", token: "gho_abcdef1234567890"},
		{name: "user-to-server token", token: "ghu_abcdef1234567890"},
		{name: "empty", token: "", wantErr: "cannot be empty"},
		{name: "no recognised prefix", token: "abcdef1234567890", wantErr: "invalid GitHub token format"},
		{name: "server-to-server tokens are not accepted", token: "ghs_abcdef", wantErr: "invalid GitHub token format"},
		{name: "prefix must lead", token: "xghp_abcdef", wantErr: "invalid GitHub token format"},
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
// running them — Run needs a terminal.
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
