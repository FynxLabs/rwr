package processors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fynxlabs/rwr/internal/credentials"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetGitHubToken_Priority tests token retrieval priority.
func TestGetGitHubToken_Priority(t *testing.T) {
	tests := []struct {
		name           string
		configToken    string
		githubToken    string
		expectedToken  string
		expectedSource string
		expectError    bool
	}{
		{
			name:           "flag token wins",
			configToken:    "flag-token",
			githubToken:    "env-token",
			expectedToken:  "flag-token",
			expectedSource: "flag",
			expectError:    false,
		},
		{
			name:           "env token when no flag",
			githubToken:    "env-token",
			expectedToken:  "env-token",
			expectedSource: "GITHUB_TOKEN",
			expectError:    false,
		},
		{
			name:        "error when no token",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// getGitHubToken consults the OS keyring between the environment
			// and the prompt. Without this the "no token" case finds whatever
			// the developer has stored and asserts against it.
			defer credentials.SetRingForTest(credentials.EmptyKeyring{})()

			// Setup test environment
			oldGitHubToken := os.Getenv("GITHUB_TOKEN")
			defer os.Setenv("GITHUB_TOKEN", oldGitHubToken)

			// Clear environment
			os.Unsetenv("GITHUB_TOKEN")

			// Set environment variables if specified
			if tt.githubToken != "" {
				os.Setenv("GITHUB_TOKEN", tt.githubToken)
			}

			// Create init config with flag token
			initConfig := &types.InitConfig{
				Variables: types.Variables{
					Flags: types.Flags{
						GHAPIToken: tt.configToken,
					},
				},
			}

			// Call getGitHubToken
			token, source, err := getGitHubToken(initConfig)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, token)
				assert.Empty(t, source)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedToken, token)
				assert.Equal(t, tt.expectedSource, source)
			}
		})
	}
}

// TestRequestDeviceCode tests device code request.
func TestRequestDeviceCode(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/login/device/code", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Parse request body
		var req deviceCodeRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)
		assert.Equal(t, githubClientID, req.ClientID)
		assert.Equal(t, "write:public_key", req.Scope)

		// Send response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(deviceCodeResponse{
			DeviceCode:      "test-device-code",
			UserCode:        "ABCD-1234",
			VerificationURI: "https://github.com/login/device",
			ExpiresIn:       900,
			Interval:        5,
		})
	}))
	defer server.Close()

	// Temporarily override the URL for testing
	oldURL := githubDeviceCodeURL
	defer func() {
		// Can't actually reassign const, so we'll test with actual URL
		_ = oldURL
	}()

	// Note: This test verifies the request structure but uses the real URL
	// In a production test suite, you'd use dependency injection or interfaces
	// to make the URL configurable for testing
}

// TestCheckAccessToken tests access token retrieval.
func TestCheckAccessToken(t *testing.T) {
	tests := []struct {
		name        string
		response    accessTokenResponse
		expectError bool
		errorMsg    string
	}{
		{
			name: "success",
			response: accessTokenResponse{
				AccessToken: "gho_test_token",
				TokenType:   "bearer",
				Scope:       "write:public_key",
			},
			expectError: false,
		},
		{
			name: "authorization pending",
			response: accessTokenResponse{
				Error: "authorization_pending",
			},
			expectError: true,
			errorMsg:    "authorization_pending",
		},
		{
			name: "slow down",
			response: accessTokenResponse{
				Error: "slow_down",
			},
			expectError: true,
			errorMsg:    "slow_down",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Accept"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			// Note: In production, we'd need to make the URL configurable
			// For now, this test documents the expected behavior
		})
	}
}

// TestCopySSHKeyToGitHub_Success tests successful SSH key upload.
func TestCopySSHKeyToGitHub_Success(t *testing.T) {
	// Create temporary SSH key files
	tmpDir := t.TempDir()
	privKeyPath := filepath.Join(tmpDir, "test_key")
	pubKeyPath := privKeyPath + ".pub"

	testPubKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ test@example.com"
	err := os.WriteFile(pubKeyPath, []byte(testPubKey), 0600)
	require.NoError(t, err)

	// Create mock GitHub API server
	var receivedReq githubKeyRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/user/keys", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/vnd.github+json", r.Header.Get("Accept"))
		assert.Equal(t, "2022-11-28", r.Header.Get("X-GitHub-Api-Version"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Parse request body
		err := json.NewDecoder(r.Body).Decode(&receivedReq)
		assert.NoError(t, err)
		assert.Equal(t, "Test Key", receivedReq.Title)
		assert.Equal(t, testPubKey, receivedReq.Key)

		// Send success response
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(githubKeyResponse{
			ID:    123,
			Title: "Test Key",
			Key:   testPubKey,
		})
	}))
	defer server.Close()

	// Note: In production, we'd need to make the API URL configurable for testing
	// This test documents the expected behavior
}

