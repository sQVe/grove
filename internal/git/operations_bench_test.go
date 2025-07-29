//go:build !integration
// +build !integration

package git

import (
	"context"
	"testing"
	"time"

	"github.com/sqve/grove/internal/testutils"
)

// BenchmarkGitOperations_CheckRepositorySafety benchmarks repository safety checks.
func BenchmarkGitOperations_CheckRepositorySafety(b *testing.B) {
	mockExec := testutils.NewMockGitExecutor()
	mockExec.SetSafeRepositoryState()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := CheckRepositorySafetyForConversion(mockExec)
		if err != nil {
			b.Fatalf("CheckRepositorySafetyForConversion failed: %v", err)
		}
	}
}

// BenchmarkGitOperations_CheckRepositorySafetyWithIssues benchmarks safety checks with issues.
func BenchmarkGitOperations_CheckRepositorySafetyWithIssues(b *testing.B) {
	mockExec := testutils.NewMockGitExecutor()
	mockExec.SetUnsafeRepositoryState()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, _ = CheckRepositorySafetyForConversion(mockExec)
		// Ignore error since we expect issues in this benchmark
	}
}

// BenchmarkGitOperations_SafetyCheckComponents benchmarks individual safety check components.
func BenchmarkGitOperations_SafetyCheckComponents(b *testing.B) {
	tests := []struct {
		name string
		fn   func(GitExecutor) ([]SafetyIssue, error)
	}{
		{"checkGitStatus", checkGitStatus},
		{"checkStashedChanges", checkStashedChanges},
		{"checkUntrackedFiles", checkUntrackedFiles},
		{"checkExistingWorktrees", checkExistingWorktrees},
		{"checkUnpushedCommits", checkUnpushedCommits},
		{"checkLocalOnlyBranches", checkLocalOnlyBranches},
		{"checkOngoingGitOperations", checkOngoingGitOperations},
	}
	
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			mockExec := testutils.NewMockGitExecutor()
			mockExec.SetSafeRepositoryState()
			
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				_, err := tt.fn(mockExec)
				if err != nil {
					b.Fatalf("%s failed: %v", tt.name, err)
				}
			}
		})
	}
}

// BenchmarkGitOperations_ParseGitStatusLine benchmarks git status parsing.
func BenchmarkGitOperations_ParseGitStatusLine(b *testing.B) {
	statusLines := []string{
		" M modified_file.txt",
		"A  added_file.txt", 
		"D  deleted_file.txt",
		"R  renamed_file.txt -> new_name.txt",
		"?? untracked_file.txt",
		"!! ignored_file.txt",
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		line := statusLines[i%len(statusLines)]
		parseGitStatusLine(line)
	}
}

// BenchmarkGitOperations_CountGitChanges benchmarks change counting.
func BenchmarkGitOperations_CountGitChanges(b *testing.B) {
	porcelainOutput := `
 M file1.txt
A  file2.txt
D  file3.txt
R  file4.txt -> file4_renamed.txt
?? untracked1.txt
?? untracked2.txt
!! ignored.txt
`
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		CountGitChanges(porcelainOutput)
	}
}

// BenchmarkGitOperations_WithTimeout benchmarks operations with context timeout.
func BenchmarkGitOperations_WithTimeout(b *testing.B) {
	mockExec := testutils.NewMockGitExecutor()
	mockExec.SetSafeRepositoryState()
	
	// Add a small delay to simulate real git commands
	mockExec.SetDelayedResponse("status --porcelain=v1", "", nil, time.Millisecond)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		
		// Simulate a safety check with timeout
		_, err := mockExec.ExecuteWithContext(ctx, "status", "--porcelain=v1")
		if err != nil && err != context.DeadlineExceeded {
			b.Fatalf("Operation failed: %v", err)
		}
		
		cancel()
	}
}