package credentials

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// A bw: source reads a personal vault value. RWR installs, logs in and unlocks
// the CLI when needed. The session is held in memory and passed only to bw.

// bwTimeout bounds one CLI call. `bw` talks to the network and, when it is
// unconfigured, to the operator; both must fail rather than stall a run, so
// stdin stays closed and the context ends the process.
const bwTimeout = 30 * time.Second

// ErrBitwardenNotInstalled distinguishes a missing optional tool from a vault error.
var ErrBitwardenNotInstalled = errors.New("bitwarden CLI (bw) is not installed")

// bwSource is a parsed `bw:<item>[/<key>]` source. The empty key means
// password, the common case; Field carries the custom field name when Key is
// "field".
type bwSource struct {
	Item  string
	Key   string
	Field string
}

// parseBitwardenSource mirrors types.validateCredentialSource, which rejects
// anything this cannot handle at decode time. An item name may itself contain
// slashes: only a final segment that names a key is the key.
func parseBitwardenSource(source string) (bwSource, error) {
	rest := strings.TrimPrefix(source, "bw:")
	slash := strings.LastIndex(rest, "/")
	if slash < 0 {
		if rest == "" {
			return bwSource{}, fmt.Errorf("source %q names no Bitwarden item", source)
		}
		return bwSource{Item: rest, Key: "password"}, nil
	}
	item, key := rest[:slash], rest[slash+1:]
	if item == "" {
		return bwSource{}, fmt.Errorf("source %q names no Bitwarden item", source)
	}
	if field, isField := strings.CutPrefix(key, "field:"); isField {
		if field == "" {
			return bwSource{}, fmt.Errorf("source %q names no custom field", source)
		}
		return bwSource{Item: item, Key: "field", Field: field}, nil
	}
	switch key {
	case "password", "username", "uri", "notes", "totp":
		return bwSource{Item: item, Key: key}, nil
	default:
		// Not a key word, so the whole rest is the item name.
		return bwSource{Item: rest, Key: "password"}, nil
	}
}

// FromBitwarden resolves one `bw:` source. Like FromKeyring it never fails the
// run directly: an unusable source is "no value here" and precedence moves to
// the next declared source, with the reason in the log - a machine without the
// CLI unlocked can still resolve the credential from keyring or prompt.
func FromBitwarden(source string) (string, bool) {
	value, err := readBitwarden(source)
	return value, err == nil && value != ""
}

func readBitwarden(source string) (string, error) {
	spec, err := parseBitwardenSource(source)
	if err != nil {
		log.Warnf("Credential source unusable: %v", err)
		return "", err
	}
	value, err := bwFetch(spec)
	if err != nil {
		// bw diagnostics name items and sessions, never the secret itself.
		log.Warnf("Credential source %s yielded no value: %v", source, err)
		return "", err
	}
	if value == "" {
		log.Warnf("Credential source %s yielded an empty value", source)
		return "", nil
	}
	log.Debugf("Credential %s resolved: %s", source, types.Redact(value))
	return value, nil
}

// bwFetch is the seam tests swap for a fake CLI; production shells out to bw.
var bwFetch = fetchWithCLI

func fetchWithCLI(spec bwSource) (string, error) {
	if _, err := exec.LookPath("bw"); err != nil {
		return "", fmt.Errorf("%w: %w", ErrBitwardenNotInstalled, err)
	}
	args, err := bwArgs(spec)
	if err != nil {
		return "", err
	}
	out, err := runBW(args)
	if err != nil {
		return "", bwHint(err)
	}
	if spec.Key == "field" {
		// The field value comes from the item JSON, not the CLI's stdout, so
		// it is the stored value exactly - newlines included.
		return bwFieldValue(out, spec.Field, spec.Item)
	}
	// bw prints the value followed by exactly one line feed, and nothing
	// else; the value itself may legitimately end (or begin) with newlines,
	// or with a bare carriage return, so remove that one delimiter only.
	return strings.TrimSuffix(out, "\n"), nil
}

// bwArgs builds the `bw get` invocation for a source. Custom fields have no
// dedicated subcommand, so they come from the full item JSON.
func bwArgs(spec bwSource) ([]string, error) {
	switch spec.Key {
	case "field":
		return []string{"get", "item", spec.Item}, nil
	case "password", "username", "uri", "notes", "totp":
		return []string{"get", spec.Key, spec.Item}, nil
	default:
		// types.validateCredentialSource rejects other keys at decode time.
		return nil, fmt.Errorf("unsupported Bitwarden key %q", spec.Key)
	}
}

// runBW runs one bw command to completion. Stdin is deliberately not attached:
// a locked vault or a first-run wizard would otherwise sit waiting for input
// rwr can never give, hanging the run instead of failing it.
func runBW(args []string) (string, error) {
	ctx, cancel := context.WithTimeout(system.RunContext(), bwTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bw", args...) // #nosec G204 -- the binary is fixed and args are operator-declared source specs rendered as discrete argv elements; no shell is involved
	cmd.Env = bitwardenEnv()
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("bw %s did not finish within %s", strings.Join(args[:min(2, len(args))], " "), bwTimeout)
	}
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return "", errors.New(message)
	}
	return stdout.String(), nil
}

// bwHint turns the classes of bw failure an operator can act on into the action
// that fixes them. Anything unrecognized passes through unchanged.
func bwHint(err error) error {
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "vault is locked"):
		return fmt.Errorf("%w (the vault is locked; RWR can unlock it in an interactive run, or use BW_SESSION for unattended runs)", err)
	case strings.Contains(message, "not logged in"), strings.Contains(message, "logged out"), strings.Contains(message, "username required"):
		return fmt.Errorf("%w (RWR can log in and unlock Bitwarden in an interactive run)", err)
	case strings.Contains(message, "not found"), strings.Contains(message, "no item"):
		return fmt.Errorf("%w (no vault item matches - `bw list items --search <name>` shows what the CLI can see)", err)
	default:
		return err
	}
}

// bwFieldValue extracts one custom field's value from `bw get item` JSON.
// Field names are operator-defined, so the match is exact and case-sensitive;
// both text and hidden fields resolve, since a credential is exactly what the
// hidden type exists for.
func bwFieldValue(itemJSON, fieldName, item string) (string, error) {
	var itemStruct struct {
		Fields []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"fields"`
	}
	if err := json.Unmarshal([]byte(itemJSON), &itemStruct); err != nil {
		return "", fmt.Errorf("bw returned item %q in a form rwr cannot parse as JSON: %v", item, err)
	}
	for _, field := range itemStruct.Fields {
		if field.Name == fieldName {
			return field.Value, nil
		}
	}
	return "", fmt.Errorf("item %q has no custom field named %q", item, fieldName)
}

// bitwardenSession is scoped to credential resolution; never exported globally.
var bitwardenSession string

func bitwardenEnv() []string {
	env := os.Environ()
	if bitwardenSession == "" {
		return env
	}
	filtered := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if !strings.HasPrefix(entry, "BW_SESSION=") {
			filtered = append(filtered, entry)
		}
	}
	return append(filtered, "BW_SESSION="+bitwardenSession)
}

func bitwardenNeedsAuth(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "vault is locked") || strings.Contains(message, "not logged in") || strings.Contains(message, "logged out") || strings.Contains(message, "username required") || strings.Contains(message, "invalid session")
}
