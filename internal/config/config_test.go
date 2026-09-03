package config

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/sqve/grove/internal/testutil"
)

// setupGitRepo creates a temporary git repo and changes to it
// Returns a cleanup function that restores the original working directory
func setupGitRepo(t *testing.T) func() {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH, skipping test")
	}

	tmpDir := testutil.TempDir(t)
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

// resetGlobal resets the global config to zero values
func resetGlobal() {
	Global.Plain = false
	Global.Debug = false
	Global.NerdFonts = true // Default is true
	Global.StaleThreshold = ""
	Global.AutoLockPatterns = nil
	Global.Timeout = 0
}

func TestLoadFromGitConfig(t *testing.T) {
	cleanup := setupGitRepo(t)
	defer cleanup()

	t.Run("loads grove.plain from git config", func(t *testing.T) {
		resetGlobal()

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
		resetGlobal()

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
		resetGlobal()

		_ = exec.Command("git", "config", "--unset", "grove.plain").Run()
		_ = exec.Command("git", "config", "--unset", "grove.debug").Run()

		LoadFromGitConfig()
		if IsPlain() || IsDebug() {
			t.Error("Expected modes to remain false on git config error")
		}
	})

	t.Run("loads grove.nerdFonts from git config", func(t *testing.T) {
		resetGlobal()

		if err := exec.Command("git", "config", "grove.nerdFonts", "false").Run(); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = exec.Command("git", "config", "--unset", "grove.nerdFonts").Run() }()

		LoadFromGitConfig()
		if IsNerdFonts() {
			t.Error("Expected NerdFonts to be false from git config")
		}
	})

	t.Run("nerdFonts defaults to true when not in git config", func(t *testing.T) {
		resetGlobal()

		_ = exec.Command("git", "config", "--unset", "grove.nerdFonts").Run()

		LoadFromGitConfig()
		if !IsNerdFonts() {
			t.Error("Expected NerdFonts to be true (default) when not in git config")
		}
	})
}

func TestLoadGlobalConfigDoesNotDeadlockOnGitError(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	done := make(chan struct{})
	go func() {
		loadGlobalConfig(&FileConfig{})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("loadGlobalConfig deadlocked after git config failed")
	}
}

func TestMainLoadingSequence(t *testing.T) {
	cleanup := setupGitRepo(t)
	defer cleanup()

	t.Run("loads multiple config values in sequence", func(t *testing.T) {
		resetGlobal()

		// Set multiple config values
		if err := exec.Command("git", "config", "grove.plain", "true").Run(); err != nil {
			t.Fatal(err)
		}
		if err := exec.Command("git", "config", "grove.debug", "true").Run(); err != nil {
			t.Fatal(err)
		}
		if err := exec.Command("git", "config", "grove.nerdFonts", "false").Run(); err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = exec.Command("git", "config", "--unset", "grove.plain").Run()
			_ = exec.Command("git", "config", "--unset", "grove.debug").Run()
			_ = exec.Command("git", "config", "--unset", "grove.nerdFonts").Run()
		}()

		LoadFromGitConfig()

		// Verify all values were loaded
		if !IsPlain() {
			t.Error("Expected grove.plain to be loaded as true")
		}
		if !IsDebug() {
			t.Error("Expected grove.debug to be loaded as true")
		}
		if IsNerdFonts() {
			t.Error("Expected grove.nerdFonts to be loaded as false")
		}
	})

	t.Run("uses defaults for unset values", func(t *testing.T) {
		resetGlobal()

		// Only set plain, leave debug and nerdFonts unset
		if err := exec.Command("git", "config", "grove.plain", "true").Run(); err != nil {
			t.Fatal(err)
		}
		_ = exec.Command("git", "config", "--unset", "grove.debug").Run()
		_ = exec.Command("git", "config", "--unset", "grove.nerdFonts").Run()
		defer func() { _ = exec.Command("git", "config", "--unset", "grove.plain").Run() }()

		LoadFromGitConfig()

		if !IsPlain() {
			t.Error("Expected grove.plain to be true from git config")
		}
		if IsDebug() {
			t.Error("Expected grove.debug to be false (default)")
		}
		if !IsNerdFonts() {
			t.Error("Expected grove.nerdFonts to be true (default)")
		}
	})
}

func TestLoadRuntimeConfigPrecedence(t *testing.T) {
	cleanup := setupGitRepo(t)
	defer cleanup()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(FileName, []byte("plain = true\ndebug = true\n"), 0o644); err != nil { //nolint:gosec
		t.Fatal(err)
	}

	LoadRuntimeConfig(dir)
	if !IsPlain() || !IsDebug() {
		t.Fatal("expected TOML values to load into Global")
	}

	if err := exec.Command("git", "config", "grove.plain", "false").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "config", "grove.debug", "false").Run(); err != nil {
		t.Fatal(err)
	}
	LoadRuntimeConfig(dir)
	if IsPlain() || IsDebug() {
		t.Fatal("expected git config to override TOML")
	}

	SetPlain(true)
	SetDebug(true)
	if !IsPlain() || !IsDebug() {
		t.Fatal("expected flags to override git config and TOML")
	}
}

