package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/fs"
)

func TestPreserveFilesToWorktree(t *testing.T) {
	t.Run("copies matching ignored files to destination", func(t *testing.T) {
		sourceDir := t.TempDir()
		destDir := t.TempDir()

		// Create a .env file in source
		envPath := filepath.Join(sourceDir, ".env")
		if err := os.WriteFile(envPath, []byte("SECRET=value"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		// Create .gitignore that ignores .env
		gitignorePath := filepath.Join(sourceDir, ".gitignore")
		if err := os.WriteFile(gitignorePath, []byte(".env\n"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		// Set default patterns
		patterns := []string{".env"}

		result, err := PreserveFilesToWorktree(sourceDir, destDir, patterns, []string{".env"}, nil)
		if err != nil {
			t.Fatalf("PreserveFilesToWorktree failed: %v", err)
		}

		if len(result.Copied) != 1 {
			t.Errorf("Expected 1 copied file, got %d", len(result.Copied))
		}

		destFile := filepath.Join(destDir, ".env")
		content, err := os.ReadFile(destFile) //nolint:gosec
		if err != nil {
			t.Fatalf("Failed to read copied file: %v", err)
		}
		if string(content) != "SECRET=value" {
			t.Errorf("File content mismatch: got %q", string(content))
		}
	})

	t.Run("skips files that already exist in destination", func(t *testing.T) {
		sourceDir := t.TempDir()
		destDir := t.TempDir()

		// Create .env in source
		envPath := filepath.Join(sourceDir, ".env")
		if err := os.WriteFile(envPath, []byte("SOURCE_VALUE"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		// Create .env in destination (already exists)
		destEnvPath := filepath.Join(destDir, ".env")
		if err := os.WriteFile(destEnvPath, []byte("DEST_VALUE"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		patterns := []string{".env"}

		result, err := PreserveFilesToWorktree(sourceDir, destDir, patterns, []string{".env"}, nil)
		if err != nil {
			t.Fatalf("PreserveFilesToWorktree failed: %v", err)
		}

		if len(result.Skipped) != 1 {
			t.Errorf("Expected 1 skipped file, got %d", len(result.Skipped))
		}

		content, _ := os.ReadFile(destEnvPath) //nolint:gosec
		if string(content) != "DEST_VALUE" {
			t.Errorf("Destination file was overwritten: got %q", string(content))
		}
	})

	t.Run("only copies files that match patterns", func(t *testing.T) {
		sourceDir := t.TempDir()
		destDir := t.TempDir()

		// Create files that match and don't match
		if err := os.WriteFile(filepath.Join(sourceDir, ".env"), []byte("match"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(sourceDir, "other.txt"), []byte("nomatch"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		patterns := []string{".env"}
		ignoredFiles := []string{".env", "other.txt"}

		result, err := PreserveFilesToWorktree(sourceDir, destDir, patterns, ignoredFiles, nil)
		if err != nil {
			t.Fatalf("PreserveFilesToWorktree failed: %v", err)
		}

		if len(result.Copied) != 1 || result.Copied[0] != testEnvFile {
			t.Errorf("Expected only .env to be copied, got %v", result.Copied)
		}

		// Verify other.txt was NOT copied
		if _, err := os.Stat(filepath.Join(destDir, "other.txt")); err == nil {
			t.Error("other.txt should not have been copied")
		}
	})

	t.Run("handles nested directory paths", func(t *testing.T) {
		sourceDir := t.TempDir()
		destDir := t.TempDir()

		// Create nested .env file
		nestedDir := filepath.Join(sourceDir, "config")
		if err := os.MkdirAll(nestedDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		envPath := filepath.Join(nestedDir, ".env.local")
		if err := os.WriteFile(envPath, []byte("NESTED=true"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		patterns := []string{".env.local"}
		ignoredFiles := []string{"config/.env.local"}

		result, err := PreserveFilesToWorktree(sourceDir, destDir, patterns, ignoredFiles, nil)
		if err != nil {
			t.Fatalf("PreserveFilesToWorktree failed: %v", err)
		}

		if len(result.Copied) != 1 {
			t.Errorf("Expected 1 copied file, got %d", len(result.Copied))
		}

		// Verify nested file was copied with directory structure
		destFile := filepath.Join(destDir, "config", ".env.local")
		if _, err := os.Stat(destFile); err != nil {
			t.Errorf("Nested file should exist: %v", err)
		}
	})

	t.Run("returns empty result when no files match", func(t *testing.T) {
		sourceDir := t.TempDir()
		destDir := t.TempDir()

		patterns := []string{".env"}
		ignoredFiles := []string{"other.txt"} // Doesn't match .env pattern

		result, err := PreserveFilesToWorktree(sourceDir, destDir, patterns, ignoredFiles, nil)
		if err != nil {
			t.Fatalf("PreserveFilesToWorktree failed: %v", err)
		}

		if len(result.Copied) != 0 || len(result.Skipped) != 0 {
			t.Errorf("Expected empty result, got copied=%d skipped=%d", len(result.Copied), len(result.Skipped))
		}
	})

	t.Run("handles wildcard patterns", func(t *testing.T) {
		sourceDir := t.TempDir()
		destDir := t.TempDir()

		// Create files matching *.local.json
		if err := os.WriteFile(filepath.Join(sourceDir, "config.local.json"), []byte("{}"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(sourceDir, "settings.local.json"), []byte("{}"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		patterns := []string{"*.local.json"}
		ignoredFiles := []string{"config.local.json", "settings.local.json"}

		result, err := PreserveFilesToWorktree(sourceDir, destDir, patterns, ignoredFiles, nil)
		if err != nil {
			t.Fatalf("PreserveFilesToWorktree failed: %v", err)
		}

		if len(result.Copied) != 2 {
			t.Errorf("Expected 2 copied files, got %d: %v", len(result.Copied), result.Copied)
		}
	})

	t.Run("excludes files matching exclude patterns", func(t *testing.T) {
		sourceDir := t.TempDir()
		destDir := t.TempDir()

		// Create .env in root (should be preserved)
		if err := os.WriteFile(filepath.Join(sourceDir, ".env"), []byte("ROOT=true"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		// Create .env in node_modules (should be excluded)
		nodeModulesDir := filepath.Join(sourceDir, "node_modules", "some-package")
		if err := os.MkdirAll(nodeModulesDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(nodeModulesDir, ".env"), []byte("EXCLUDED=true"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		patterns := []string{".env"}
		ignoredFiles := []string{".env", "node_modules/some-package/.env"}
		excludePatterns := []string{"node_modules"}

		result, err := PreserveFilesToWorktree(sourceDir, destDir, patterns, ignoredFiles, excludePatterns)
		if err != nil {
			t.Fatalf("PreserveFilesToWorktree failed: %v", err)
		}

		// Only root .env should be copied, not the one in node_modules
		if len(result.Copied) != 1 {
			t.Errorf("Expected 1 copied file, got %d: %v", len(result.Copied), result.Copied)
		}
		if len(result.Copied) > 0 && result.Copied[0] != testEnvFile {
			t.Errorf("Expected root .env to be copied, got %v", result.Copied)
		}

		// Verify node_modules/.env was NOT copied
		excludedFile := filepath.Join(destDir, "node_modules", "some-package", ".env")
		if _, err := os.Stat(excludedFile); err == nil {
			t.Error("node_modules/.env should not have been copied")
		}
	})
}

func TestFindIgnoredFilesInWorktree(t *testing.T) {
	// This function wraps git ls-files, hard to unit test without real git repo
	// Integration tests cover this in create_test.go
}

func TestGetPreservePatternsForCreate(t *testing.T) {
	t.Run("uses TOML config when present", func(t *testing.T) {
		tmpDir := t.TempDir()

		tomlContent := `[preserve]
patterns = [".custom", "*.secret"]
`
		tomlPath := filepath.Join(tmpDir, ".grove.toml")
		if err := os.WriteFile(tomlPath, []byte(tomlContent), 0o644); err != nil { //nolint:gosec
			t.Fatal(err)
		}

		patterns := config.GetMergedPreservePatterns(tmpDir)

		if len(patterns) != 2 {
			t.Errorf("Expected 2 patterns, got %d: %v", len(patterns), patterns)
		}
		if patterns[0] != ".custom" || patterns[1] != "*.secret" {
			t.Errorf("Unexpected patterns: %v", patterns)
		}
	})

	t.Run("falls back to defaults when no config", func(t *testing.T) {
		tmpDir := t.TempDir()
		// No .grove.toml, no git config

		patterns := config.GetMergedPreservePatterns(tmpDir)

		// Should get defaults
		if len(patterns) == 0 {
			t.Error("Expected default patterns, got none")
		}
		// Check that .env is in defaults
		found := false
		for _, p := range patterns {
			if p == ".env" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected .env in default patterns")
		}
	})
}
