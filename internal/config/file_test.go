package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	t.Run("parses preserve patterns from TOML", func(t *testing.T) {
		tmpDir := t.TempDir()
		tomlContent := `[preserve]
patterns = [".env", ".env.local", "*.secret"]
`
		tomlPath := filepath.Join(tmpDir, ".grove.toml")
		if err := os.WriteFile(tomlPath, []byte(tomlContent), 0o644); err != nil { //nolint:gosec
			t.Fatal(err)
		}

		cfg, err := LoadFromFile(tmpDir)
		if err != nil {
			t.Fatalf("LoadFromFile failed: %v", err)
		}

		expected := []string{".env", ".env.local", "*.secret"}
		if len(cfg.Preserve.Patterns) != len(expected) {
			t.Errorf("Expected %d patterns, got %d", len(expected), len(cfg.Preserve.Patterns))
		}
		for i, exp := range expected {
			if i >= len(cfg.Preserve.Patterns) || cfg.Preserve.Patterns[i] != exp {
				t.Errorf("Expected pattern %d to be %q, got %q", i, exp, cfg.Preserve.Patterns[i])
			}
		}
	})

	t.Run("parses preserve exclude patterns from TOML", func(t *testing.T) {
		tmpDir := t.TempDir()
		tomlContent := `[preserve]
exclude = ["node_modules", "vendor", ".cache"]
`
		tomlPath := filepath.Join(tmpDir, ".grove.toml")
		if err := os.WriteFile(tomlPath, []byte(tomlContent), 0o644); err != nil { //nolint:gosec
			t.Fatal(err)
		}

		cfg, err := LoadFromFile(tmpDir)
		if err != nil {
			t.Fatalf("LoadFromFile failed: %v", err)
		}

		expected := []string{"node_modules", "vendor", ".cache"}
		if len(cfg.Preserve.Exclude) != len(expected) {
			t.Errorf("Expected %d exclude patterns, got %d", len(expected), len(cfg.Preserve.Exclude))
		}
		for i, exp := range expected {
			if i >= len(cfg.Preserve.Exclude) || cfg.Preserve.Exclude[i] != exp {
				t.Errorf("Expected exclude pattern %d to be %q, got %q", i, exp, cfg.Preserve.Exclude[i])
			}
		}
	})

	t.Run("parses hooks from TOML", func(t *testing.T) {
		tmpDir := t.TempDir()
		tomlContent := `[hooks]
add = ["pnpm i", "pnpm build"]
`
		tomlPath := filepath.Join(tmpDir, ".grove.toml")
		if err := os.WriteFile(tomlPath, []byte(tomlContent), 0o644); err != nil { //nolint:gosec
			t.Fatal(err)
		}

		cfg, err := LoadFromFile(tmpDir)
		if err != nil {
			t.Fatalf("LoadFromFile failed: %v", err)
		}

		expected := []string{"pnpm i", "pnpm build"}
		if len(cfg.Hooks.Add) != len(expected) {
			t.Errorf("Expected %d hooks, got %d", len(expected), len(cfg.Hooks.Add))
		}
		for i, exp := range expected {
			if i >= len(cfg.Hooks.Add) || cfg.Hooks.Add[i] != exp {
				t.Errorf("Expected hook %d to be %q, got %q", i, exp, cfg.Hooks.Add[i])
			}
		}
	})

	t.Run("parses plain and debug from TOML", func(t *testing.T) {
		tmpDir := t.TempDir()
		tomlContent := `plain = true
debug = false
`
		tomlPath := filepath.Join(tmpDir, ".grove.toml")
		if err := os.WriteFile(tomlPath, []byte(tomlContent), 0o644); err != nil { //nolint:gosec
			t.Fatal(err)
		}

		cfg, err := LoadFromFile(tmpDir)
		if err != nil {
			t.Fatalf("LoadFromFile failed: %v", err)
		}

		if cfg.Plain == nil || *cfg.Plain != true {
			t.Error("Expected plain to be true")
		}
		if cfg.Debug == nil || *cfg.Debug != false {
			t.Error("Expected debug to be false")
		}
	})

	t.Run("returns empty config when file missing", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg, err := LoadFromFile(tmpDir)
		if err != nil {
			t.Fatalf("LoadFromFile should not error on missing file: %v", err)
		}

		if len(cfg.Preserve.Patterns) != 0 {
			t.Errorf("Expected empty patterns, got %v", cfg.Preserve.Patterns)
		}
		if len(cfg.Hooks.Add) != 0 {
			t.Errorf("Expected empty hooks, got %v", cfg.Hooks.Add)
		}
	})

	t.Run("returns error on invalid TOML", func(t *testing.T) {
		tmpDir := t.TempDir()
		tomlContent := `[preserve
patterns = invalid
`
		tomlPath := filepath.Join(tmpDir, ".grove.toml")
		if err := os.WriteFile(tomlPath, []byte(tomlContent), 0o644); err != nil { //nolint:gosec
			t.Fatal(err)
		}

		_, err := LoadFromFile(tmpDir)
		if err == nil {
			t.Error("Expected error on invalid TOML")
		}
	})
}

