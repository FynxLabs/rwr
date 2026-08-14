package types

import "strings"

// minRedactableSecret is the shortest secret worth substring-replacing in a log
// line. See LogArgs.
const minRedactableSecret = 4

type Command struct {
	Exec        string            `mapstructure:"exec" yaml:"exec" json:"exec" toml:"exec"`
	Variables   map[string]string `mapstructure:"variables,omitempty" yaml:"variables,omitempty" json:"variables,omitempty" toml:"variables,omitempty"`
	Args        []string          `mapstructure:"args,omitempty" yaml:"args,omitempty" json:"args,omitempty" toml:"args,omitempty"`
	LogName     string            `mapstructure:"logName,omitempty" yaml:"logName,omitempty" json:"logName,omitempty" toml:"logName,omitempty"`
	AsUser      string            `mapstructure:"asUser,omitempty" yaml:"asUser,omitempty" json:"asUser,omitempty" toml:"asUser,omitempty"`
	Interactive bool              `mapstructure:"interactive" yaml:"interactive" json:"interactive" toml:"interactive"`
	Elevated    bool              `mapstructure:"elevated" yaml:"elevated" json:"elevated" toml:"elevated"`

	// Escalates marks a command rwr runs unprivileged that calls sudo itself.
	//
	// It is not the same as Elevated and cannot be derived from it. Homebrew
	// refuses to run as root, so rwr must not elevate it - but a cask install
	// shells out to sudo to write into /Applications. sudo reads that password
	// from /dev/tty, straight past the pipes rwr captured, so under the
	// dashboard the prompt is invisible and the run hangs on a password nobody
	// was asked for.
	//
	// Not a blueprint field: it is a property of the tool, declared by the
	// provider definition.
	Escalates bool `mapstructure:"-" yaml:"-" json:"-" toml:"-"`

	// Stdin is fed to the command on standard input. It is deliberately not a
	// blueprint field: it exists so secrets can be handed to a tool without
	// appearing in argv, where every local user can read them out of `ps` and
	// where the debug logger would print them. `chpasswd` is the reason it exists.
	Stdin string `mapstructure:"-" yaml:"-" json:"-" toml:"-"`

	// Secrets lists values that must be scrubbed from any log line describing this
	// command. Some tools accept a credential only as an argument (chocolatey's
	// --password, cargo's login token), so the value cannot always be moved to
	// Stdin - but it should still never be written to a log file. Callers that put
	// a credential in Args are expected to list it here.
	Secrets []string `mapstructure:"-" yaml:"-" json:"-" toml:"-"`
}

// LogArgs returns Args with every value in Secrets replaced, for logging.
func (c Command) LogArgs() []string {
	if len(c.Secrets) == 0 {
		return c.Args
	}

	safe := make([]string, len(c.Args))
	for i, arg := range c.Args {
		for _, secret := range c.Secrets {
			// Substring replacement on a very short secret would corrupt unrelated
			// arguments - a password of "1" would rewrite every digit in the argv.
			// Below this length the log damage outweighs the disclosure, and a
			// secret that short is not protecting anything anyway.
			if len(secret) < minRedactableSecret {
				continue
			}
			arg = strings.ReplaceAll(arg, secret, RedactedPlaceholder)
		}
		safe[i] = arg
	}
	return safe
}
