//go:build !linux

package credentials

import "runtime"

func (k osKeyring) GetNoninteractive(name string) (string, error) {
	// Windows Credential Manager reads do not unlock a desktop keychain.
	// macOS's generic Keychain API may show an access dialog: decline runtime
	// access until an adapter with interaction explicitly disabled is provided.
	if runtime.GOOS == "windows" {
		return k.Get(name)
	}
	return "", ErrKeyringUnavailable
}
