package credentials

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/types"
)

// A `bw:` source reads a value from the personal vault through the Bitwarden
// CLI (`bw`), which must be installed and unlocked - `bw unlock` exports
// BW_SESSION, and every `bw get` call in that shell decrypts without further
// prompts. rwr never handles the master password and never stores vault
// material beyond the resolved credential value, which the registry already
// keeps out of logs and out of blueprint reach until exposeCredentials.
//
// The source covers values only: password, username, uri, notes, totp, and
// custom fields. Files (attachments) have no credential-shaped reading and are
// the job of a scripts blueprint - see the bitwarden example tree.

// bwTimeout bounds one CLI call. `bw` talks to the network and, when it is
// unconfigured, to the operator; both must fail rather than stall a run, so
// stdin stays closed and the context ends the process.
const bwTimeout = 30 * time.Second

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
	spec, err := parseBitwardenSource(source)
	if err != nil {
		log.Warnf("Credential source unusable: %v", err)
		return "", false
	}
	value, err := bwFetch(spec)
	if err != nil {
		// bw diagnostics name items and sessions, never the secret itself.
		log.Warnf("Credential source %s yielded no value: %v", source, err)
		return "", false
	}
	if value == "" {
		log.Warnf("Credential source %s yielded an empty value", source)
		return "", false
	}
	log.Debugf("Credential %s resolved: %s", source, types.Redact(value))
	return value, true
}

// bwFetch is the seam tests swap for a fake CLI; production shells out to bw.
var bwFetch = fetchWithCLI

func fetchWithCLI(spec bwSource) (string, error) {
	if _, err := exec.LookPath("bw"); err != nil {
		return "", fmt.Errorf("the Bitwarden CLI (bw) is not on PATH - install it (https://bitwarden.com/help/cli/) or drop this source: %w", err)
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
	// bw prints the value followed by exactly one newline; the value itself
	// may legitimately end (or begin) with newlines, so remove the CLI's one,
	// not every trailing one.
	out = strings.TrimSuffix(out, "\n")
	out = strings.TrimSuffix(out, "\r")
	return out, nil
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
	ctx, cancel := context.WithTimeout(context.Background(), bwTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bw", args...) // #nosec G204 -- the binary is fixed and args are operator-declared source specs rendered as discrete argv elements; no shell is involved
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
		return fmt.Errorf("%w (the vault is locked: run `bw unlock` and export BW_SESSION before running rwr)", err)
	case strings.Contains(message, "not logged in"), strings.Contains(message, "logged out"), strings.Contains(message, "username required"):
		return fmt.Errorf("%w (log in with `bw login`, then `bw unlock` and export BW_SESSION)", err)
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
