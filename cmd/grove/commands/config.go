package commands

import (
	"strings"
)

// isValidConfigKey validates that key is in grove.* namespace
func isValidConfigKey(key string) bool {
	if key == "" {
		return false
	}
	return strings.HasPrefix(strings.ToLower(key), "grove.")
}

// isValidBooleanValue validates boolean configuration values
func isValidBooleanValue(value string) bool {
	if value == "" {
		return false
	}
	lower := strings.ToLower(value)
	validValues := []string{"true", "false", "yes", "no", "on", "off", "1", "0"}
	for _, valid := range validValues {
		if lower == valid {
			return true
		}
	}
	return false
}

// isMultiValueKey returns true if the key supports multiple values
func isMultiValueKey(key string) bool {
	return strings.EqualFold(key, "grove.convert.preserveignored")
}

// getConfigCompletions returns completion suggestions for config keys
func getConfigCompletions(toComplete string) []string {
	allKeys := []string{"grove.plain", "grove.debug", "grove.convert.preserveIgnored"}
	var completions []string
	for _, key := range allKeys {
		if strings.HasPrefix(key, toComplete) {
			completions = append(completions, key)
		}
	}
	return completions
}

// getBooleanCompletions returns completion suggestions for boolean values
func getBooleanCompletions(toComplete string) []string {
	booleans := []string{"true", "false"}
	var completions []string
	for _, b := range booleans {
		if strings.HasPrefix(b, toComplete) {
			completions = append(completions, b)
		}
	}
	return completions
}