func TestDefaults(t *testing.T) {
	t.Run("DefaultConfig has expected values", func(t *testing.T) {
		if DefaultConfig.Plain != false {
			t.Error("Expected DefaultConfig.Plain to be false")
		}
		if DefaultConfig.Debug != false {
			t.Error("Expected DefaultConfig.Debug to be false")
		}
		if DefaultConfig.NerdFonts != true {
			t.Error("Expected DefaultConfig.NerdFonts to be true")
		}

		expectedPatterns := []string{
			".env",
			".env.keys",
			".env.local",
			".env.*.local",
			".envrc",
			".grove.toml",
			"docker-compose.override.yml",
		}

		if len(DefaultConfig.PreservePatterns) != len(expectedPatterns) {
			t.Errorf("Expected %d preserve patterns, got %d", len(expectedPatterns), len(DefaultConfig.PreservePatterns))
		}

		for i, expected := range expectedPatterns {
			if i >= len(DefaultConfig.PreservePatterns) || DefaultConfig.PreservePatterns[i] != expected {
				t.Errorf("Expected preserve pattern %d to be %q, got %q", i, expected, DefaultConfig.PreservePatterns[i])
			}
		}

		expectedExcludePatterns := []string{".cache", ".venv", "__pycache__", "build", "coverage", "dist", "node_modules", "out", "target", "vendor", "venv"}
		if len(DefaultConfig.PreserveExcludePatterns) != len(expectedExcludePatterns) {
			t.Errorf("Expected %d preserve exclude patterns, got %d", len(expectedExcludePatterns), len(DefaultConfig.PreserveExcludePatterns))
		}
		for i, expected := range expectedExcludePatterns {
			if i >= len(DefaultConfig.PreserveExcludePatterns) || DefaultConfig.PreserveExcludePatterns[i] != expected {
				t.Errorf("Expected preserve exclude pattern %d to be %q, got %q", i, expected, DefaultConfig.PreserveExcludePatterns[i])
			}
		}

		if len(DefaultConfig.PreserveDirectories) != 0 {
			t.Errorf("Expected DefaultConfig.PreserveDirectories to be empty, got %v", DefaultConfig.PreserveDirectories)
		}
	})
}

