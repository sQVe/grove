package list

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sqve/grove/internal/commands/shared"
	"github.com/sqve/grove/internal/errors"
)

func TestValidateListOptions(t *testing.T) {
	tests := []struct {
		name        string
		options     *ListOptions
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid default options",
			options: &ListOptions{
				Sort:      SortByActivity,
				StaleDays: shared.DefaultStaleDays,
			},
			expectError: false,
		},
		{
			name: "dirty and clean flags together",
			options: &ListOptions{
				Sort:      SortByActivity,
				DirtyOnly: true,
				CleanOnly: true,
				StaleDays: shared.DefaultStaleDays,
			},
			expectError: true,
			errorMsg:    "cannot use --dirty and --clean flags together",
		},
		{
			name: "dirty and stale flags together",
			options: &ListOptions{
				Sort:      SortByActivity,
				DirtyOnly: true,
				StaleOnly: true,
				StaleDays: shared.DefaultStaleDays,
			},
			expectError: true,
			errorMsg:    "cannot use --dirty and --stale flags together",
		},
		{
			name: "clean and stale flags together",
			options: &ListOptions{
				Sort:      SortByActivity,
				CleanOnly: true,
				StaleOnly: true,
				StaleDays: shared.DefaultStaleDays,
			},
			expectError: true,
			errorMsg:    "cannot use --clean and --stale flags together",
		},
		{
			name: "invalid sort option",
			options: &ListOptions{
				Sort:      "invalid",
				StaleDays: shared.DefaultStaleDays,
			},
			expectError: true,
			errorMsg:    "invalid sort option",
		},
		{
			name: "negative stale days",
			options: &ListOptions{
				Sort:      SortByActivity,
				StaleOnly: true,
				StaleDays: -1,
			},
			expectError: true,
			errorMsg:    "stale days must be positive",
		},
		{
			name: "porcelain with verbose",
			options: &ListOptions{
				Sort:      SortByActivity,
				Porcelain: true,
				Verbose:   true,
				StaleDays: shared.DefaultStaleDays,
			},
			expectError: true,
			errorMsg:    "cannot use --porcelain and --verbose flags together",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateListOptions(tt.options)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)

				var groveErr *errors.GroveError
				require.True(t, errors.As(err, &groveErr))
				assert.Equal(t, errors.ErrCodeConfigInvalid, groveErr.Code)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestListSortOptions(t *testing.T) {
	tests := []struct {
		name   string
		option ListSortOption
		valid  bool
	}{
		{"activity sort", SortByActivity, true},
		{"name sort", SortByName, true},
		{"status sort", SortByStatus, true},
		{"invalid sort", "invalid", false},
		{"empty sort", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &ListOptions{
				Sort:      tt.option,
				StaleDays: shared.DefaultStaleDays,
			}

			err := validateListOptions(options)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestNewListCmd(t *testing.T) {
	cmd := NewListCmd()

	assert.Equal(t, "list", cmd.Use)
	assert.Equal(t, "List all worktrees with status information", cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	// Verify flags exist
	flags := cmd.Flags()
	assert.NotNil(t, flags.Lookup("sort"))
	assert.NotNil(t, flags.Lookup("verbose"))
	assert.NotNil(t, flags.Lookup("porcelain"))
	assert.NotNil(t, flags.Lookup("dirty"))
	assert.NotNil(t, flags.Lookup("stale"))
	assert.NotNil(t, flags.Lookup("clean"))
	assert.NotNil(t, flags.Lookup("days"))
}
