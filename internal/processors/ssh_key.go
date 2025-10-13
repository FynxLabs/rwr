package processors

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
)

const (
	// RWR GitHub App - https://github.com/apps/rwr-rinse-wash-repeat
	// App ID: 2107251
	githubClientID       = "Iv23lifvLgztwMVAOEEu"
	githubDeviceCodeURL  = "https://github.com/login/device/code"
	githubAccessTokenURL = "https://github.com/login/oauth/access_token"
)

// GitHub API request structure
type githubKeyRequest struct {
	Title string `json:"title"`
	Key   string `json:"key"`
}

// GitHub API success response structure
type githubKeyResponse struct {
	Key       string `json:"key"`
	ID        int    `json:"id"`
	URL       string `json:"url"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	Verified  bool   `json:"verified"`
	ReadOnly  bool   `json:"read_only"`
}

// GitHub API error structure
type githubError struct {
	Message          string `json:"message"`
	DocumentationURL string `json:"documentation_url,omitempty"`
	Errors           []struct {
		Resource string `json:"resource"`
		Code     string `json:"code"`
		Field    string `json:"field"`
		Message  string `json:"message"`
	} `json:"errors,omitempty"`
}

// OAuth device flow structures
type deviceCodeRequest struct {
	ClientID string `json:"client_id"`
	Scope    string `json:"scope"`
}

type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type accessTokenRequest struct {
	ClientID   string `json:"client_id"`
	DeviceCode string `json:"device_code"`
	GrantType  string `json:"grant_type"`
}

type accessTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error,omitempty"`
}

func ProcessSSHKeys(blueprintData []byte, format string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var sshKeyData types.SSHKeyData
	var err error

	log.Debugf("Processing SSH keys from blueprint")

	// Unmarshal the blueprint data
	err = helpers.UnmarshalBlueprint(blueprintData, format, &sshKeyData)
	if err != nil {
		return fmt.Errorf("error unmarshaling SSH key blueprint: %v", err)
	}

	// Filter SSH keys based on active profiles
	filteredSSHKeys := helpers.FilterByProfiles(sshKeyData.SSHKeys, initConfig.Variables.Flags.Profiles)

	log.Debugf("Filtering SSH keys: %d total, %d matching active profiles %v",
		len(sshKeyData.SSHKeys), len(filteredSSHKeys), initConfig.Variables.Flags.Profiles)

	err = processSSHKeys(filteredSSHKeys, osInfo, initConfig)
	if err != nil {
		log.Errorf("Error processing SSH Keys: %v", err)
		return fmt.Errorf("error processing SSH Keys: %w", err)
	}

	return nil
}

func processSSHKeys(sshKeys []types.SSHKey, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	// Process the SSH keys
	for _, sshKey := range sshKeys {
		// Ensure required packages are installed
		err := ensureSSHPackages(osInfo, initConfig)
		if err != nil {
			return fmt.Errorf("error ensuring SSH packages: %v", err)
		}

		// Generate SSH key
		keyPath, err := generateSSHKey(sshKey)
		if err != nil {
			log.Errorf("Error generating SSH key %s: %v", sshKey.Name, err)
			continue // Continue with the next key instead of returning
		}

		// Copy public key to GitHub if requested
		if sshKey.CopyToGitHub {
			err = copySSHKeyToGitHub(sshKey, initConfig)
			if err != nil {
				log.Errorf("Error copying SSH key %s to GitHub: %v", sshKey.Name, err)
				continue // Continue with the next key instead of returning
			}
		}

		// Set as RWR SSH Key if requested
		if sshKey.SetAsRWRSSHKey {
			err = setAsRWRSSHKey(keyPath)
			if err != nil {
				log.Errorf("Error setting SSH key %s as RWR SSH Key: %v", sshKey.Name, err)
				continue // Continue with the next key instead of returning
			}
		}
	}

	return nil
}

