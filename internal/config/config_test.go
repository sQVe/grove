package config

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestGlobalConfig(t *testing.T) {
	t.Run("initial state is false for both modes", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}

		if IsPlain() {
			t.Error("Expected plain mode to be false initially")
		}
		if IsDebug() {
			t.Error("Expected debug mode to be false initially")
		}
	})

	t.Run("plain mode can be enabled", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}

		Global.Plain = true
		if !IsPlain() {
			t.Error("Expected plain mode to be true after setting Global.Plain = true")
		}
	})

	t.Run("debug mode can be enabled", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}

		Global.Debug = true
		if !IsDebug() {
			t.Error("Expected debug mode to be true after setting Global.Debug = true")
		}
	})
}

func TestLoadFromEnv(t *testing.T) {
	t.Run("both false with no env vars", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}
		_ = os.Unsetenv("GROVE_PLAIN")
		_ = os.Unsetenv("GROVE_DEBUG")

		LoadFromEnv()
		if IsPlain() || IsDebug() {
			t.Error("Expected both modes to be false with no env vars")
		}
	})

	t.Run("plain true with GROVE_PLAIN=1", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}
		_ = os.Unsetenv("GROVE_DEBUG")
		_ = os.Setenv("GROVE_PLAIN", "1")
		defer func() { _ = os.Unsetenv("GROVE_PLAIN") }()

		LoadFromEnv()
		if !IsPlain() {
			t.Error("Expected plain mode to be true with GROVE_PLAIN=1")
		}
	})

	t.Run("debug true with GROVE_DEBUG=1", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}
		_ = os.Unsetenv("GROVE_PLAIN")
		_ = os.Setenv("GROVE_DEBUG", "1")
		defer func() { _ = os.Unsetenv("GROVE_DEBUG") }()

		LoadFromEnv()
		if !IsDebug() {
			t.Error("Expected debug mode to be true with GROVE_DEBUG=1")
		}
	})

	t.Run("plain true with GROVE_PLAIN=true", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}
		_ = os.Unsetenv("GROVE_DEBUG")
		_ = os.Setenv("GROVE_PLAIN", "true")
		defer func() { _ = os.Unsetenv("GROVE_PLAIN") }()

		LoadFromEnv()
		if !IsPlain() {
			t.Error("Expected plain mode to be true with GROVE_PLAIN=true")
		}
	})

	t.Run("debug true with GROVE_DEBUG=true", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}
		_ = os.Unsetenv("GROVE_PLAIN")
		_ = os.Setenv("GROVE_DEBUG", "true")
		defer func() { _ = os.Unsetenv("GROVE_DEBUG") }()

		LoadFromEnv()
		if !IsDebug() {
			t.Error("Expected debug mode to be true with GROVE_DEBUG=true")
		}
	})

	t.Run("both false with invalid env values", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}
		_ = os.Setenv("GROVE_PLAIN", "yes")
		_ = os.Setenv("GROVE_DEBUG", "on")
		defer func() {
			_ = os.Unsetenv("GROVE_PLAIN")
			_ = os.Unsetenv("GROVE_DEBUG")
		}()

		LoadFromEnv()
		if IsPlain() || IsDebug() {
			t.Error("Expected both modes to be false with invalid env values")
		}
	})
}

func TestLoadFromGitConfig(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH, skipping test")
	}

	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "config", "user.name", "Test").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "config", "user.email", "test@example.com").Run(); err != nil {
		t.Fatal(err)
	}

	t.Run("loads grove.plain from git config", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}
		_ = os.Unsetenv("GROVE_PLAIN")
		_ = os.Unsetenv("GROVE_DEBUG")

		if err := exec.Command("git", "config", "grove.plain", "true").Run(); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = exec.Command("git", "config", "--unset", "grove.plain").Run() }()

		LoadFromGitConfig()
		if !IsPlain() {
			t.Error("Expected plain mode to be true from git config")
		}
	})

	t.Run("loads grove.debug from git config", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}
		_ = os.Unsetenv("GROVE_PLAIN")
		_ = os.Unsetenv("GROVE_DEBUG")

		if err := exec.Command("git", "config", "grove.debug", "true").Run(); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = exec.Command("git", "config", "--unset", "grove.debug").Run() }()

		LoadFromGitConfig()
		if !IsDebug() {
			t.Error("Expected debug mode to be true from git config")
		}
	})

	t.Run("handles git config errors gracefully", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}

		LoadFromGitConfig()
		if IsPlain() || IsDebug() {
			t.Error("Expected modes to remain false on git config error")
		}
	})
}

