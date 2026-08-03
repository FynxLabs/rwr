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

	err := form.Run()
	if err != nil {
		return "", fmt.Errorf("authentication prompt failed: %w", err)
	}

	return GitHubAuthChoice(authChoice), nil
}

// PromptGitHubToken displays a secure input form for a GitHub personal access token.
// It validates that the token starts with a recognized prefix (ghp_, gho_, ghu_).
func PromptGitHubToken() (string, error) {
	var token string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter your GitHub personal access token").
				Description("Token needs 'write:public_key' scope").
				EchoMode(huh.EchoModePassword).
				Value(&token).
				Validate(validateGitHubToken),
		),
	)

	err := form.Run()
	if err != nil {
		return "", fmt.Errorf("token entry failed: %w", err)
	}

	return token, nil
}

// validateGitHubToken checks that a pasted value looks like a GitHub personal
// access token: non-empty and carrying a recognised prefix (ghp_ personal, gho_
// OAuth, ghu_ user-to-server).
func validateGitHubToken(s string) error {
	if s == "" {
		return fmt.Errorf("token cannot be empty")
	}
	if !strings.HasPrefix(s, "ghp_") && !strings.HasPrefix(s, "gho_") && !strings.HasPrefix(s, "ghu_") {
		return fmt.Errorf("invalid GitHub token format")
	}
	return nil
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
	// plaintext. The config file is the fallback, not a second copy — when the
	// keyring save succeeds, no file gains the token value.
	saveErr := credentials.SaveToKeyring("gh_api_token", token)
	if saveErr == nil {
		log.Debugf("Token saved to the OS keyring")
		return nil
	}
	log.Warnf("OS keyring unavailable (%v); saving the token to %s instead — "+
		"it is stored in plaintext, readable only by your user (0600)",
		saveErr, viper.ConfigFileUsed())

	// Set the new token
	viper.Set("repository.gh_api_token", token)

	// The file holds the token, and viper writes at 0644 with no way to ask
	// for less. Pre-creating it at 0600 means the token is never on disk
	// world-readable, not even briefly. ConfigFileUsed is empty when no config
	// was loaded — then SafeWriteConfig below creates the file and the tighten
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

	err := form.Run()
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