func ensureSSHPackages(osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	switch runtime.GOOS {
	case "windows":
		pkgData := &types.PackagesData{
			Packages: []types.Package{
				{Name: "openssh", Action: "install", PackageManager: "chocolatey"},
			},
		}
		return ProcessPackages(nil, pkgData, "", osInfo, initConfig)
	case "darwin":
		pkgData := &types.PackagesData{
			Packages: []types.Package{
				{Name: "openssh", Action: "install", PackageManager: "brew"},
			},
		}
		return ProcessPackages(nil, pkgData, "", osInfo, initConfig)
	default:
		// For Linux, OpenSSH is typically pre-installed
		return nil
	}
}

func generateSSHKey(sshKey types.SSHKey) (string, error) {
	sshPath := filepath.Join(sshKey.Path, sshKey.Name)

	// Check if the SSH key already exists
	if _, err := os.Stat(sshPath); err == nil {
		log.Warnf("SSH key %s already exists. Skipping generation.", sshPath)
		return sshPath, nil
	}

	// Build the command differently based on whether we need a passphrase
	var cmd types.Command

	if sshKey.NoPassphrase {
		// For no passphrase, use a single string command that properly handles the empty string
		cmdStr := fmt.Sprintf("ssh-keygen -t %s -C %s -f %s -N ''",
			sshKey.Type, sshKey.Comment, sshPath)

		cmd = types.Command{
			Exec: cmdStr,
			Args: []string{},
		}
	} else {
		// For normal case with passphrase prompt
		cmd = types.Command{
			Exec: "ssh-keygen",
			Args: []string{
				"-t", sshKey.Type,
				"-C", sshKey.Comment,
				"-f", sshPath,
			},
		}
	}

	err := system.RunCommand(cmd, true)
	if err != nil {
		return "", fmt.Errorf("error generating SSH key: %v", err)
	}

	log.Infof("SSH key generated: %s", sshPath)
	return sshPath, nil
}

func setAsRWRSSHKey(keyPath string) error {
	// Read the private key file
	privateKey, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("error reading private key file: %v", err)
	}

	// Encode the private key as base64
	encodedKey := base64.StdEncoding.EncodeToString(privateKey)

	// Set the encoded key in Viper configuration
	viper.Set("repository.ssh_private_key", encodedKey)

	// Write the updated configuration to file
	err = viper.WriteConfig()
	if err != nil {
		return fmt.Errorf("error writing updated configuration: %v", err)
	}

	log.Infof("SSH key %s set as RWR SSH Key", keyPath)
	return nil
}

