// Package prompts provides interactive user prompts for authentication and configuration.
// It handles GitHub authentication method selection, token management, and
// credential storage using the Charm huh library for terminal-based forms.
package prompts

import (
	"fmt"
	"strings"

	"charm.land/huh/v2"
	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/credentials"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/reporting"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
)

// GitHubAuthChoice represents the user's authentication choice.
type GitHubAuthChoice string

const (
	GitHubAuthOAuth  GitHubAuthChoice = "oauth"
	GitHubAuthManual GitHubAuthChoice = "manual"
	GitHubAuthSkip   GitHubAuthChoice = "skip"
)

// PromptGitHubAuthMethod displays a selection form for the user to choose between
// OAuth device flow, manual token entry, or skipping GitHub authentication.
func PromptGitHubAuthMethod() (GitHubAuthChoice, error) {
	var authChoice string

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
				Value(&authChoice),
		),
	)

	// Through the terminal lease, never bare: a form that reads stdin while
	// the TUI owns it freezes the run (and ctrl-c with it).
	err := reporting.WithTerminal(form.Run)
	if err != nil {
		return "", fmt.Errorf("authentication prompt failed: %w", err)
	}

	return GitHubAuthChoice(authChoice), nil
}

// PromptGitHubToken displays a secure input form for a GitHub personal access token.
// It validates that the token starts with a prefix GitHub issues; see
// gitHubTokenPrefixes.
func PromptGitHubToken() (string, error) {
	var token string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter your GitHub personal access token").
				// Both permission models, because the answer differs and the
				// classic-only wording sent fine-grained users looking for a
				// scope their token does not have.
				Description("Classic token: 'write:public_key' scope. Fine-grained token: 'Git SSH keys' user permission, write.").
				EchoMode(huh.EchoModePassword).
				Value(&token).
				Validate(validateGitHubToken),
		),
	)

	err := reporting.WithTerminal(form.Run)
	if err != nil {
		return "", fmt.Errorf("token entry failed: %w", err)
	}

	return token, nil
}

// gitHubTokenPrefixes are the token forms that can do the one thing this
// prompt collects a token for: POST /user/keys, which adds an SSH key to the
// authenticated user's account.
//
// github_pat_ is the one whose absence actually hurt. Fine-grained personal
// access tokens are what "Generate new token" offers first, so the common case
// was an operator pasting a valid, correctly-scoped token and being told it was
// invalid.
//
// The list is what /user/keys accepts, not every prefix GitHub issues. See
// gitHubTokenRejections for the ones deliberately left out.
//
// Order matters for the message only; HasPrefix is exact either way.
var gitHubTokenPrefixes = []string{
	"github_pat_", // fine-grained PAT, with the "Git SSH keys" user permission
	"ghp_",        // classic PAT, with the write:public_key scope
	"gho_",        // OAuth token, what the device flow returns
	"ghu_",        // GitHub App user access token (user-to-server)
}

// gitHubTokenRejections are real GitHub tokens that cannot do this job, mapped
// to the reason. Passing one through validation only moves the failure to the
// upload, where it surfaces as a bare 403 with nothing explaining it.
var gitHubTokenRejections = map[string]string{
	// An installation token acts as the app installation, not as a person, so
	// there is no authenticated user for /user/keys to add a key to. The
	// GitHub App token that does have a user is ghu_, which is accepted.
	"ghs_": "that is a GitHub App installation token, which has no authenticated user to add a key to; use a user access token (ghu_) or a personal access token",
}

// validateGitHubToken checks that a pasted value is a GitHub token that can
// upload an SSH key for the authenticated user.
func validateGitHubToken(s string) error {
	if s == "" {
		return fmt.Errorf("token cannot be empty")
	}
	for _, prefix := range gitHubTokenPrefixes {
		if strings.HasPrefix(s, prefix) {
			return nil
		}
	}
	// A real token of the wrong kind gets told which kind it is. "does not
	// look like a GitHub token" would be untrue and unactionable.
	for prefix, reason := range gitHubTokenRejections {
		if strings.HasPrefix(s, prefix) {
			return fmt.Errorf("%s", reason)
		}
	}
	// Name the accepted prefixes. "invalid GitHub token format" told an
	// operator holding a real token nothing about what was wrong with it.
	return fmt.Errorf("does not look like a GitHub token; expected one starting with %s", strings.Join(gitHubTokenPrefixes, ", "))
}

// PromptAndSaveGitHubToken collects a GitHub token via interactive prompt
// and persists it to the rwr config file.
func PromptAndSaveGitHubToken(initConfig *types.InitConfig) (string, error) {
	token, err := PromptGitHubToken()
	if err != nil {
		return "", err
	}

	// Save token using the same logic as OAuth
	if err := SaveGitHubTokenToConfig(token, initConfig); err != nil {
		log.Warnf("Failed to save token to config: %v", err)
		log.Infof("Token obtained but could not be saved to config. Re-run with --gh-api-key to supply it directly.")
	} else {
		log.Debugf("Token saved")
	}

	return token, nil
}

