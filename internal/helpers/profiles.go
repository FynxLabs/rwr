package helpers

import (
	"slices"
)

// ShouldInclude determines if an item should be included based on active profiles.
// An item should be included if:
// 1. It has no profiles specified (base item - always included)
// 2. At least one of its profiles matches an active profile
// 3. "all" is in active profiles (special case to include everything).
// Profile-gated items are opt-in: when no active profiles are given, only base
// items apply.
func ShouldInclude(itemProfiles []string, activeProfiles []string) bool {
	// If no profiles are specified for the item, it's a base item - always include
	if len(itemProfiles) == 0 {
		return true
	}

	// "all" is a special case: include everything regardless of item profiles
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

// CountGated counts the items a profile-filtered run would skip: entries with
// one or more profiles when their profile is not active. Processors use this to
// warn about profile-scoped work a run leaves undone, and bootstrap uses it to
// keep the run-once marker honest.
func CountGated[T interface{ GetProfiles() []string }](items []T, activeProfiles []string) int {
	if slices.Contains(activeProfiles, "all") {
		return 0
	}
	count := 0
	for _, item := range items {
		if !ShouldInclude(item.GetProfiles(), activeProfiles) {
			count++
		}
	}
	return count
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
