package commands

import (
	"testing"
)

func TestValidateConfigKey(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		valid bool
	}{
		{
			name:  "valid grove.plain key",
			key:   "grove.plain",
			valid: true,
		},
		{
			name:  "valid grove.debug key",
			key:   "grove.debug",
			valid: true,
		},
		{
			name:  "valid grove.convert.preserve key",
			key:   "grove.convert.preserve",
			valid: true,
		},
		{
			name:  "invalid user.name key",
			key:   "user.name",
			valid: false,
		},
		{
			name:  "invalid core.autocrlf key",
			key:   "core.autocrlf",
			valid: false,
		},
		{
			name:  "empty key",
			key:   "",
			valid: false,
		},
		{
			name:  "case insensitive grove key",
			key:   "Grove.Plain",
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidConfigKey(tt.key); got != tt.valid {
				t.Errorf("isValidConfigKey(%q) = %v, want %v", tt.key, got, tt.valid)
			}
		})
	}
}

func TestValidateBooleanValue(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{
			name:  "true",
			value: "true",
			valid: true,
		},
		{
			name:  "false",
			value: "false",
			valid: true,
		},
		{
			name:  "yes",
			value: "yes",
			valid: true,
		},
		{
			name:  "no",
			value: "no",
			valid: true,
		},
		{
			name:  "on",
			value: "on",
			valid: true,
		},
		{
			name:  "off",
			value: "off",
			valid: true,
		},
		{
			name:  "1",
			value: "1",
			valid: true,
		},
		{
			name:  "0",
			value: "0",
			valid: true,
		},
		{
			name:  "case insensitive TRUE",
			value: "TRUE",
			valid: true,
		},
		{
			name:  "case insensitive False",
			value: "False",
			valid: true,
		},
		{
			name:  "invalid maybe",
			value: "maybe",
			valid: false,
		},
		{
			name:  "invalid 2",
			value: "2",
			valid: false,
		},
		{
			name:  "empty value",
			value: "",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidBooleanValue(tt.value); got != tt.valid {
				t.Errorf("isValidBooleanValue(%q) = %v, want %v", tt.value, got, tt.valid)
			}
		})
	}
}

func TestIsMultiValueKey(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		multiValue bool
	}{
		{
			name:       "grove.convert.preserve is multi-value",
			key:        "grove.convert.preserve",
			multiValue: true,
		},
		{
			name:       "grove.plain is single-value",
			key:        "grove.plain",
			multiValue: false,
		},
		{
			name:       "grove.debug is single-value",
			key:        "grove.debug",
			multiValue: false,
		},
		{
			name:       "case insensitive multi-value",
			key:        "Grove.Convert.Preserve",
			multiValue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMultiValueKey(tt.key); got != tt.multiValue {
				t.Errorf("isMultiValueKey(%q) = %v, want %v", tt.key, got, tt.multiValue)
			}
		})
	}
}

func TestGetConfigCompletions(t *testing.T) {
	tests := []struct {
		name       string
		toComplete string
		want       []string
	}{
		{
			name:       "empty completion shows all keys",
			toComplete: "",
			want:       []string{"grove.plain", "grove.debug", "grove.convert.preserve"},
		},
		{
			name:       "partial grove completion",
			toComplete: "grove.p",
			want:       []string{"grove.plain"},
		},
		{
			name:       "partial grove.c completion",
			toComplete: "grove.c",
			want:       []string{"grove.convert.preserve"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getConfigCompletions(tt.toComplete)
			if len(got) != len(tt.want) {
				t.Errorf("getConfigCompletions(%q) returned %d items, want %d", tt.toComplete, len(got), len(tt.want))
				return
			}
			for i, want := range tt.want {
				if i >= len(got) || got[i] != want {
					t.Errorf("getConfigCompletions(%q)[%d] = %q, want %q", tt.toComplete, i, got[i], want)
				}
			}
		})
	}
}

func TestGetBooleanCompletions(t *testing.T) {
	tests := []struct {
		name       string
		toComplete string
		want       []string
	}{
		{
			name:       "empty completion shows boolean values",
			toComplete: "",
			want:       []string{"true", "false"},
		},
		{
			name:       "partial true completion",
			toComplete: "t",
			want:       []string{"true"},
		},
		{
			name:       "partial false completion",
			toComplete: "f",
			want:       []string{"false"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getBooleanCompletions(tt.toComplete)
			if len(got) != len(tt.want) {
				t.Errorf("getBooleanCompletions(%q) returned %d items, want %d", tt.toComplete, len(got), len(tt.want))
				return
			}
			for i, want := range tt.want {
				if i >= len(got) || got[i] != want {
					t.Errorf("getBooleanCompletions(%q)[%d] = %q, want %q", tt.toComplete, i, got[i], want)
				}
			}
		})
	}
}
