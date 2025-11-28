package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/fs"
)

func TestRunAddHooks(t *testing.T) {
	t.Run("runs single command successfully", func(t *testing.T) {
		workDir := t.TempDir()

		// Create a simple test file to prove we ran in the right directory
		commands := []string{"touch test-file.txt"}

		result := RunAddHooks(workDir, commands)

		if len(result.Succeeded) != 1 {
			t.Errorf("Expected 1 succeeded command, got %d", len(result.Succeeded))
		}

		// Verify file was created in workDir
		testFile := filepath.Join(workDir, "test-file.txt")
		if _, err := os.Stat(testFile); err != nil {
			t.Errorf("Expected test file to exist: %v", err)
		}
	})

	t.Run("runs multiple commands sequentially", func(t *testing.T) {
		workDir := t.TempDir()

		commands := []string{
			"touch first.txt",
			"touch second.txt",
		}

		result := RunAddHooks(workDir, commands)

		if len(result.Succeeded) != 2 {
			t.Errorf("Expected 2 succeeded commands, got %d", len(result.Succeeded))
		}

		// Verify both files exist
		if _, err := os.Stat(filepath.Join(workDir, "first.txt")); err != nil {
			t.Error("first.txt should exist")
		}
		if _, err := os.Stat(filepath.Join(workDir, "second.txt")); err != nil {
			t.Error("second.txt should exist")
		}
	})

	t.Run("stops on first failure", func(t *testing.T) {
		workDir := t.TempDir()

		commands := []string{
			"touch first.txt",
			"false", // This will fail
			"touch third.txt",
		}

		result := RunAddHooks(workDir, commands)

		if len(result.Succeeded) != 1 {
			t.Errorf("Expected 1 succeeded command, got %d", len(result.Succeeded))
		}

		if result.Failed == nil {
			t.Fatal("Expected a failed command")
		}

		if result.Failed.Command != "false" {
			t.Errorf("Expected 'false' to fail, got %q", result.Failed.Command)
		}

		// Verify third command was NOT run
		if _, err := os.Stat(filepath.Join(workDir, "third.txt")); err == nil {
			t.Error("third.txt should NOT exist (stopped on failure)")
		}
	})

	t.Run("returns empty result for empty commands", func(t *testing.T) {
		workDir := t.TempDir()

		result := RunAddHooks(workDir, nil)

		if len(result.Succeeded) != 0 || result.Failed != nil {
			t.Error("Expected empty result for no commands")
		}
	})

	t.Run("captures exit code on failure", func(t *testing.T) {
		workDir := t.TempDir()

		commands := []string{"exit 42"}

		result := RunAddHooks(workDir, commands)

		if result.Failed == nil {
			t.Fatal("Expected a failed command")
		}

		if result.Failed.ExitCode != 42 {
			t.Errorf("Expected exit code 42, got %d", result.Failed.ExitCode)
		}
	})

	t.Run("captures stdout and stderr on failure", func(t *testing.T) {
		workDir := t.TempDir()

		commands := []string{"echo 'out message'; echo 'err message' >&2; false"}

		result := RunAddHooks(workDir, commands)

		if result.Failed == nil {
			t.Fatal("Expected a failed command")
		}

		if result.Failed.Stdout == "" {
			t.Error("Expected stdout to be captured")
		}

		if result.Failed.Stderr == "" {
			t.Error("Expected stderr to be captured")
		}
	})

	t.Run("inherits environment variables", func(t *testing.T) {
		workDir := t.TempDir()

		_ = os.Setenv("GROVE_TEST_VAR", "test_value")
		defer func() { _ = os.Unsetenv("GROVE_TEST_VAR") }()

		commands := []string{"echo $GROVE_TEST_VAR > env-output.txt"}

		result := RunAddHooks(workDir, commands)

		if len(result.Succeeded) != 1 {
			t.Fatalf("Expected command to succeed, got failed: %v", result.Failed)
		}

		content, err := os.ReadFile(filepath.Join(workDir, "env-output.txt")) //nolint:gosec
		if err != nil {
			t.Fatal(err)
		}

		if string(content) != "test_value\n" {
			t.Errorf("Expected 'test_value', got %q", string(content))
		}
	})
}

func TestGetAddHooks(t *testing.T) {
	t.Run("returns hooks from TOML config", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .grove.toml with hooks
		tomlContent := `[hooks]
add = ["pnpm i", "pnpm build"]
`
		tomlPath := filepath.Join(tmpDir, ".grove.toml")
		if err := os.WriteFile(tomlPath, []byte(tomlContent), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		hooks := GetAddHooks(tmpDir)

		if len(hooks) != 2 {
			t.Fatalf("Expected 2 hooks, got %d", len(hooks))
		}
		if hooks[0] != "pnpm i" || hooks[1] != "pnpm build" {
			t.Errorf("Unexpected hooks: %v", hooks)
		}
	})

	t.Run("returns empty when no config", func(t *testing.T) {
		tmpDir := t.TempDir()
		// No .grove.toml

		hooks := GetAddHooks(tmpDir)

		if len(hooks) != 0 {
			t.Errorf("Expected empty hooks, got %v", hooks)
		}
	})

	t.Run("returns empty when no add hooks defined", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .grove.toml without hooks section
		tomlContent := `[preserve]
patterns = [".env"]
`
		tomlPath := filepath.Join(tmpDir, ".grove.toml")
		if err := os.WriteFile(tomlPath, []byte(tomlContent), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		hooks := GetAddHooks(tmpDir)

		if len(hooks) != 0 {
			t.Errorf("Expected empty hooks, got %v", hooks)
		}
	})
}
