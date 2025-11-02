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

// PromptGitHubAuthMethod prompts the user to choose a GitHub authentication method
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

// PromptGitHubToken prompts the user to manually enter a GitHub token
func PromptGitHubToken() (string, error) {
	var token string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter your GitHub personal access token").
				Description("Token needs 'write:public_key' scope").
				Password(true).
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

// PromptAndSaveGitHubToken prompts for a token and saves it to config
func PromptAndSaveGitHubToken() (string, error) {
	token, err := PromptGitHubToken()
	if err != nil {
		return "", err
	}

	// Save token to config
	viper.Set("repository.gh_api_token", token)
	if err := viper.WriteConfig(); err != nil {
		// If config doesn't exist, try SafeWriteConfig
		if err := viper.SafeWriteConfig(); err != nil {
			log.Warnf("Failed to save token to config: %v", err)
			log.Infof("Token obtained but not saved. Use --gh-api-key=%s", token)
		} else {
			log.Debugf("Token saved to config")
		}
	} else {
		log.Debugf("Token saved to config")
	}

	return token, nil
}

// PromptForGitHubAuth handles the complete GitHub authentication prompt flow
// It prompts for auth method, then executes the chosen method
// oauthFunc should be the function that performs OAuth flow (e.g., processors.AuthenticateWithGitHub)
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
		token, err := PromptAndSaveGitHubToken()
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
