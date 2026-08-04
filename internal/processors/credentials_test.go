package processors

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
)

const (
	fakeToken = "ghp_notarealtokenbutlooksliketone"
	fakeKey   = "LS0tLS1CRUdJTiBPUEVOU1NIIFBSSVZBVEUgS0VZLS0tLS0K"
)

// setUserDefinedAndEnvVariables exported every viper key as RWR_VAR_*, and
// setupCommandEnvironment copies os.Environ() into every command rwr spawns - so
// the GitHub token and the base64 SSH private key were readable by any script a
// blueprint chose to run. Blueprints are cloned from git repositories.
//
// This runs a real process and reads its actual environment.
func TestSpawnedCommandsDoNotInheritCredentials(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /usr/bin/env")
	}

	viper.Reset()
	defer viper.Reset()

	viper.Set("repository.gh_api_token", fakeToken)
	viper.Set("repository.ssh_private_key", fakeKey)
	viper.Set("repository.init-file", "init.yaml")

	initConfig := &types.InitConfig{}
	initConfig.Variables.UserDefined = map[string]interface{}{}

	if err := setUserDefinedAndEnvVariables(initConfig); err != nil {
		t.Fatalf("setUserDefinedAndEnvVariables: %v", err)
	}
	defer func() {
		for _, key := range []string{
			"RWR_VAR_REPOSITORY_GH_API_TOKEN",
			"RWR_VAR_REPOSITORY_SSH_PRIVATE_KEY",
			"RWR_VAR_REPOSITORY_INIT-FILE",
		} {
			_ = os.Unsetenv(key)
		}
	}()

	// Capture the environment a spawned command actually receives.
	envPath := filepath.Join(t.TempDir(), "env.txt")
	if err := system.RunCommand(types.Command{
		Exec:    "/usr/bin/env",
		LogName: envPath,
	}, false); err != nil {
		t.Fatalf("RunCommand: %v", err)
	}

	raw, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("reading captured environment: %v", err)
	}
	env := string(raw)

	if strings.Contains(env, fakeToken) {
		t.Error("the GitHub token reached a spawned command's environment")
	}
	if strings.Contains(env, fakeKey) {
		t.Error("the SSH private key reached a spawned command's environment")
	}
	if strings.Contains(env, "RWR_VAR_REPOSITORY_GH_API_TOKEN") {
		t.Error("RWR_VAR_REPOSITORY_GH_API_TOKEN was exported")
	}
	if strings.Contains(env, "RWR_VAR_REPOSITORY_SSH_PRIVATE_KEY") {
		t.Error("RWR_VAR_REPOSITORY_SSH_PRIVATE_KEY was exported")
	}

	// Non-secret config must still be exported - that is the point of the feature.
	if !strings.Contains(env, "RWR_VAR_REPOSITORY_INIT-FILE") {
		t.Error("non-secret config values should still be exported to commands")
	}
}

// A blueprint template must not be able to render the credentials into a file it
// controls the path of.
func TestTemplatesCannotRenderCredentials(t *testing.T) {
	variables := types.Variables{
		Flags: types.Flags{
			GHAPIToken: fakeToken,
			SSHKey:     fakeKey,
		},
	}

	rendered, err := helpers.ResolveTemplate(
		[]byte("token={{ .Flags.ghAPIToken }} key={{ .Flags.sshKey }}"),
		variables,
	)
	if err != nil {
		// Erroring is an acceptable outcome; leaking is not.
		if strings.Contains(err.Error(), fakeToken) || strings.Contains(err.Error(), fakeKey) {
			t.Fatalf("template error leaked a credential: %v", err)
		}
		return
	}

	out := string(rendered)
	if strings.Contains(out, fakeToken) {
		t.Errorf("template rendered the GitHub token: %s", out)
	}
	if strings.Contains(out, fakeKey) {
		t.Errorf("template rendered the SSH key: %s", out)
	}
}

// A script that legitimately needs a token can be given one, without the tree
// handing out everything else.
func TestSpawnedCommandsReceiveOptedInCredentials(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /usr/bin/env")
	}

	viper.Reset()
	defer viper.Reset()
	defer types.SetExposedCredentials(nil)

	viper.Set("repository.gh_api_token", fakeToken)
	viper.Set("repository.ssh_private_key", fakeKey)

	types.SetExposedCredentials([]string{"gh_api_token"})

	initConfig := &types.InitConfig{}
	initConfig.Variables.UserDefined = map[string]interface{}{}
	if err := setUserDefinedAndEnvVariables(initConfig); err != nil {
		t.Fatalf("setUserDefinedAndEnvVariables: %v", err)
	}
	defer func() {
		_ = os.Unsetenv("RWR_VAR_REPOSITORY_GH_API_TOKEN")
		_ = os.Unsetenv("RWR_VAR_REPOSITORY_SSH_PRIVATE_KEY")
	}()

	envPath := filepath.Join(t.TempDir(), "env.txt")
	if err := system.RunCommand(types.Command{
		Exec:    "/usr/bin/env",
		LogName: envPath,
	}, false); err != nil {
		t.Fatalf("RunCommand: %v", err)
	}

	raw, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("reading captured environment: %v", err)
	}
	env := string(raw)

	if !strings.Contains(env, fakeToken) {
		t.Error("an opted-in token should reach the command that needs it")
	}
	// Opting into one credential must not hand over the other.
	if strings.Contains(env, fakeKey) {
		t.Error("the ssh key was exposed without being opted into")
	}
}
