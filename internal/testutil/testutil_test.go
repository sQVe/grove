package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTempDir(t *testing.T) {
	t.Run("returns resolved path", func(t *testing.T) {
		dir := TempDir(t)

		// Should be an absolute path
		if !filepath.IsAbs(dir) {
			t.Errorf("expected absolute path, got %q", dir)
		}

		// Directory should exist
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("directory should exist: %v", err)
		}
		if !info.IsDir() {
			t.Error("expected directory, got file")
		}

		// Should be resolved (no symlinks)
		resolved, err := filepath.EvalSymlinks(dir)
		if err != nil {
			t.Fatalf("failed to resolve: %v", err)
		}
		if resolved != dir {
			t.Errorf("expected already resolved path, got %q resolved to %q", dir, resolved)
		}
	})
}

func TestInTempDir(t *testing.T) {
	t.Run("executes fn in temp directory", func(t *testing.T) {
		origDir, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}

		var capturedDir string
		InTempDir(t, func(dir string) {
			capturedDir = dir
			cwd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if cwd != dir {
				t.Errorf("expected cwd %q, got %q", dir, cwd)
			}
		})

		// Verify cwd was restored
		afterDir, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if afterDir != origDir {
			t.Errorf("expected cwd to be restored to %q, got %q", origDir, afterDir)
		}

		// Captured dir should have been a temp directory
		if capturedDir == "" {
			t.Error("expected fn to be called with temp dir")
		}
	})

	t.Run("restores cwd even when fn panics", func(t *testing.T) {
		origDir, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			_ = recover()
			afterDir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if afterDir != origDir {
				t.Errorf("expected cwd to be restored to %q after panic, got %q", origDir, afterDir)
			}
		}()

		InTempDir(t, func(_ string) {
			panic("test panic")
		})
	})
}

func TestSaveCwd(t *testing.T) {
	t.Run("returns cleanup function that restores cwd", func(t *testing.T) {
		origDir, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}

		cleanup := SaveCwd(t)

		// Change to a temp directory
		tempDir := t.TempDir()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatal(err)
		}

		// Call cleanup
		cleanup()

		// Verify cwd was restored
		afterDir, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if afterDir != origDir {
			t.Errorf("expected cwd to be restored to %q, got %q", origDir, afterDir)
		}
	})
}

func TestChdir(t *testing.T) {
	t.Run("changes directory", func(t *testing.T) {
		origDir, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chdir(origDir) }()

		tempDir := t.TempDir()
		Chdir(t, tempDir)

		cwd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}

		// Resolve both paths to handle symlinks
		resolvedTempDir, _ := filepath.EvalSymlinks(tempDir)
		resolvedCwd, _ := filepath.EvalSymlinks(cwd)

		if resolvedCwd != resolvedTempDir {
			t.Errorf("expected cwd %q, got %q", resolvedTempDir, resolvedCwd)
		}
	})
}

func TestWriteFile(t *testing.T) {
	t.Run("writes content with default permissions", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "test.txt")

		WriteFile(t, path, "hello world")

		content, err := os.ReadFile(path) // nolint:gosec // Test-controlled path
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(content) != "hello world" {
			t.Errorf("expected %q, got %q", "hello world", string(content))
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "nested", "deep", "test.txt")

		WriteFile(t, path, "nested content")

		content, err := os.ReadFile(path) // nolint:gosec // Test-controlled path
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(content) != "nested content" {
			t.Errorf("expected %q, got %q", "nested content", string(content))
		}
	})
}

func TestWriteFileMode(t *testing.T) {
	t.Run("writes with specified permissions", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "executable.sh")

		WriteFileMode(t, path, "#!/bin/bash", 0o755)

		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("failed to stat file: %v", err)
		}

		// Check executable bit is set (permissions vary by umask)
		if info.Mode().Perm()&0o100 == 0 {
			t.Error("expected executable permission to be set")
		}
	})
}

func TestMustExec(t *testing.T) {
	t.Run("returns output on success", func(t *testing.T) {
		tempDir := t.TempDir()
		output := MustExec(t, tempDir, "echo", "hello")

		if output != "hello\n" {
			t.Errorf("expected %q, got %q", "hello\n", output)
		}
	})

	t.Run("uses specified directory", func(t *testing.T) {
		tempDir := t.TempDir()
		WriteFile(t, filepath.Join(tempDir, "marker.txt"), "")

		output := MustExec(t, tempDir, "ls")

		if output != "marker.txt\n" {
			t.Errorf("expected 'marker.txt\\n', got %q", output)
		}
	})
}
