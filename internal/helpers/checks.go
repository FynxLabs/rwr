package helpers

// Contains reports whether item is present in the given string slice.
func Contains(slice []string, item string) bool {
	for _, value := range slice {
		if value == item {
			return true
		}
	}
	return false
}
