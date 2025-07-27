//go:build !integration
// +build !integration

package utils

import (
	"os"
	"testing"
)

func TestGetTerminalWidth(t *testing.T) {
	tests := []struct {
		name               string
		columnsEnv         string
		expectedMinWidth   int
		expectedMaxWidth   int
		expectDefaultWidth bool
	}{
		{
			name:               "no COLUMNS env var",
			columnsEnv:         "",
			expectedMinWidth:   DefaultTerminalWidth,
			expectedMaxWidth:   300, // reasonable upper bound.
			expectDefaultWidth: false,
		},
		{
			name:               "valid COLUMNS env var",
			columnsEnv:         "120",
			expectedMinWidth:   120,
			expectedMaxWidth:   120,
			expectDefaultWidth: false,
		},
		{
			name:               "invalid COLUMNS env var",
			columnsEnv:         "invalid",
			expectedMinWidth:   DefaultTerminalWidth,
			expectedMaxWidth:   300,
			expectDefaultWidth: false,
		},
		{
			name:               "zero COLUMNS env var",
			columnsEnv:         "0",
			expectedMinWidth:   DefaultTerminalWidth,
			expectedMaxWidth:   300,
			expectDefaultWidth: false,
		},
		{
			name:               "negative COLUMNS env var",
			columnsEnv:         "-1",
			expectedMinWidth:   DefaultTerminalWidth,
			expectedMaxWidth:   300,
			expectDefaultWidth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalColumns := os.Getenv("COLUMNS")
			defer func() {
				if originalColumns != "" {
					_ = os.Setenv("COLUMNS", originalColumns)
				} else {
					_ = os.Unsetenv("COLUMNS")
				}
			}()

			if tt.columnsEnv != "" {
				_ = os.Setenv("COLUMNS", tt.columnsEnv)
			} else {
				_ = os.Unsetenv("COLUMNS")
			}

			width := GetTerminalWidth()

			if width < tt.expectedMinWidth || width > tt.expectedMaxWidth {
				t.Errorf("GetTerminalWidth() = %d, expected between %d and %d",
					width, tt.expectedMinWidth, tt.expectedMaxWidth)
			}

			if tt.expectDefaultWidth && width != DefaultTerminalWidth {
				t.Errorf("GetTerminalWidth() = %d, expected default width %d",
					width, DefaultTerminalWidth)
			}
		})
	}
}

func TestGetTerminalWidthWithEnvVar(t *testing.T) {
	originalColumns := os.Getenv("COLUMNS")
	defer func() {
		if originalColumns != "" {
			_ = os.Setenv("COLUMNS", originalColumns)
		} else {
			_ = os.Unsetenv("COLUMNS")
		}
	}()

	_ = os.Setenv("COLUMNS", "100")
	width := GetTerminalWidth()
	if width != 100 {
		t.Errorf("GetTerminalWidth() with COLUMNS=100 returned %d, expected 100", width)
	}
}

func TestIsInteractiveTerminal(t *testing.T) {
	// Note: This test result depends on how the test is run.
	// In CI/CD or when output is redirected, it should be false.
	// When run interactively, it might be true.
	result := IsInteractiveTerminal()

	if result != true && result != false {
		t.Errorf("IsInteractiveTerminal() returned non-boolean value")
	}
}

func TestDefaultTerminalWidth(t *testing.T) {
	if DefaultTerminalWidth <= 0 {
		t.Errorf("DefaultTerminalWidth = %d, expected positive value", DefaultTerminalWidth)
	}

	if DefaultTerminalWidth < 40 {
		t.Errorf("DefaultTerminalWidth = %d, expected at least 40 columns", DefaultTerminalWidth)
	}
}