// SaveGitHubTokenToConfig persists a GitHub token: to the OS keyring when a
// backend is available, falling back to the rwr config file (plaintext at
// 0600) with a warning naming the file. If a different token already exists
// and interactive mode is on, it prompts the user to confirm the replacement.
func SaveGitHubTokenToConfig(token string, initConfig *types.InitConfig) error {
	// Check for an existing token: the config file (the grandfathered plaintext
	// store, still readable) and the keyring (where new tokens land).
	existingToken := viper.GetString("repository.gh_api_token")
	if existingToken == "" {
		existingToken, _ = credentials.FromKeyring("gh_api_token")
	}

	// If token exists and is different, prompt to confirm replacement (if interactive)
	if existingToken != "" && existingToken != token && initConfig.Variables.Flags.Interactive {
		replace, err := PromptConfirmTokenReplace()
		if err != nil {
			return fmt.Errorf("confirmation prompt failed: %w", err)
		}
		if !replace {
			return fmt.Errorf("user declined to replace existing token")
		}
	}

	// The registry holds the value for this run either way; persistence below
	// only decides where the next run reads it from.
	types.SetCredentialValue("gh_api_token", token)

	// Keyring first: it is the only store that keeps the token off disk in
	// plaintext. The config file is the fallback, not a second copy - when the
	// keyring save succeeds, no file gains the token value.
	saveErr := credentials.SaveToKeyring("gh_api_token", token)
	if saveErr == nil {
		log.Debugf("Token saved to the OS keyring")
		return nil
	}
	log.Warnf("OS keyring unavailable (%v); saving the token to %s instead - "+
		"it is stored in plaintext, readable only by your user (0600)",
		saveErr, viper.ConfigFileUsed())

	// Set the new token
	viper.Set("repository.gh_api_token", token)

	// The file holds the token, and viper writes at 0644 with no way to ask
	// for less. Pre-creating it at 0600 means the token is never on disk
	// world-readable, not even briefly. ConfigFileUsed is empty when no config
	// was loaded - then SafeWriteConfig below creates the file and the tighten
	// after it is the only guard (a first-run-only window).
	if path := viper.ConfigFileUsed(); path != "" {
		if err := helpers.PrecreateSecureConfigFile(path); err != nil {
			return err
		}
	}

	// Try to write config
	if err := viper.WriteConfig(); err != nil {
		// If config doesn't exist, create it
		if err := viper.SafeWriteConfig(); err != nil {
			return err
		}
	}

	// Belt and braces: verify nothing widened it.
	return helpers.SecureConfigFile(viper.ConfigFileUsed())
}

// PromptConfirmTokenReplace asks the user to confirm overwriting an existing
// GitHub token in the config file. Returns true if the user agrees.
func PromptConfirmTokenReplace() (bool, error) {
	var confirm bool

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("A GitHub token already exists in your config").
				Description("Do you want to replace it with the new token?").
				Affirmative("Yes, replace it").
				Negative("No, keep existing").
				Value(&confirm),
		),
	)

	err := reporting.WithTerminal(form.Run)
	if err != nil {
		return false, fmt.Errorf("confirmation prompt failed: %w", err)
	}

	return confirm, nil
}

// PromptForGitHubAuth orchestrates the complete GitHub authentication flow.
// It prompts for an auth method, then executes OAuth or manual token entry.
// The oauthFunc parameter should perform the OAuth device flow (e.g., processors.AuthenticateWithGitHub).
// Returns the token, its source ("oauth"/"manual"), and any error.
func PromptForGitHubAuth(initConfig *types.InitConfig, oauthFunc func(*types.InitConfig) (string, error)) (string, string, error) {
	// Check if we're in non-interactive mode
	if !initConfig.Variables.Flags.Interactive {
		return "", "", fmt.Errorf(`GitHub token not found. Please use one of:
  1. --gh-api-key / --gh-key flag
  2. --gh-auth to authenticate via OAuth
  3. GITHUB_TOKEN environment variable

To enable interactive prompts, use --interactive flag (or remove --interactive=false)`)
	}

	choice, err := PromptGitHubAuthMethod()
	if err != nil {
		return "", "", err
	}

	switch choice {
	case GitHubAuthOAuth:
		// Trigger OAuth flow
		token, err := oauthFunc(initConfig)
		if err != nil {
			return "", "", fmt.Errorf("OAuth authentication failed: %w", err)
		}
		return token, "oauth-prompted", nil

	case GitHubAuthManual:
		// Prompt for manual token entry
		token, err := PromptAndSaveGitHubToken(initConfig)
		if err != nil {
			return "", "", err
		}
		return token, "manual-entry", nil

	case GitHubAuthSkip:
		return "", "", fmt.Errorf("user chose to skip GitHub authentication")

	default:
		return "", "", fmt.Errorf("unknown authentication choice: %s", choice)
	}
}
