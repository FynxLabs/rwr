package processors

import "strings"

// containsString checks if a string contains a substring.
func containsString(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
