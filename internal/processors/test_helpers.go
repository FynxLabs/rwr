package processors

import "strings"

// Helper function to check if a string contains a substring
func containsString(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
