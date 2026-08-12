package uninstall

import (
	"runtime"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/state"
	"github.com/fynxlabs/rwr/internal/status"
	"github.com/fynxlabs/rwr/internal/types"
)

func serviceEntry(name string, identity map[string]string) state.Entry {
	merged := map[string]string{"name": name}
	for k, v := range identity {
		merged[k] = v
	}
	return state.Entry{
		Processor: types.BlueprintTypeServices,
		Action:    "enable", OK: true, Outcome: "ok",
		Identity: merged,
	}
}

// Every platform rwr provisions services on has a reversal, taken from the
// same tool the services processor applies with. macOS had none: uninstall
// skipped every service and said "not enabled or not queryable", even though
// the processor has had a full launchd backend the whole time.
func TestDisableServiceCommandPerPlatform(t *testing.T) {
	t.Parallel()

	cmd, ok := disableServiceCommand("nginx", serviceEntry("nginx", nil))

	switch runtime.GOOS {
	case types.OSLinux:
		if !ok {
			t.Fatal("linux has no service reversal")
		}
		if cmd.Exec != "systemctl" {
			t.Errorf("exec = %q, want systemctl", cmd.Exec)
		}
		if strings.Join(cmd.Args, " ") != "disable --now nginx" {
			t.Errorf("args = %v", cmd.Args)
		}
	case types.OSDarwin:
		if !ok {
			t.Fatal("darwin has no service reversal")
		}
		if cmd.Exec != "launchctl" {
			t.Errorf("exec = %q, want launchctl", cmd.Exec)
		}
		// The inverse of the processor's enable action, which loads the
		// daemon from /Library/LaunchDaemons.
		if strings.Join(cmd.Args, " ") != "unload -w /Library/LaunchDaemons/nginx.plist" {
			t.Errorf("args = %v", cmd.Args)
		}
	default:
		if ok {
			t.Errorf("%s claims a service reversal it does not have", runtime.GOOS)
		}
	}
}

// A daemon can be installed somewhere other than /Library/LaunchDaemons, and
// the apply records where it put it. The recorded target wins over the
// convention.
func TestDisableServiceCommandPrefersTheRecordedTarget(t *testing.T) {
	if runtime.GOOS != types.OSDarwin {
		t.Skip("launchd only")
	}
	t.Parallel()

	entry := serviceEntry("nginx", map[string]string{"target": "/Users/me/Library/LaunchAgents/nginx.plist"})

	cmd, ok := disableServiceCommand("nginx", entry)
	if !ok {
		t.Fatal("darwin has no service reversal")
	}
	if got := cmd.Args[len(cmd.Args)-1]; got != "/Users/me/Library/LaunchAgents/nginx.plist" {
		t.Errorf("plist = %q, want the recorded target", got)
	}
}

// The three reasons a service is left alone are different situations and the
// operator has to be able to tell them apart. They were one sentence -
// "not enabled or not queryable" - which on macOS always meant the third and
// never said so.
//
// The presence query is injected rather than asked of the host, or the test
// could only assert "some reason": which one comes back depends on the
// service manager the machine happens to run, and being able to reach each of
// them is the point.
func TestReverseServiceSkipReasonsAreDistinct(t *testing.T) {
	if runtime.GOOS != types.OSLinux && runtime.GOOS != types.OSDarwin {
		t.Skipf("%s has no service reversal; covered by the platform test", runtime.GOOS)
	}

	tests := []struct {
		name  string
		state status.Presence
		want  string
	}{
		{name: "already disabled", state: status.Absent, want: "already disabled"},
		{name: "unqueryable", state: status.Unknown, want: "cannot query the service; not disabling it blind"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer SetServiceStateForTest(func(string) status.Presence { return tt.state })()

			reason, err := reverseService(serviceEntry("nginx", nil))
			if err != nil {
				t.Fatalf("a skip must not be an error: %v", err)
			}
			if reason != tt.want {
				t.Errorf("reason = %q, want %q", reason, tt.want)
			}
		})
	}
}

// A recorded entry with no name at all is its own case, and it must not reach
// the service manager.
func TestReverseServiceWithoutANameSkips(t *testing.T) {
	defer SetServiceStateForTest(func(string) status.Presence {
		t.Error("queried the service manager for an entry with no name")
		return status.Unknown
	})()

	reason, err := reverseService(serviceEntry("", nil))
	if err != nil {
		t.Fatal(err)
	}
	if reason != "no recorded service name" {
		t.Errorf("reason = %q, want the missing-name reason", reason)
	}
}

// A platform with no reversal says so, and says which platform, rather than
// implying the service was already disabled. It must decide that before it
// consults the service manager, since there is nothing it could do with the
// answer.
func TestReverseServiceNamesAPlatformWithNoReversal(t *testing.T) {
	if runtime.GOOS == types.OSLinux || runtime.GOOS == types.OSDarwin {
		t.Skipf("%s has a service reversal", runtime.GOOS)
	}
	defer SetServiceStateForTest(func(string) status.Presence {
		t.Error("queried the service manager on a platform with no reversal")
		return status.Unknown
	})()

	reason, err := reverseService(serviceEntry("nginx", nil))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(reason, "no service reversal on "+runtime.GOOS) {
		t.Errorf("reason = %q, want it to name the platform", reason)
	}
}
