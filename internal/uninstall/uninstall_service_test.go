package uninstall

import (
	"runtime"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/state"
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
func TestReverseServiceSkipReasonsAreDistinct(t *testing.T) {
	if _, err := reverseService(serviceEntry("", nil)); err != nil {
		t.Fatal(err)
	}
	reason, err := reverseService(serviceEntry("", nil))
	if err != nil {
		t.Fatal(err)
	}
	if reason != "no recorded service name" {
		t.Errorf("missing name reason = %q", reason)
	}

	// A name no service manager knows: on a platform with a reversal that is
	// "already disabled" or an honest "cannot query"; on one without, it must
	// name the platform rather than blaming the service.
	reason, err = reverseService(serviceEntry("rwr-test-definitely-not-a-real-service", nil))
	if err != nil {
		t.Fatalf("reversing an absent service should skip, not fail: %v", err)
	}
	if reason == "" {
		t.Fatal("an unknown service was not skipped")
	}
	switch runtime.GOOS {
	case types.OSLinux, types.OSDarwin:
		if strings.Contains(reason, "no service reversal on") {
			t.Errorf("%s does have a service reversal, but reported %q", runtime.GOOS, reason)
		}
	default:
		if !strings.Contains(reason, "no service reversal on "+runtime.GOOS) {
			t.Errorf("reason = %q, want it to name the platform", reason)
		}
	}
}