func TestWriteToFile(t *testing.T) {
	t.Run("writes valid TOML to file", func(t *testing.T) {
		tmpDir := t.TempDir()

		plainTrue := true
		debugFalse := false
		cfg := FileConfig{
			Plain: &plainTrue,
			Debug: &debugFalse,
		}
		cfg.Preserve.Patterns = []string{".env", ".secret"}
		cfg.Hooks.Add = []string{"npm install"}

		if err := WriteToFile(tmpDir, &cfg); err != nil {
			t.Fatalf("WriteToFile failed: %v", err)
		}

		// Read back and verify
		loaded, err := LoadFromFile(tmpDir)
		if err != nil {
			t.Fatalf("LoadFromFile failed: %v", err)
		}

		if loaded.Plain == nil || cfg.Plain == nil || *loaded.Plain != *cfg.Plain {
			t.Errorf("Expected plain=%v, got %v", cfg.Plain, loaded.Plain)
		}
		if len(loaded.Preserve.Patterns) != 2 {
			t.Errorf("Expected 2 patterns, got %d", len(loaded.Preserve.Patterns))
		}
		if len(loaded.Hooks.Add) != 1 {
			t.Errorf("Expected 1 hook, got %d", len(loaded.Hooks.Add))
		}
	})

	t.Run("uses atomic write (temp file + rename)", func(t *testing.T) {
		tmpDir := t.TempDir()
		tomlPath := filepath.Join(tmpDir, ".grove.toml")

		if err := os.WriteFile(tomlPath, []byte("plain = false\n"), 0o644); err != nil { //nolint:gosec
			t.Fatal(err)
		}

		plainTrue := true
		cfg := FileConfig{Plain: &plainTrue}
		if err := WriteToFile(tmpDir, &cfg); err != nil {
			t.Fatalf("WriteToFile failed: %v", err)
		}

		// Verify file was updated
		loaded, err := LoadFromFile(tmpDir)
		if err != nil {
			t.Fatalf("LoadFromFile failed: %v", err)
		}
		if loaded.Plain == nil || *loaded.Plain != true {
			t.Error("Expected atomic write to update file")
		}

		// Verify no temp files left behind
		entries, _ := os.ReadDir(tmpDir)
		for _, e := range entries {
			if e.Name() != ".grove.toml" {
				t.Errorf("Temp file left behind: %s", e.Name())
			}
		}
	})
}

func TestFileConfigExists(t *testing.T) {
	t.Run("returns true when file exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		tomlPath := filepath.Join(tmpDir, ".grove.toml")
		if err := os.WriteFile(tomlPath, []byte(""), 0o644); err != nil { //nolint:gosec
			t.Fatal(err)
		}

		if !FileConfigExists(tmpDir) {
			t.Error("Expected FileConfigExists to return true")
		}
	})

	t.Run("returns false when file missing", func(t *testing.T) {
		tmpDir := t.TempDir()

		if FileConfigExists(tmpDir) {
			t.Error("Expected FileConfigExists to return false")
		}
	})
}

func setupGitRepoForFileTests(t *testing.T) (tmpDir string, cleanup func()) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	tmpDir = t.TempDir()
	oldWd, _ := os.Getwd()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.name", "Test"},
		{"git", "config", "user.email", "test@example.com"},
	}
	for _, args := range cmds {
		if err := exec.Command(args[0], args[1:]...).Run(); err != nil { //nolint:gosec
			_ = os.Chdir(oldWd)
			t.Fatal(err)
		}
	}

	return tmpDir, func() { _ = os.Chdir(oldWd) }
}