func TestLoadFromGitConfigWithDefaults(t *testing.T) {
	cleanup := setupGitRepo(t)
	defer cleanup()

	t.Run("uses defaults when no git config set", func(t *testing.T) {
		resetGlobal()

		_ = exec.Command("git", "config", "--unset", "grove.plain").Run()
		_ = exec.Command("git", "config", "--unset", "grove.debug").Run()
		LoadFromGitConfig()

		if Global.Plain != DefaultConfig.Plain {
			t.Errorf("Expected Plain to be %v (default), got %v", DefaultConfig.Plain, Global.Plain)
		}
		if Global.Debug != DefaultConfig.Debug {
			t.Errorf("Expected Debug to be %v (default), got %v", DefaultConfig.Debug, Global.Debug)
		}
	})

	t.Run("git config overrides defaults", func(t *testing.T) {
		resetGlobal()

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
}

func TestGetStaleThreshold(t *testing.T) {
	t.Run("returns default when Global is empty", func(t *testing.T) {
		resetGlobal()

		threshold := GetStaleThreshold()
		if threshold != DefaultConfig.StaleThreshold {
			t.Errorf("Expected %q, got %q", DefaultConfig.StaleThreshold, threshold)
		}
	})

	t.Run("returns Global threshold when set", func(t *testing.T) {
		resetGlobal()
		Global.StaleThreshold = "90d"

		threshold := GetStaleThreshold()
		if threshold != "90d" {
			t.Errorf("Expected '90d', got %q", threshold)
		}
	})
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"30 days", "30d", 30 * 24 * time.Hour, false},
		{"2 weeks", "2w", 14 * 24 * time.Hour, false},
		{"6 months", "6m", 180 * 24 * time.Hour, false},
		{"1 day", "1d", 24 * time.Hour, false},
		{"1 week", "1w", 7 * 24 * time.Hour, false},
		{"1 month", "1m", 30 * 24 * time.Hour, false},
		{"uppercase D", "30D", 30 * 24 * time.Hour, false},
		{"uppercase W", "2W", 14 * 24 * time.Hour, false},
		{"uppercase M", "6M", 180 * 24 * time.Hour, false},
		{"empty string", "", 0, true},
		{"invalid format", "abc", 0, true},
		{"missing number", "d", 0, true},
		{"invalid unit", "30x", 0, true},
		{"negative number", "-5d", 0, true},
		{"zero", "0d", 0, true},
		{"overflow", "106752d", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetAutoLockPatterns(t *testing.T) {
	t.Run("returns defaults when Global is empty", func(t *testing.T) {
		resetGlobal()

		patterns := GetAutoLockPatterns()
		if len(patterns) != len(DefaultConfig.AutoLockPatterns) {
			t.Errorf("Expected %d patterns, got %d", len(DefaultConfig.AutoLockPatterns), len(patterns))
		}

		for i, expected := range DefaultConfig.AutoLockPatterns {
			if i >= len(patterns) || patterns[i] != expected {
				t.Errorf("Expected pattern %d to be %q, got %q", i, expected, patterns[i])
			}
		}
	})

	t.Run("returns Global patterns when set", func(t *testing.T) {
		resetGlobal()
		customPatterns := []string{"develop", "release/*"}
		Global.AutoLockPatterns = customPatterns

		patterns := GetAutoLockPatterns()
		if len(patterns) != len(customPatterns) {
			t.Errorf("Expected %d patterns, got %d", len(customPatterns), len(patterns))
		}

		for i, expected := range customPatterns {
			if i >= len(patterns) || patterns[i] != expected {
				t.Errorf("Expected pattern %d to be %q, got %q", i, expected, patterns[i])
			}
		}
	})
}

func TestLoadFromGitConfigResetsAutoLockPatterns(t *testing.T) {
	cleanup := setupGitRepo(t)
	defer cleanup()

	t.Run("resets AutoLockPatterns to default when no git config", func(t *testing.T) {
		resetGlobal()
		// Set a custom value that shouldn't survive a reload
		Global.AutoLockPatterns = []string{"custom-branch"}

		// Ensure no git config is set
		_ = exec.Command("git", "config", "--unset-all", "grove.autoLock").Run()

		LoadFromGitConfig()

		// AutoLockPatterns should be reset to defaults, not retain "custom-branch"
		patterns := Global.AutoLockPatterns
		if len(patterns) == 1 && patterns[0] == "custom-branch" {
			t.Error("AutoLockPatterns was NOT reset to default - bug: stale value retained")
		}
		// Should match default patterns
		expected := DefaultConfig.AutoLockPatterns
		if len(patterns) != len(expected) {
			t.Errorf("Expected %d patterns (defaults), got %d: %v", len(expected), len(patterns), patterns)
		}
	})
}

func TestShouldAutoLock(t *testing.T) {
	t.Run("matches default patterns", func(t *testing.T) {
		resetGlobal()

		if !ShouldAutoLock("main") {
			t.Error("Expected 'main' to match default auto-lock patterns")
		}
		if !ShouldAutoLock("master") {
			t.Error("Expected 'master' to match default auto-lock patterns")
		}
		if ShouldAutoLock("feature") {
			t.Error("Expected 'feature' not to match default auto-lock patterns")
		}
	})

	t.Run("matches custom patterns", func(t *testing.T) {
		resetGlobal()
		Global.AutoLockPatterns = []string{"develop", "release/*", "production"}

		if !ShouldAutoLock("develop") {
			t.Error("Expected 'develop' to match")
		}
		if !ShouldAutoLock("production") {
			t.Error("Expected 'production' to match")
		}
		if ShouldAutoLock("main") {
			t.Error("Expected 'main' not to match custom patterns")
		}
	})

	t.Run("matches glob patterns", func(t *testing.T) {
		resetGlobal()
		Global.AutoLockPatterns = []string{"release/*", "hotfix/*"}

		if !ShouldAutoLock("release/v1.0") {
			t.Error("Expected 'release/v1.0' to match 'release/*'")
		}
		if !ShouldAutoLock("release/v2.0.1") {
			t.Error("Expected 'release/v2.0.1' to match 'release/*'")
		}
		if !ShouldAutoLock("hotfix/urgent") {
			t.Error("Expected 'hotfix/urgent' to match 'hotfix/*'")
		}
		if ShouldAutoLock("feature/new") {
			t.Error("Expected 'feature/new' not to match")
		}
		if ShouldAutoLock("release") {
			t.Error("Expected 'release' (without slash) not to match 'release/*'")
		}
	})
}

func TestGetTimeout(t *testing.T) {
	t.Run("returns zero when Global.Timeout is zero (no timeout)", func(t *testing.T) {
		resetGlobal()

		timeout := GetTimeout()
		if timeout != 0 {
			t.Errorf("Expected 0 (no timeout), got %v", timeout)
		}
	})

	t.Run("returns Global timeout when set", func(t *testing.T) {
		resetGlobal()
		Global.Timeout = 60 * time.Second

		timeout := GetTimeout()
		if timeout != 60*time.Second {
			t.Errorf("Expected 60s, got %v", timeout)
		}
	})

	t.Run("returns default after LoadFromGitConfig", func(t *testing.T) {
		// LoadFromGitConfig requires a git repo context
		cleanup := setupGitRepo(t)
		defer cleanup()

		resetGlobal()
		LoadFromGitConfig()

		timeout := GetTimeout()
		if timeout != DefaultConfig.Timeout {
			t.Errorf("Expected default %v, got %v", DefaultConfig.Timeout, timeout)
		}
	})
}
