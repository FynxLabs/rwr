package cmd

import (
	"fmt"
	"strings"
)

// normalizeProfileArgs rejoins comma lists split by the shell before Cobra
// interprets their continuation as positional arguments. Other arguments and
// everything after -- retain their original meaning.
func normalizeProfileArgs(args []string) ([]string, error) {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return append(out, args[i:]...), nil
		}
		var value string
		switch {
		case arg == "--profile" || arg == "-p":
			if i+1 == len(args) || strings.HasPrefix(args[i+1], "-") {
				return nil, fmt.Errorf("%s requires a profile name", arg)
			}
			i++
			value = args[i]
		case strings.HasPrefix(arg, "--profile="):
			value = strings.TrimPrefix(arg, "--profile=")
		case strings.HasPrefix(arg, "-p") && !strings.HasPrefix(arg, "--"):
			value = strings.TrimPrefix(strings.TrimPrefix(arg, "-p"), "=")
		default:
			out = append(out, arg)
			continue
		}
		value = strings.TrimSpace(value)
		for i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") &&
			(strings.HasSuffix(value, ",") || strings.HasPrefix(strings.TrimSpace(args[i+1]), ",")) {
			i++
			value += strings.TrimSpace(args[i])
		}
		profiles := strings.Split(value, ",")
		for j := range profiles {
			profiles[j] = strings.TrimSpace(profiles[j])
			if profiles[j] == "" {
				return nil, fmt.Errorf("--profile contains an empty name; supply names such as --profile nvidia,desktop")
			}
		}
		out = append(out, "--profile="+strings.Join(profiles, ","))
	}
	return out, nil
}
