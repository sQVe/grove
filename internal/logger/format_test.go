package logger

import "testing"

func TestStepFormat(t *testing.T) {
	tests := []struct {
		name    string
		step    int
		total   int
		message string
		want    string
	}{
		{
			name:    "first of three",
			step:    1,
			total:   3,
			message: "Fetching remote",
			want:    "Step 1/3: Fetching remote",
		},
		{
			name:    "middle step",
			step:    2,
			total:   4,
			message: "Creating worktree",
			want:    "Step 2/4: Creating worktree",
		},
		{
			name:    "last of ten",
			step:    10,
			total:   10,
			message: "Done",
			want:    "Step 10/10: Done",
		},
		{
			name:    "single step",
			step:    1,
			total:   1,
			message: "Processing",
			want:    "Step 1/1: Processing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StepFormat(tt.step, tt.total, tt.message)
			if got != tt.want {
				t.Errorf("StepFormat(%d, %d, %q) = %q, want %q",
					tt.step, tt.total, tt.message, got, tt.want)
			}
		})
	}
}
