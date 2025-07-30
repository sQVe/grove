//go:build !integration
// +build !integration

package create

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/testutils"
)

// BenchmarkPathGenerator_GeneratePath benchmarks path generation for worktrees.
// BENCHMARK STRATEGY: Measures the performance of path generation across different branch name patterns
// Tests include simple names, complex names with hyphens, and very long names to identify
// performance bottlenecks in path sanitization and collision detection algorithms.
func BenchmarkPathGenerator_GeneratePath(b *testing.B) {
	// Create helper using proper benchmark context
	helper := testutils.NewUnitTestHelper(b).WithCleanFilesystem()
	generator := NewPathGenerator()

	branchNames := []string{
		"feature-simple-branch",
		"bugfix-complex-branch-name-with-many-parts",
		"feature-api-v2-improvements",
		"hotfix-critical-security-fix-2024",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		testDir := helper.GetUniqueTestPath(fmt.Sprintf("benchmark-test-%d", i)) // Fresh dir each time to avoid collisions
		branchName := branchNames[i%len(branchNames)]
		_, err := generator.GeneratePath(branchName, testDir)
		if err != nil {
			b.Fatalf("GeneratePath failed: %v", err)
		}
	}
}

// BenchmarkPathGenerator_GeneratePathWithCollisions benchmarks path generation with collision handling.
// COLLISION TESTING STRATEGY: Measures performance when paths already exist and require suffix generation
// Pre-creates directories that match the target path to force the collision resolution algorithm
// to find alternative paths with numeric suffixes (e.g., feature-branch-1, feature-branch-2, etc.)
func BenchmarkPathGenerator_GeneratePathWithCollisions(b *testing.B) {
	// Create helper using proper benchmark context
	helper := testutils.NewUnitTestHelper(b).WithCleanFilesystem()
	generator := NewPathGenerator()

	// Create 10 collision attempts to test the collision resolution
	branchName := "feature-benchmark-test"

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		testDir := helper.GetUniqueTestPath(fmt.Sprintf("benchmark-collision-test-%d", i))

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
