//go:build !integration
// +build !integration

package create

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkPathGenerator_GeneratePath benchmarks path generation for worktrees.
func BenchmarkPathGenerator_GeneratePath(b *testing.B) {
	generator := NewPathGenerator()

	branchNames := []string{
		"feature-simple-branch",
		"bugfix-complex-branch-name-with-many-parts",
		"feature-api-v2-improvements",
		"hotfix-critical-security-fix-2024",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		testDir := b.TempDir() // Fresh dir each time to avoid collisions
		branchName := branchNames[i%len(branchNames)]
		_, err := generator.GeneratePath(branchName, testDir)
		if err != nil {
			b.Fatalf("GeneratePath failed: %v", err)
		}
	}
}

// BenchmarkPathGenerator_GeneratePathWithCollisions benchmarks path generation with collision handling.
func BenchmarkPathGenerator_GeneratePathWithCollisions(b *testing.B) {
	generator := NewPathGenerator()

	// Create 10 collision attempts to test the collision resolution
	branchName := "feature-benchmark-test"

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		testDir := b.TempDir()

		// Create some existing directories to simulate collisions
		basePath := filepath.Join(testDir, branchName)
		err := os.MkdirAll(basePath, 0o755)
		if err != nil {
			b.Fatalf("Failed to create collision directory: %v", err)
		}

		// Also create the first collision path
		collisionPath := fmt.Sprintf("%s-1", basePath)
		err = os.MkdirAll(collisionPath, 0o755)
		if err != nil {
			b.Fatalf("Failed to create collision directory: %v", err)
		}

		_, err = generator.GeneratePath(branchName, testDir)
		if err != nil {
			b.Fatalf("GeneratePath with collisions failed: %v", err)
		}
	}
}
