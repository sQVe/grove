package config

import (
	"os"
	"os/exec"
	"testing"
)

// setupGitRepo creates a temporary git repo and changes to it
// Returns a cleanup function that restores the original working directory
func setupGitRepo(t *testing.T) func() {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH, skipping test")
	}

	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

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

	return func() { _ = os.Chdir(oldWd) }
}

func TestIsTruthy(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"1", true},
		{"True", true},
		{"TRUE", true},
		{"TrUe", true},
		{" true ", true},
		{" 1 ", true},
		{"  true  ", true},
		{"yes", true},
		{"YES", true},
		{"Yes", true},
		{" yes ", true},
		{"on", true},
		{"ON", true},
		{"On", true},
		{" on ", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"off", false},
		{"", false},
		{" ", false},
		{"   ", false},
		{"anything", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isTruthy(tt.input)
			if result != tt.expected {
				t.Errorf("isTruthy(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Run("both false with no env vars", func(t *testing.T) {
		Global = struct {
			Plain            bool
			Debug            bool
			PreservePatterns []string
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
			Plain            bool
			Debug            bool
			PreservePatterns []string
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
			Plain            bool
			Debug            bool
			PreservePatterns []string
		}{}
		_ = os.Unsetenv("GROVE_PLAIN")
		_ = os.Setenv("GROVE_DEBUG", "1")
		defer func() { _ = os.Unsetenv("GROVE_DEBUG") }()

		LoadFromEnv()
		if !IsDebug() {
			t.Error("Expected debug mode to be true with GROVE_DEBUG=1")
		}
	})

	t.Run("both true with yes/on env values", func(t *testing.T) {
		Global = struct {
			Plain            bool
			Debug            bool
			PreservePatterns []string
		}{}
		_ = os.Setenv("GROVE_PLAIN", "yes")
		_ = os.Setenv("GROVE_DEBUG", "on")
		defer func() {
			_ = os.Unsetenv("GROVE_PLAIN")
			_ = os.Unsetenv("GROVE_DEBUG")
		}()

		LoadFromEnv()
		if !IsPlain() || !IsDebug() {
			t.Error("Expected both modes to be true with yes/on env values")
		}
	})

	t.Run("both false with invalid env values", func(t *testing.T) {
		Global = struct {
			Plain            bool
			Debug            bool
			PreservePatterns []string
		}{}
		_ = os.Setenv("GROVE_PLAIN", "invalid")
		_ = os.Setenv("GROVE_DEBUG", "nope")
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
	cleanup := setupGitRepo(t)
	defer cleanup()

	t.Run("loads grove.plain from git config", func(t *testing.T) {
		Global = struct {
			Plain            bool
			Debug            bool
			PreservePatterns []string
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
			Plain            bool
			Debug            bool
			PreservePatterns []string
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
			Plain            bool
			Debug            bool
			PreservePatterns []string
		}{}

		_ = exec.Command("git", "config", "--unset", "grove.plain").Run()
		_ = exec.Command("git", "config", "--unset", "grove.debug").Run()

		LoadFromGitConfig()
		if IsPlain() || IsDebug() {
			t.Error("Expected modes to remain false on git config error")
		}
	})
}

func TestPrecedence(t *testing.T) {
	cleanup := setupGitRepo(t)
	defer cleanup()

	t.Run("ENV overrides git config", func(t *testing.T) {
		Global = struct {
			Plain            bool
			Debug            bool
			PreservePatterns []string
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
			Plain            bool
			Debug            bool
			PreservePatterns []string
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

func TestMainLoadingSequence(t *testing.T) {
	cleanup := setupGitRepo(t)
	defer cleanup()

	t.Run("main loading sequence works correctly", func(t *testing.T) {
		Global = struct {
			Plain            bool
			Debug            bool
			PreservePatterns []string
		}{}

		if err := exec.Command("git", "config", "grove.plain", "true").Run(); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = exec.Command("git", "config", "--unset", "grove.plain").Run() }()

		_ = os.Unsetenv("GROVE_PLAIN")
		_ = os.Unsetenv("GROVE_DEBUG")

		LoadFromGitConfig()
		LoadFromEnv()

		if !IsPlain() {
			t.Error("Expected git config to be loaded when env not set")
		}
	})

	t.Run("env overrides git config in loading sequence", func(t *testing.T) {
		Global = struct {
			Plain            bool
			Debug            bool
			PreservePatterns []string
		}{}

		if err := exec.Command("git", "config", "grove.plain", "true").Run(); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = exec.Command("git", "config", "--unset", "grove.plain").Run() }()

		_ = os.Setenv("GROVE_PLAIN", "false")
		defer func() { _ = os.Unsetenv("GROVE_PLAIN") }()

		LoadFromGitConfig()
		LoadFromEnv()

		if IsPlain() {
			t.Error("Expected env variable to override git config (env=false should win over git=true)")
		}
	})
}

func TestDefaults(t *testing.T) {
	t.Run("DefaultConfig has expected values", func(t *testing.T) {
		if DefaultConfig.Plain != false {
			t.Error("Expected DefaultConfig.Plain to be false")
		}
		if DefaultConfig.Debug != false {
			t.Error("Expected DefaultConfig.Debug to be false")
		}

		expectedPatterns := []string{
			".env",
			".env.local",
			".env.development.local",
			".env.test.local",
			".env.production.local",
			"*.local.json",
			"*.local.yaml",
			"*.local.yml",
			"*.local.toml",
		}

		if len(DefaultConfig.PreservePatterns) != len(expectedPatterns) {
			t.Errorf("Expected %d preserve patterns, got %d", len(expectedPatterns), len(DefaultConfig.PreservePatterns))
		}

		for i, expected := range expectedPatterns {
			if i >= len(DefaultConfig.PreservePatterns) || DefaultConfig.PreservePatterns[i] != expected {
				t.Errorf("Expected preserve pattern %d to be %q, got %q", i, expected, DefaultConfig.PreservePatterns[i])
			}
		}
	})
}

func TestGetPreservePatterns(t *testing.T) {
	t.Run("returns defaults when Global is empty", func(t *testing.T) {
		Global = struct {
			Plain            bool
			Debug            bool
			PreservePatterns []string
		}{}

		patterns := GetPreservePatterns()
		if len(patterns) != len(DefaultConfig.PreservePatterns) {
			t.Errorf("Expected %d patterns, got %d", len(DefaultConfig.PreservePatterns), len(patterns))
		}

		for i, expected := range DefaultConfig.PreservePatterns {
			if i >= len(patterns) || patterns[i] != expected {
				t.Errorf("Expected pattern %d to be %q, got %q", i, expected, patterns[i])
			}
		}
	})

	t.Run("returns Global patterns when set", func(t *testing.T) {
		customPatterns := []string{".custom", "*.test"}
		Global = struct {
			Plain            bool
			Debug            bool
			PreservePatterns []string
		}{
			PreservePatterns: customPatterns,
		}

		patterns := GetPreservePatterns()
		if len(patterns) != len(customPatterns) {
			t.Errorf("Expected %d patterns, got %d", len(customPatterns), len(patterns))
		}

		for i, expected := range customPatterns {
			if i >= len(patterns) || patterns[i] != expected {
				t.Errorf("Expected pattern %d to be %q, got %q", i, expected, patterns[i])
			}
		}
	})

	t.Run("returns consistent patterns", func(t *testing.T) {
		Global = struct {
			Plain            bool
			Debug            bool
			PreservePatterns []string
		}{}

		patterns1 := GetPreservePatterns()
		patterns2 := GetPreservePatterns()

		if len(patterns1) != len(patterns2) {
			t.Errorf("Expected consistent length, got %d and %d", len(patterns1), len(patterns2))
		}

		for i, expected := range patterns1 {
			if i >= len(patterns2) || patterns2[i] != expected {
				t.Errorf("Expected pattern %d to be %q in both calls", i, expected)
			}
		}
	})
}

func TestLoadFromGitConfigWithDefaults(t *testing.T) {
	cleanup := setupGitRepo(t)
	defer cleanup()

	t.Run("uses defaults when no git config set", func(t *testing.T) {
		Global = struct {
			Plain            bool
			Debug            bool
			PreservePatterns []string
		}{}

		_ = exec.Command("git", "config", "--unset", "grove.plain").Run()
		_ = exec.Command("git", "config", "--unset", "grove.debug").Run()
		_ = exec.Command("git", "config", "--unset-all", "grove.convert.preserve").Run()

		LoadFromGitConfig()

		if Global.Plain != DefaultConfig.Plain {
			t.Errorf("Expected Plain to be %v (default), got %v", DefaultConfig.Plain, Global.Plain)
		}
		if Global.Debug != DefaultConfig.Debug {
			t.Errorf("Expected Debug to be %v (default), got %v", DefaultConfig.Debug, Global.Debug)
		}
		if len(Global.PreservePatterns) != len(DefaultConfig.PreservePatterns) {
			t.Errorf("Expected %d preserve patterns (default), got %d", len(DefaultConfig.PreservePatterns), len(Global.PreservePatterns))
		}
	})

	t.Run("git config overrides defaults", func(t *testing.T) {
		Global = struct {
			Plain            bool
			Debug            bool
			PreservePatterns []string
		}{}

		if err := exec.Command("git", "config", "grove.plain", "true").Run(); err != nil {
			t.Fatal(err)
		}
		if err := exec.Command("git", "config", "grove.debug", "false").Run(); err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = exec.Command("git", "config", "--unset", "grove.plain").Run()
			_ = exec.Command("git", "config", "--unset", "grove.debug").Run()
		}()

		LoadFromGitConfig()

		if Global.Plain != true {
			t.Error("Expected git config to override Plain default")
		}
		if Global.Debug != false {
			t.Error("Expected git config to override Debug default")
		}
	})

	t.Run("preserve patterns replace defaults", func(t *testing.T) {
		Global = struct {
			Plain            bool
			Debug            bool
			PreservePatterns []string
		}{}

		if err := exec.Command("git", "config", "grove.convert.preserve", ".custom").Run(); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = exec.Command("git", "config", "--unset-all", "grove.convert.preserve").Run() }()

		LoadFromGitConfig()

		if len(Global.PreservePatterns) != 1 || Global.PreservePatterns[0] != ".custom" {
			t.Errorf("Expected preserve patterns to be replaced with ['.custom'], got %v", Global.PreservePatterns)
		}
	})

	t.Run("multiple preserve patterns from git config", func(t *testing.T) {
		Global = struct {
			Plain            bool
			Debug            bool
			PreservePatterns []string
		}{}

		if err := exec.Command("git", "config", "--add", "grove.convert.preserve", ".env").Run(); err != nil {
			t.Fatal(err)
		}
		if err := exec.Command("git", "config", "--add", "grove.convert.preserve", "*.local").Run(); err != nil {
			t.Fatal(err)
		}
		if err := exec.Command("git", "config", "--add", "grove.convert.preserve", ".secret").Run(); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = exec.Command("git", "config", "--unset-all", "grove.convert.preserve").Run() }()

		LoadFromGitConfig()

		expected := []string{".env", "*.local", ".secret"}
		if len(Global.PreservePatterns) != len(expected) {
			t.Errorf("Expected %d patterns, got %d: %v", len(expected), len(Global.PreservePatterns), Global.PreservePatterns)
		}
		for i, exp := range expected {
			if i >= len(Global.PreservePatterns) || Global.PreservePatterns[i] != exp {
				t.Errorf("Expected pattern %d to be %q, got %q", i, exp, Global.PreservePatterns[i])
			}
		}
	})
}
