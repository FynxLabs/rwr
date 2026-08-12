// Package credentials resolves the credentials an init file declares: each one
// is looked up through its ordered sources (env:<VAR>, keyring, prompt) before
// any processor runs, and the resolved values are handed to the types registry,
// which gates every path out to a blueprint.
package credentials

import (
	"errors"
	"fmt"

	zkeyring "github.com/zalando/go-keyring"
)

// keyringService namespaces rwr's entries in the OS keyring; the account within
// the service is the credential name.
const keyringService = "rwr"

// Keyring is the slice of the OS keyring rwr uses. It is an interface so tests
// run keyring-less with a fake - CI has no Secret Service, and desktop keyring
// behavior varies too much to assert against.
type Keyring interface {
	// Get returns the stored value, or ErrKeyringNotFound when the entry does
	// not exist, or another error when the backend is unavailable.
	Get(name string) (string, error)
	Set(name, value string) error
}

// ErrKeyringNotFound reports an entry that does not exist, as distinct from a
// backend that cannot be reached at all.
var ErrKeyringNotFound = errors.New("keyring entry not found")

// Ring is the keyring in use. Tests swap in a fake; production keeps the OS
// keyring (Secret Service / macOS Keychain / Windows Credential Manager).
var Ring Keyring = osKeyring{}

// EmptyKeyring answers "not found" for every name and refuses every save. It
// is what a test wants whenever the code under test happens to consult the
// keyring but the test is not about the keyring.
type EmptyKeyring struct{}

// Get implements Keyring.
func (EmptyKeyring) Get(string) (string, error) { return "", ErrKeyringNotFound }

// Set implements Keyring.
func (EmptyKeyring) Set(string, string) error { return errors.New("keyring unavailable") }

// SetRingForTest installs a keyring for the duration of a test and returns the
// restore func, in the same shape as system.SetProvidersForTest.
//
// It exists because reading the real keyring from a test is not merely flaky,
// it is a disclosure: the "no token available" cases in the SSH key tests
// found the developer's own GitHub token in the macOS Keychain and testify
// printed the value into the terminal on the assertion failure. CI never saw
// it because GitHub runners have no keyring backend, so the gate could not
// catch it either.
//
// Any test that exercises a code path reaching FromKeyring must call this.
//
//	defer credentials.SetRingForTest(credentials.EmptyKeyring{})()
func SetRingForTest(ring Keyring) (restore func()) {
	previous := Ring
	Ring = ring
	return func() { Ring = previous }
}

type osKeyring struct{}

func (osKeyring) Get(name string) (string, error) {
	value, err := zkeyring.Get(keyringService, name)
	if errors.Is(err, zkeyring.ErrNotFound) {
		return "", ErrKeyringNotFound
	}
	return value, err
}

func (osKeyring) Set(name, value string) error {
	return zkeyring.Set(keyringService, name, value)
}

// FromKeyring reads a credential from the keyring. A missing entry or an
// unavailable backend both yield "not found": the keyring is never
// load-bearing on read - precedence just moves to the next source.
func FromKeyring(name string) (string, bool) {
	value, err := Ring.Get(name)
	if err != nil || value == "" {
		return "", false
	}
	return value, true
}

// SaveToKeyring persists a credential value. Unlike reads, an explicit save
// against an unavailable backend fails loudly: silently not persisting is how
// an operator ends up re-prompted forever, or worse, reaching for a plaintext
// file.
func SaveToKeyring(name, value string) error {
	if err := Ring.Set(name, value); err != nil {
		return fmt.Errorf("saving %s to the OS keyring: %w", name, err)
	}
	return nil
}