func TestMergedConfig(t *testing.T) {
	t.Run("TOML takes precedence for preserve patterns", func(t *testing.T) {
		tmpDir, cleanup := setupGitRepoForFileTests(t)
		defer cleanup()

		_ = exec.Command("git", "config", "grove.preserve", ".gitconfig-pattern").Run() //nolint:gosec

		tomlContent := `[preserve]
patterns = [".toml-pattern"]
`
		_ = os.WriteFile(filepath.Join(tmpDir, ".grove.toml"), []byte(tomlContent), 0o644) //nolint:gosec

		patterns := GetMergedPreservePatterns(tmpDir)

		if len(patterns) != 1 || patterns[0] != ".toml-pattern" {
			t.Errorf("Expected TOML patterns to take precedence, got %v", patterns)
		}
	})

	t.Run("git config used when no TOML preserve patterns", func(t *testing.T) {
		tmpDir, cleanup := setupGitRepoForFileTests(t)
		defer cleanup()

		_ = exec.Command("git", "config", "grove.preserve", ".gitconfig-pattern").Run() //nolint:gosec

		patterns := GetMergedPreservePatterns(tmpDir)

		if len(patterns) != 1 || patterns[0] != ".gitconfig-pattern" {
			t.Errorf("Expected git config patterns, got %v", patterns)
		}
	})

	t.Run("defaults used when neither TOML nor git config", func(t *testing.T) {
		tmpDir, cleanup := setupGitRepoForFileTests(t)
		defer cleanup()

		// No TOML, no git config

		patterns := GetMergedPreservePatterns(tmpDir)

		if len(patterns) != len(DefaultConfig.PreservePatterns) {
			t.Errorf("Expected default patterns, got %v", patterns)
		}
	})

	t.Run("git config takes precedence for plain setting", func(t *testing.T) {
		tmpDir, cleanup := setupGitRepoForFileTests(t)
		defer cleanup()

		tomlContent := `plain = false`
		_ = os.WriteFile(filepath.Join(tmpDir, ".grove.toml"), []byte(tomlContent), 0o644) //nolint:gosec

		_ = exec.Command("git", "config", "grove.plain", "true").Run() //nolint:gosec

		plain := GetMergedPlain(tmpDir)

		// Git config should win for personal settings
		if plain != true {
			t.Error("Expected git config to take precedence for plain")
		}
	})

	t.Run("TOML used for plain when no git config", func(t *testing.T) {
		tmpDir, cleanup := setupGitRepoForFileTests(t)
		defer cleanup()

		tomlContent := `plain = true`
		_ = os.WriteFile(filepath.Join(tmpDir, ".grove.toml"), []byte(tomlContent), 0o644) //nolint:gosec

		plain := GetMergedPlain(tmpDir)

		if plain != true {
			t.Error("Expected TOML value when no git config")
		}
	})

	t.Run("TOML plain=false is respected (not treated as unset)", func(t *testing.T) {
		tmpDir, cleanup := setupGitRepoForFileTests(t)
		defer cleanup()

		// This tests the bug fix: plain = false should be explicitly set, not treated as "not set"
		tomlContent := `plain = false`
		_ = os.WriteFile(filepath.Join(tmpDir, ".grove.toml"), []byte(tomlContent), 0o644) //nolint:gosec

		plain := GetMergedPlain(tmpDir)

		// plain = false in TOML should be returned (not the default)
		if plain != false {
			t.Error("Expected TOML plain=false to be respected")
		}
	})

	t.Run("git config takes precedence for debug setting", func(t *testing.T) {
		tmpDir, cleanup := setupGitRepoForFileTests(t)
		defer cleanup()

		tomlContent := `debug = false`
		_ = os.WriteFile(filepath.Join(tmpDir, ".grove.toml"), []byte(tomlContent), 0o644) //nolint:gosec

		_ = exec.Command("git", "config", "grove.debug", "true").Run() //nolint:gosec

		debug := GetMergedDebug(tmpDir)

		if debug != true {
			t.Error("Expected git config to take precedence for debug")
		}
	})

	t.Run("GetMergedPreserveExcludePatterns uses TOML first", func(t *testing.T) {
		tmpDir, cleanup := setupGitRepoForFileTests(t)
		defer cleanup()

		tomlContent := `[preserve]
exclude = ["vendor", ".cache"]
`
		_ = os.WriteFile(filepath.Join(tmpDir, ".grove.toml"), []byte(tomlContent), 0o644) //nolint:gosec
		_ = exec.Command("git", "config", "grove.preserveExclude", "node_modules").Run()   //nolint:gosec

		patterns := GetMergedPreserveExcludePatterns(tmpDir)

		if len(patterns) != 2 || patterns[0] != "vendor" || patterns[1] != ".cache" {
			t.Errorf("Expected TOML patterns, got %v", patterns)
		}
	})

	t.Run("GetMergedPreserveExcludePatterns falls back to defaults", func(t *testing.T) {
		tmpDir, cleanup := setupGitRepoForFileTests(t)
		defer cleanup()

		patterns := GetMergedPreserveExcludePatterns(tmpDir)

		if len(patterns) != len(DefaultConfig.PreserveExcludePatterns) {
			t.Errorf("Expected default patterns, got %v", patterns)
		}
	})
}