func TestPrecedence(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH, skipping test")
	}

	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "config", "user.name", "Test").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "config", "user.email", "test@example.com").Run(); err != nil {
		t.Fatal(err)
	}

	t.Run("ENV overrides git config", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}

		if err := exec.Command("git", "config", "grove.plain", "false").Run(); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = exec.Command("git", "config", "--unset", "grove.plain").Run() }()

		_ = os.Setenv("GROVE_PLAIN", "1")
		defer func() { _ = os.Unsetenv("GROVE_PLAIN") }()

		LoadFromGitConfig()
		LoadFromEnv()

		if !IsPlain() {
			t.Error("Expected ENV to override git config (ENV=true, git=false)")
		}
	})

	t.Run("git config used when ENV not set", func(t *testing.T) {
		Global = struct {
			Plain bool
			Debug bool
		}{}
		_ = os.Unsetenv("GROVE_PLAIN")
		_ = os.Unsetenv("GROVE_DEBUG")

		if err := exec.Command("git", "config", "grove.debug", "true").Run(); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = exec.Command("git", "config", "--unset", "grove.debug").Run() }()

		LoadFromGitConfig()

		if !IsDebug() {
			t.Error("Expected git config value to be used when ENV not set")
		}
	})
}

func TestDefaultPatterns(t *testing.T) {
	t.Run("returns default patterns when none configured", func(t *testing.T) {
		expected := []string{
			".env",
			".env.local",
			".env.development.local",
			"*.local.json",
			"*.local.yaml",
			"*.local.yml",
			"*.local.toml",
		}

		got := GetDefaultPreserveIgnoredPatterns()
		if len(got) != len(expected) {
			t.Errorf("Expected %d default patterns, got %d", len(expected), len(got))
		}

		for i, want := range expected {
			if i >= len(got) || got[i] != want {
				t.Errorf("Default pattern[%d] = %q, want %q", i, got[i], want)
			}
		}
	})

	t.Run("patterns are in stable order", func(t *testing.T) {
		patterns1 := GetDefaultPreserveIgnoredPatterns()
		patterns2 := GetDefaultPreserveIgnoredPatterns()

		if len(patterns1) != len(patterns2) {
			t.Error("Default patterns should return consistent length")
		}

		for i, p1 := range patterns1 {
			if i >= len(patterns2) || patterns2[i] != p1 {
				t.Error("Default patterns should return consistent order")
			}
		}
	})
}

func TestMultiValueParsing(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH, skipping test")
	}

	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "config", "user.name", "Test").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "config", "user.email", "test@example.com").Run(); err != nil {
		t.Fatal(err)
	}

	t.Run("parses multiple preserve ignored patterns", func(t *testing.T) {
		patterns := []string{".env", "*.secret", ".local"}
		for _, pattern := range patterns {
			if err := exec.Command("git", "config", "--add", "grove.convert.preserve", pattern).Run(); err != nil { //nolint:gosec
				t.Fatal(err)
			}
		}
		defer func() {
			_ = exec.Command("git", "config", "--unset-all", "grove.convert.preserve").Run()
		}()

		got := GetPreserveIgnoredPatterns()

		if len(got) < len(patterns) {
			t.Errorf("Expected at least %d patterns (configured), got %d", len(patterns), len(got))
		}

		gotStr := strings.Join(got, ",")
		for _, pattern := range patterns {
			if !strings.Contains(gotStr, pattern) {
				t.Errorf("Expected pattern %q to be present in %v", pattern, got)
			}
		}
	})

	t.Run("preserves order of patterns", func(t *testing.T) {
		orderedPatterns := []string{"first.pattern", "second.pattern", "third.pattern"}
		for _, pattern := range orderedPatterns {
			if err := exec.Command("git", "config", "--add", "grove.convert.preserve", pattern).Run(); err != nil { //nolint:gosec
				t.Fatal(err)
			}
		}
		defer func() {
			_ = exec.Command("git", "config", "--unset-all", "grove.convert.preserve").Run()
		}()

		got := GetPreserveIgnoredPatterns()

		var foundPatterns []string
		for _, pattern := range got {
			for _, ordered := range orderedPatterns {
				if pattern == ordered {
					foundPatterns = append(foundPatterns, pattern)
					break
				}
			}
		}

		if len(foundPatterns) != len(orderedPatterns) {
			t.Errorf("Expected to find %d ordered patterns, found %d", len(orderedPatterns), len(foundPatterns))
		}

		for i, expected := range orderedPatterns {
			if i >= len(foundPatterns) || foundPatterns[i] != expected {
				t.Errorf("Pattern order[%d] = %q, want %q", i, foundPatterns[i], expected)
			}
		}
	})
}
