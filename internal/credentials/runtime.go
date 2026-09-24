package credentials

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/freehold-digital/rwr/internal/system"
	"github.com/freehold-digital/rwr/internal/types"
)

// Resolver lives for a run, caches unavailable connections, and never acquires
// credentials interactively. Setup may attach a session it explicitly opened.
type Resolver struct {
	Config      *types.InitConfig
	sessions    map[string]Session
	unavailable map[string]error
	setupErrors map[string]error
}

func NewResolver(c *types.InitConfig) *Resolver {
	return &Resolver{Config: c, sessions: map[string]Session{}, unavailable: map[string]error{}, setupErrors: map[string]error{}}
}
func (r *Resolver) Close() {
	for _, s := range r.sessions {
		_ = s.Close() //nolint:errcheck // Session cleanup is best effort; values are dropped regardless.
	}
	clear(r.sessions)
	clear(r.unavailable)
}
func (r *Resolver) Attach(name string, s Session) {
	if old := r.sessions[name]; old != nil {
		_ = old.Close() //nolint:errcheck // Replacing a run-owned session discards it.
	}
	r.sessions[name] = s
	delete(r.unavailable, name)
}
func (r *Resolver) Connection(name string) (types.CredentialConnection, error) {
	for _, c := range r.Config.CredentialProviders {
		if c.Name == name {
			return c, nil
		}
	}
	return types.CredentialConnection{}, fmt.Errorf("unknown credential connection %q", name)
}
func (r *Resolver) Session(ctx context.Context, name string) (Session, error) {
	if system.IsDryRun() {
		return nil, &Unavailable{Denied}
	}
	if s := r.Config.Variables.Flags.Selection; s != nil && s.DenyProviders {
		return nil, &Unavailable{Denied}
	}
	if s := r.sessions[name]; s != nil {
		return s, nil
	}
	if err := r.unavailable[name]; err != nil {
		return nil, err
	}
	c, err := r.Connection(name)
	if err != nil {
		return nil, err
	}
	s, err := OpenProvider(ctx, c, SetupOptions{})
	if err != nil {
		r.unavailable[name] = err
		return nil, err
	}
	r.sessions[name] = s
	return s, nil
}
func (r *Resolver) Read(ctx context.Context, name string) (string, error) {
	if system.IsDryRun() {
		return "", &Unavailable{Denied}
	}
	var spec *types.CredentialSpec
	for i := range r.Config.Credentials {
		if r.Config.Credentials[i].Name == name {
			spec = &r.Config.Credentials[i]
			break
		}
	}
	if spec == nil {
		if !types.IsManagedCredential(name) {
			return "", fmt.Errorf("undeclared credential %q", name)
		}
		if value, ok := types.CredentialValue(name); ok && value != "" {
			return value, nil
		}
		return "", &Unavailable{Missing}
	}
	sources := spec.Sources
	if len(sources) == 0 {
		sources = defaultSources(name)
	}
	for _, source := range sources {
		switch {
		case strings.HasPrefix(source, "env:"):
			if value := os.Getenv(strings.TrimPrefix(source, "env:")); value != "" {
				return value, nil
			}
		case source == "keyring":
			// Runtime keyring access is deliberately opt-in; desktop backends can
			// unlock or prompt on Get. Custom noninteractive backends advertise it.
			if ring, ok := Ring.(NoninteractiveKeyring); ok {
				keys := []string{name}
				if len(spec.References) > 0 {
					keys = nil
					for _, ref := range spec.References {
						c, err := r.Connection(ref.Connection)
						if err == nil {
							keys = append(keys, "v1/"+connectionIdentity(c)+"/"+name)
						}
					}
				}
				for _, key := range keys {
					if value, err := ring.GetNoninteractive(key); err == nil && value != "" {
						return value, nil
					}
				}
			}
		case strings.HasPrefix(source, "bw:"):
			if s := r.Config.Variables.Flags.Selection; s != nil && s.DenyProviders {
				continue
			}
			value, err := readBitwarden(source)
			if err == nil && value != "" {
				return value, nil
			}
		case source == "prompt": // setup owns prompting; never fall through to it here
		default:
			return "", fmt.Errorf("unknown credential source")
		}
	}
	for _, ref := range spec.References {
		session, err := r.Session(ctx, ref.Connection)
		if err != nil {
			continue
		}
		value, err := session.ReadSecret(ctx, ref)
		if err == nil && value != "" {
			return value, nil
		}
	}
	return "", &Unavailable{Missing}
}

// NoninteractiveKeyring is optional: a generic Get cannot promise no UI.
type NoninteractiveKeyring interface{ GetNoninteractive(string) (string, error) }

// Setup caches explicit Skip and failed acquisition so another entry cannot
// restart prompts as a fallback. A subsequent RWR invocation gets a fresh scope.
func (r *Resolver) Setup(ctx context.Context, name string, o SetupOptions) (Session, error) {
	if selection := r.Config.Variables.Flags.Selection; selection != nil && selection.DenyProviders {
		return nil, &Unavailable{Denied}
	}
	if s := r.sessions[name]; s != nil {
		return s, nil
	}
	if err := r.setupErrors[name]; err != nil {
		return nil, err
	}
	c, err := r.Connection(name)
	if err != nil {
		return nil, err
	}
	s, err := OpenProvider(ctx, c, o)
	if err != nil {
		r.setupErrors[name] = err
		return nil, err
	}
	r.Attach(name, s)
	return s, nil
}

// ReadSetup permits an explicitly declared credential prompt only inside a
// native setup task. It never persists the answer implicitly.
func (r *Resolver) ReadSetup(ctx context.Context, name string) (string, error) {
	value, err := r.Read(ctx, name)
	if err == nil {
		return value, nil
	}
	if system.IsDryRun() || !r.Config.Variables.Flags.Interactive || !stdinIsTerminal() {
		return "", err
	}
	if selection := r.Config.Variables.Flags.Selection; selection != nil && selection.DenyProviders {
		return "", err
	}
	for _, spec := range r.Config.Credentials {
		if spec.Name != name {
			continue
		}
		sources := spec.Sources
		if len(sources) == 0 {
			sources = defaultSources(name)
		}
		for _, source := range sources {
			if source == "prompt" {
				value, promptErr := promptForCredential(name, spec.Description)
				if promptErr != nil {
					return "", system.ErrCancelled
				}
				if value == "" {
					return "", &Unavailable{Missing}
				}
				return value, nil
			}
		}
	}
	return "", err
}

func (r *Resolver) String() string   { return "credential runtime (private)" }
func (r *Resolver) GoString() string { return r.String() }