// AuthenticateWithGitHub performs OAuth device flow authentication
func AuthenticateWithGitHub(initConfig *types.InitConfig) (string, error) {
	log.Infof("Starting GitHub authentication...")

	// Step 1: Request device code
	deviceResp, err := requestDeviceCode()
	if err != nil {
		return "", fmt.Errorf("failed to request device code: %w", err)
	}

	// Step 2: Display instructions to user
	log.Infof("")
	log.Infof("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Infof("  GitHub Authentication Required")
	log.Infof("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Infof("")
	log.Infof("1. Visit: %s", deviceResp.VerificationURI)
	log.Infof("2. Enter code: %s", deviceResp.UserCode)
	log.Infof("")
	log.Infof("Waiting for authorization...")
	log.Infof("")

	// Step 3: Poll for access token
	token, err := pollForAccessToken(deviceResp.DeviceCode, deviceResp.Interval)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	// Step 4: Store token in config
	viper.Set("repository.gh_api_token", token)

	// Try to write config - create if doesn't exist
	err = viper.WriteConfig()
	if err != nil {
		// If config doesn't exist, try SafeWriteConfig
		err = viper.SafeWriteConfig()
		if err != nil {
			log.Warnf("Failed to save token to config: %v", err)
			log.Infof("Token obtained but not saved. Use --gh-api-key=%s", token)
		} else {
			log.Infof("✓ Authentication successful! Token saved to config.")
		}
	} else {
		log.Infof("✓ Authentication successful! Token saved to config.")
	}

	return token, nil
}

// requestDeviceCode requests a device code from GitHub
func requestDeviceCode() (*deviceCodeResponse, error) {
	payload := deviceCodeRequest{
		ClientID: githubClientID,
		Scope:    "write:public_key",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", githubDeviceCodeURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var deviceResp deviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceResp); err != nil {
		return nil, err
	}

	return &deviceResp, nil
}

// pollForAccessToken polls GitHub for access token approval
func pollForAccessToken(deviceCode string, interval int) (string, error) {
	if interval == 0 {
		interval = 5 // Default to 5 seconds
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-timeout:
			return "", fmt.Errorf("authentication timeout: user did not authorize within 5 minutes")
		case <-ticker.C:
			token, err := checkAccessToken(deviceCode)
			if err != nil {
				if strings.Contains(err.Error(), "authorization_pending") {
					continue
				}
				if strings.Contains(err.Error(), "slow_down") {
					ticker.Reset(time.Duration(interval+5) * time.Second)
					continue
				}
				return "", err
			}
			return token, nil
		}
	}
}

// checkAccessToken attempts to get access token
func checkAccessToken(deviceCode string) (string, error) {
	payload := accessTokenRequest{
		ClientID:   githubClientID,
		DeviceCode: deviceCode,
		GrantType:  "urn:ietf:params:oauth:grant-type:device_code",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", githubAccessTokenURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tokenResp accessTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("OAuth error: %s", tokenResp.Error)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("no access token received")
	}

	return tokenResp.AccessToken, nil
}

// getGitHubToken retrieves GitHub token with priority: flag → env
func getGitHubToken(initConfig *types.InitConfig) (string, string, error) {
	// Priority 1: Explicit token from config/flag
	if token := initConfig.Variables.Flags.GHAPIToken; token != "" {
		log.Debugf("Using GitHub token from --gh-api-key flag")
		return token, "flag", nil
	}

	// Priority 2: GITHUB_TOKEN environment variable
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		log.Debugf("Using GitHub token from GITHUB_TOKEN environment variable")
		return token, "GITHUB_TOKEN", nil
	}

	// No token found
	return "", "", fmt.Errorf(`GitHub token not found. Please use one of:
	 1. --gh-api-key / --gh-key flag
	 2. --gh-auth to authenticate via OAuth
	 3. GITHUB_TOKEN environment variable`)
}

func copySSHKeyToGitHub(sshKey types.SSHKey, initConfig *types.InitConfig) error {
	// Get GitHub token with fallback hierarchy
	token, source, err := getGitHubToken(initConfig)
	if err != nil {
		return err
	}

	log.Infof("Using GitHub token from: %s", source)

	// Read SSH public key
	sshPath := filepath.Join(sshKey.Path, sshKey.Name)
	publicKeyPath := sshPath + ".pub"
	publicKeyBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("error reading public key file: %v", err)
	}

	// Get or prompt for title
	var title string
	if sshKey.GithubTitle == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("error getting hostname: %v", err)
		}

		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Enter GitHub SSH key title (default: %s): ", hostname)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading user input: %v", err)
		}

		title = strings.TrimSpace(input)
		if title == "" {
			title = hostname
		}
	} else {
		title = sshKey.GithubTitle
	}

	// Create request payload
	payload := githubKeyRequest{
		Title: title,
		Key:   strings.TrimSpace(string(publicKeyBytes)),
	}

	// Marshal JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.github.com/user/keys", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("network error connecting to GitHub API: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}

	// Handle response based on status code
	switch resp.StatusCode {
	case 201:
		log.Infof("SSH public key added to GitHub: %s", title)
		return nil
	case 401:
		return fmt.Errorf("authentication failed: invalid GitHub API token")
	case 403:
		return fmt.Errorf("forbidden: GitHub token requires 'write:public_key' scope")
	case 422:
		// Parse error details
		var ghErr githubError
		if err := json.Unmarshal(body, &ghErr); err == nil && ghErr.Message != "" {
			// Check if it's a duplicate key error
			if len(ghErr.Errors) > 0 {
				for _, e := range ghErr.Errors {
					if e.Field == "key" && strings.Contains(strings.ToLower(e.Message), "already in use") {
						return fmt.Errorf("validation failed: this SSH key already exists in your GitHub account")
					}
				}
			}
			return fmt.Errorf("validation failed: %s", ghErr.Message)
		}
		return fmt.Errorf("validation failed: key may already exist or be invalid")
	default:
		return fmt.Errorf("unexpected GitHub API response (%d): %s", resp.StatusCode, string(body))
	}
}
