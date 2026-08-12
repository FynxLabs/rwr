package system

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/reporting"
)

// PromptUserChoice displays an interactive selection prompt and returns
// the user's chosen option, falling back to defaultOption on invalid input.
func PromptUserChoice(prompt string, options []string, defaultOption string) string {
	choice := defaultOption
	// Through the terminal lease: headless this runs directly; under the TUI
	// the dashboard suspends around it. A bare stdin read while the dashboard
	// owns the terminal freezes the run.
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
