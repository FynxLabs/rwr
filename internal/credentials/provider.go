package credentials

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/freehold-digital/rwr/internal/types"
)

type Availability string

const (
	Missing         Availability = "missing"
	Locked          Availability = "locked"
	Unauthenticated Availability = "unauthenticated"
	Denied          Availability = "denied"
	Skipped         Availability = "skipped"
)

type Unavailable struct{ State Availability }

func (e *Unavailable) Error() string { return "credential provider is " + string(e.State) }
func IsUnavailable(err error) bool   { var e *Unavailable; return errors.As(err, &e) }

// SetupOptions permits acquisition only during explicit provider setup.
// Optional capabilities keep future adapters independent of Bitwarden.
type SetupOptions struct {
	Install      bool
	Authenticate bool
	Interactive  bool
}
type Provider interface {
	Open(context.Context, types.CredentialConnection, SetupOptions) (Session, error)
}
type Session interface {
	ReadSecret(context.Context, types.CredentialReference) (string, error)
	Close() error
}
type AttachmentSession interface {
	ReadAttachment(context.Context, types.CredentialAttachment) ([]byte, error)
	ReplaceAttachment(context.Context, types.CredentialAttachment, []byte) error
}
type ProviderFactory func() Provider

var providerMu sync.RWMutex
var providerFactories = map[string]ProviderFactory{"bitwarden": func() Provider { return &bitwardenProvider{} }}

// RegisterProvider installs an adapter factory and returns a restoration closure
// for embedded callers/tests. Each open gets independent state.
func RegisterProvider(id string, factory ProviderFactory) func() {
	providerMu.Lock()
	old, exists := providerFactories[id]
	providerFactories[id] = factory
	providerMu.Unlock()
	return func() {
		providerMu.Lock()
		defer providerMu.Unlock()
		if exists {
			providerFactories[id] = old
		} else {
			delete(providerFactories, id)
		}
	}
}
func OpenProvider(ctx context.Context, c types.CredentialConnection, o SetupOptions) (Session, error) {
	providerMu.RLock()
	f := providerFactories[c.Provider]
	providerMu.RUnlock()
	if f == nil {
		return nil, fmt.Errorf("unsupported credential provider %q", c.Provider)
	}
	return f().Open(ctx, c, o)
}
