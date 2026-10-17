package system

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"charm.land/huh/v2"
	"charm.land/log/v2"
	"github.com/freehold-digital/rwr/internal/reporting"
)

// PromptUserChoice displays an interactive selection prompt and returns
// the user's chosen option, falling back to defaultOption on invalid input.
func PromptUserChoice(prompt string, options []string, defaultOption string) string {
	choice := defaultOption
	if reporting.SupportsInlinePrompts() {
		form := huh.NewForm(huh.NewGroup(huh.NewSelect[string]().Title(prompt).
			Options(huh.NewOptions(options...)...).Value(&choice)))
		if err := reporting.RunForm(form); err != nil {
			return defaultOption
		}
		return choice
	}
	// The raw input loop is used only by the headless fallback.
	_ = reporting.WithTerminal(func() error { //nolint:errcheck // a failed read falls back to the default
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Printf("%s [%s]: ", prompt, strings.Join(options, "/"))
			input, err := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "" {
				choice = defaultOption
				return err
			}

			for _, option := range options {
				if strings.EqualFold(input, option) {
					choice = option
					return nil
				}
			}
			if err != nil {
				// Stdin is closed or unreadable; looping would spin forever.
				choice = defaultOption
				return err
			}

			log.Warnf("Invalid input. Please choose from the available options.")
		}
	})
	return choice
}