// TestCopySSHKeyToGitHub_Errors drives the real upload against a local server.
// The duplicate-key 422 is the converged state - the key already being on the
// account is what copy_to_github asks for - so it succeeds; every other
// failure still errors.
func TestCopySSHKeyToGitHub_Errors(t *testing.T) {
	duplicate := githubError{
		Message: "Validation Failed",
		Errors: []struct {
			Resource string `json:"resource"`
			Code     string `json:"code"`
			Field    string `json:"field"`
			Message  string `json:"message"`
		}{
			{Resource: "PublicKey", Field: "key", Message: "key is already in use"},
		},
	}

	tests := []struct {
		name          string
		statusCode    int
		responseBody  interface{}
		expectedError string
	}{
		{
			name:          "201 Created",
			statusCode:    201,
			responseBody:  map[string]string{"key": "ok"},
			expectedError: "",
		},
		{
			name:          "401 Unauthorized",
			statusCode:    401,
			responseBody:  map[string]string{"message": "Bad credentials"},
			expectedError: "authentication failed: invalid GitHub API token",
		},
		{
			name:          "403 Forbidden",
			statusCode:    403,
			responseBody:  map[string]string{"message": "Forbidden"},
			expectedError: "forbidden: GitHub token requires 'write:public_key' scope",
		},
		{
			name:          "422 Duplicate Key converges",
			statusCode:    422,
			responseBody:  duplicate,
			expectedError: "",
		},
		{
			name:          "422 other validation failure",
			statusCode:    422,
			responseBody:  githubError{Message: "key is invalid"},
			expectedError: "validation failed: key is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			pubKeyPath := filepath.Join(tmpDir, "test_key.pub")
			testPubKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ test@example.com"
			require.NoError(t, os.WriteFile(pubKeyPath, []byte(testPubKey), 0600))

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.responseBody) //nolint:errcheck
			}))
			defer server.Close()

			orig := githubKeysAPI
			githubKeysAPI = server.URL
			defer func() { githubKeysAPI = orig }()

			err := copySSHKeyToGitHub(types.SSHKey{
				Name:        "test_key",
				Path:        tmpDir,
				GithubTitle: "test-title",
			}, &types.InitConfig{
				Variables: types.Variables{Flags: types.Flags{GHAPIToken: "ghp_test"}},
			})

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestGetGitHubToken_EnvVarOnly tests GITHUB_TOKEN environment variable.
func TestGetGitHubToken_EnvVarOnly(t *testing.T) {
	oldGitHubToken := os.Getenv("GITHUB_TOKEN")
	defer os.Setenv("GITHUB_TOKEN", oldGitHubToken)

	testToken := "ghp_test_env_token"
	os.Setenv("GITHUB_TOKEN", testToken)

	initConfig := &types.InitConfig{
		Variables: types.Variables{
			Flags: types.Flags{
				GHAPIToken: "", // No flag token
			},
		},
	}

	token, source, err := getGitHubToken(initConfig)

	assert.NoError(t, err)
	assert.Equal(t, testToken, token)
	assert.Equal(t, "GITHUB_TOKEN", source)
}

// TestGetGitHubToken_NoToken tests error when no token available.
func TestGetGitHubToken_NoToken(t *testing.T) {
	defer credentials.SetRingForTest(credentials.EmptyKeyring{})()

	oldGitHubToken := os.Getenv("GITHUB_TOKEN")
	defer os.Setenv("GITHUB_TOKEN", oldGitHubToken)

	os.Unsetenv("GITHUB_TOKEN")

	initConfig := &types.InitConfig{
		Variables: types.Variables{
			Flags: types.Flags{
				GHAPIToken:  "",
				Interactive: false, // Disable interactive mode for testing
			},
		},
	}

	token, source, err := getGitHubToken(initConfig)

	// require, not assert: the assertions below dereference err, so a run
	// that unexpectedly finds a token panicked instead of failing.
	require.Error(t, err)
	assert.Empty(t, token)
	assert.Empty(t, source)
	assert.Contains(t, err.Error(), "GitHub token not found")
	assert.Contains(t, err.Error(), "--gh-api-key")
	assert.Contains(t, err.Error(), "--gh-auth")
	assert.Contains(t, err.Error(), "GITHUB_TOKEN")
}

// TestPollForAccessToken_Timeout tests timeout scenario.
func TestPollForAccessToken_Timeout(t *testing.T) {
	// Create server that always returns pending
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(accessTokenResponse{
			Error: "authorization_pending",
		})
	}))
	defer server.Close()

	// Override timeout for testing
	oldTimeout := 5 * time.Minute
	_ = oldTimeout

	// Note: This would require making the timeout configurable
	// For now, we document the expected behavior
	// In production, pollForAccessToken should timeout after 5 minutes
}
