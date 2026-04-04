// Package prompts provides interactive user prompts for authentication and configuration.
// It handles GitHub authentication method selection, token management, and
// credential storage using the Charm huh library for terminal-based forms.
package prompts

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
)

// GitHubAuthChoice represents the user's authentication choice
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
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("token cannot be empty")
					}
					if !strings.HasPrefix(s, "ghp_") && !strings.HasPrefix(s, "gho_") && !strings.HasPrefix(s, "ghu_") {
						return fmt.Errorf("invalid GitHub token format")
					}
					return nil
				}),
		),
	)

	err := form.Run()
	if err != nil {
		return "", fmt.Errorf("token entry failed: %w", err)
	}

	return token, nil
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
		log.Infof("Token obtained but not saved. Use --gh-api-key=%s", token)
	} else {
		log.Debugf("Token saved to config")
	}

	return token, nil
}

// SaveGitHubTokenToConfig writes a GitHub token to the rwr config file.
// If a different token already exists and interactive mode is on, it prompts
// the user to confirm the replacement.
func SaveGitHubTokenToConfig(token string, initConfig *types.InitConfig) error {
	// Check if token already exists in config
	existingToken := viper.GetString("repository.gh_api_token")

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

	// Set the new token
	viper.Set("repository.gh_api_token", token)

	// Try to write config
	err := viper.WriteConfig()
	if err != nil {
		// If config doesn't exist, create it
		return viper.SafeWriteConfig()
	}
	return nil
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
