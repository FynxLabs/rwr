package helpers

// ResolveInteractive determines if an operation should be interactive.
// Blueprint-level setting takes priority over the global flag.
// If blueprintInteractive is nil, the global flag is used.
func ResolveInteractive(blueprintInteractive *bool, globalInteractive bool) bool {
	if blueprintInteractive != nil {
		return *blueprintInteractive
	}
	return globalInteractive
}
