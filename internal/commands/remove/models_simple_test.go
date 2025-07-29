package remove

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveOptions_ValidateSimple(t *testing.T) {
	tests := []struct {
		name    string
		options RemoveOptions
		wantErr bool
	}{
		{
			name:    "valid options",
			options: RemoveOptions{Force: true, Days: 30},
			wantErr: false,
		},
		{
			name:    "invalid days",
			options: RemoveOptions{Days: -1},
			wantErr: true,
		},
		{
			name:    "conflicting flags",
			options: RemoveOptions{Force: true, DryRun: true},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.options.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBulkCriteria_ValidateSimple(t *testing.T) {
	tests := []struct {
		name     string
		criteria BulkCriteria
		wantErr  bool
	}{
		{
			name:     "valid merged",
			criteria: BulkCriteria{Merged: true, DaysOld: 30},
			wantErr:  false,
		},
		{
			name:     "no criteria",
			criteria: BulkCriteria{DaysOld: 30},
			wantErr:  true,
		},
		{
			name:     "multiple criteria",
			criteria: BulkCriteria{Merged: true, Stale: true, DaysOld: 30},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.criteria.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSafetyReport_Methods(t *testing.T) {
	report := SafetyReport{Path: "/test"}

	assert.False(t, report.HasWarnings())
	assert.Empty(t, report.WarningsText())

	report.AddWarning("Warning 1")
	assert.True(t, report.HasWarnings())
	assert.Equal(t, "Warning 1", report.WarningsText())

	report.AddWarning("Warning 2")
	assert.Equal(t, "Warning 1\nWarning 2", report.WarningsText())
}

func TestRemoveResults_Methods(t *testing.T) {
	results := RemoveResults{
		Removed: []string{"/path1", "/path2"},
		Skipped: []RemoveSkip{{Path: "/path3", Reason: "test"}},
		Failed:  []RemoveFailure{{Path: "/path4", Error: errors.New("test")}},
	}

	assert.True(t, results.HasResults())
	assert.Equal(t, 4, results.TotalProcessed())
}

func TestRemoveSummary_SuccessRate(t *testing.T) {
	summary := RemoveSummary{Total: 10, Removed: 7}
	assert.Equal(t, 70.0, summary.SuccessRate())

	summary = RemoveSummary{Total: 0, Removed: 0}
	assert.Equal(t, 0.0, summary.SuccessRate())
}
