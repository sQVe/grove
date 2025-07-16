package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "long version flag",
			args:     []string{"--version"},
			expected: "grove version v0.1.0",
		},
		{
			name:     "short version flag",
			args:     []string{"-v"},
			expected: "grove version v0.1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new command instance for each test
			cmd := rootCmd
			cmd.SetArgs(tt.args)
			
			// Capture output
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			
			// Execute command
			err := cmd.Execute()
			require.NoError(t, err)
			
			// Check output
			output := strings.TrimSpace(buf.String())
			assert.Equal(t, tt.expected, output)
		})
	}
}

func TestRootCommandHelp(t *testing.T) {
	cmd := rootCmd
	cmd.SetArgs([]string{"--help"})
	
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	
	err := cmd.Execute()
	require.NoError(t, err)
	
	output := buf.String()
	assert.Contains(t, output, "Grove transforms Git worktrees")
	assert.Contains(t, output, "Usage:")
	assert.Contains(t, output, "Available Commands:")
	assert.Contains(t, output, "Flags:")
}

func TestRootCommandDefault(t *testing.T) {
	cmd := rootCmd
	cmd.SetArgs([]string{})
	
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	
	err := cmd.Execute()
	require.NoError(t, err)
	
	output := buf.String()
	// When no arguments are provided, cobra shows help by default
	assert.Contains(t, output, "Grove transforms Git worktrees")
	assert.Contains(t, output, "Usage:")
	assert.Contains(t, output, "Available Commands:")
}