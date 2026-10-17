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
// the user's chosen option. Cancellation and read errors are never choices.
func PromptUserChoice(prompt string, options []string, defaultOption string) (string, error) {
	choice := defaultOption
	if reporting.SupportsInlinePrompts() {
		form := huh.NewForm(huh.NewGroup(huh.NewSelect[string]().Title(prompt).
			Options(huh.NewOptions(options...)...).Value(&choice)))
		if err := reporting.RunForm(form); err != nil {
			return "", err
		}
		return choice, nil
	}
	// The raw input loop is used only by the headless fallback.
	err := reporting.WithTerminal(func() error {
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
	if err != nil {
		return "", err
	}
	return choice, nil
}
