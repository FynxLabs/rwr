package system

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/log"
)

func PromptUserChoice(prompt string, options []string, defaultOption string) string {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [%s]: ", prompt, strings.Join(options, "/"))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			return defaultOption
		}

		for _, option := range options {
			if strings.EqualFold(input, option) {
				return option
			}
		}

		log.Warnf("Invalid input. Please choose from the available options.")
	}
}
