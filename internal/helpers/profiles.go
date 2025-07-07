package helpers

import (
	"slices"
)

// ShouldInclude determines if an item should be included based on active profiles.
// An item should be included if:
// 1. It has no profiles specified (base item - always included)
// 2. At least one of its profiles matches an active profile
// 3. "all" is in active profiles (special case to include everything)
func ShouldInclude(itemProfiles []string, activeProfiles []string) bool {
	// If no profiles are specified for the item, it's a base item - always include
	if len(itemProfiles) == 0 {
		return true
	}

	// If no active profiles are specified, include ALL items (permissive default behavior)
	if len(activeProfiles) == 0 {
		return true
	}

	// If "all" is in active profiles, include everything
	if slices.Contains(activeProfiles, "all") {
		return true
	}

	// Check if any of the item's profiles match active profiles
	for _, itemProfile := range itemProfiles {
		if slices.Contains(activeProfiles, itemProfile) {
			return true
		}
	}

	return false
}

// FilterByProfiles filters a slice of items that have a Profiles field based on active profiles.
// This is a generic function that works with any type that has a Profiles []string field.
func FilterByProfiles[T interface{ GetProfiles() []string }](items []T, activeProfiles []string) []T {
	if len(activeProfiles) == 0 {
		// If no profiles specified, include ALL items (base behavior should be permissive)
		// This allows RWR to work without requiring profile knowledge
		return items
	}

	// If "all" is in active profiles, return everything
	if slices.Contains(activeProfiles, "all") {
		return items
	}

	// Filter based on profile matching
	var filtered []T
	for _, item := range items {
		if ShouldInclude(item.GetProfiles(), activeProfiles) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// GetUniqueProfiles extracts all unique profile names from a slice of items.
// This is useful for discovering available profiles in a configuration.
func GetUniqueProfiles[T interface{ GetProfiles() []string }](items []T) []string {
	profileSet := make(map[string]bool)

	for _, item := range items {
		for _, profile := range item.GetProfiles() {
			if profile != "" {
				profileSet[profile] = true
			}
		}
	}

	var profiles []string
	for profile := range profileSet {
		profiles = append(profiles, profile)
	}

	// Sort profiles for consistent output
	slices.Sort(profiles)
	return profiles
}

// ValidateProfiles checks if all provided active profiles exist in the available profiles.
// Returns a slice of invalid profiles that don't exist in the configuration.
func ValidateProfiles(activeProfiles []string, availableProfiles []string) []string {
	var invalid []string

	for _, activeProfile := range activeProfiles {
		// Skip validation for the special "all" profile
		if activeProfile == "all" {
			continue
		}

		if !slices.Contains(availableProfiles, activeProfile) {
			invalid = append(invalid, activeProfile)
		}
	}

	return invalid
}
