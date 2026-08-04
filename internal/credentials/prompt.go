package credentials

import (
	"fmt"
	"os"

	"charm.land/huh/v2"
	"charm.land/log/v2"
	"golang.org/x/term"
)

// stdinIsTerminal is a var so resolution tests can force the non-TTY path
// without detaching their own stdin.
var stdinIsTerminal = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// promptForCredential asks the operator for a credential value with masked
// input. The value never echoes; the description says what is being asked for
// and why.
var promptForCredential = func(name, description string) (string, error) {
	var value string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(fmt.Sprintf("Enter credential %q", name)).
				Description(description).
				EchoMode(huh.EchoModePassword).
				Value(&value).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("value cannot be empty")
					}
					return nil
				}),
		),
	)
	if err := form.Run(); err != nil {
		return "", fmt.Errorf("prompt for credential %q failed: %w", name, err)
	}
	return value, nil
}

// offerKeyringSave asks (default yes) whether to keep a just-prompted value in
// the OS keyring so the next run does not ask again. Declining is fine; a
// failed save is reported but does not fail the run - the value is already in
// hand.
var offerKeyringSave = func(name, value string) {
	confirm := true
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Save %q to the OS keyring?", name)).
				Description("Next run will read it from the keyring instead of asking.").
				Affirmative("Yes, save it").
				Negative("No, ask again next time").
				Value(&confirm),
		),
	)
	if err := form.Run(); err != nil || !confirm {
		return
	}
	if err := SaveToKeyring(name, value); err != nil {
		log.Warnf("Could not save %q to the keyring: %v", name, err)
	}
}
