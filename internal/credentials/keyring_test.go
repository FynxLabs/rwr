package credentials

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// EmptyKeyring is the stand-in every test outside this package reaches for, so
// its two answers are the contract: nothing is ever found, nothing is ever
// saved. A test that got a value back from it would be reading the machine.
func TestEmptyKeyringFindsAndSavesNothing(t *testing.T) {
	var ring Keyring = EmptyKeyring{}

	value, err := ring.Get("gh_api_token")
	assert.Empty(t, value)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrKeyringNotFound))

	assert.Error(t, ring.Set("gh_api_token", "ghp_whatever"))
}

// FromKeyring turns the empty ring into the "not found" the precedence chain
// expects, rather than surfacing the backend error.
func TestFromKeyringWithEmptyRing(t *testing.T) {
	defer SetRingForTest(EmptyKeyring{})()

	value, ok := FromKeyring("gh_api_token")
	assert.False(t, ok)
	assert.Empty(t, value)
}

// The restore func has to put the real keyring back, or one test that swaps
// the ring silently disarms every test that runs after it.
func TestSetRingForTestRestores(t *testing.T) {
	original := Ring

	restore := SetRingForTest(EmptyKeyring{})
	assert.NotEqual(t, original, Ring)

	restore()
	assert.Equal(t, original, Ring)
}
