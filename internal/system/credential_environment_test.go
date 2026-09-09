package system

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestCommandEnvironmentOnlyExportsExposedCredentials(t *testing.T) {
	types.RegisterCredentials(nil)
	types.SetExposedCredentials(nil)
	t.Cleanup(func() { types.RegisterCredentials(nil); types.SetExposedCredentials(nil) })
	t.Setenv("RWR_CRED_HIDDEN", "inherited-secret")
	t.Setenv("RWR_CRED_BW_SESSION", "inherited-session")
	t.Setenv("RWR_SETTING", "preserved")
	types.SetCredentialValue("bw_session", "registry-session")
	for _, exposed := range []bool{false, true} {
		if exposed {
			types.SetExposedCredentials([]string{"bw_session"})
		}
		command := &exec.Cmd{}
		setupCommandEnvironment(command, types.Command{Variables: map[string]string{"RWR_CRED_HIDDEN": "injected-secret", "RWR_CRED_BW_SESSION": "injected-session"}})
		values := map[string]string{}
		for _, entry := range command.Env {
			key, value, _ := strings.Cut(entry, "=")
			values[key] = value
		}
		if _, ok := values["RWR_CRED_HIDDEN"]; ok {
			t.Fatal("unexposed credential inherited")
		}
		if values["RWR_SETTING"] != "preserved" {
			t.Fatal("ordinary RWR environment lost")
		}
		want := ""
		if exposed {
			want = "registry-session"
		}
		if values["RWR_CRED_BW_SESSION"] != want {
			t.Fatal("credential bypassed export registry")
		}
	}
}
