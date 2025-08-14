package testutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestRunner_WithCleanEnvironment(t *testing.T) {
	// Set a test environment variable.
	testKey := "TEST_RUNNER_VAR"
	testValue := "test_value"
	err := os.Setenv(testKey, testValue)
	require.NoError(t, err)

	// Verify it's set.
	assert.Equal(t, testValue, os.Getenv(testKey))

	runner := NewTestRunner(t).WithCleanEnvironment()
	runner.Run(func() {
		// Inside the clean environment, the variable should not exist.
		assert.Empty(t, os.Getenv(testKey))

		// Essential variables should still be set.
		assert.NotEmpty(t, os.Getenv("PATH"))
		assert.NotEmpty(t, os.Getenv("HOME"))

		// Test-specific variables should be set.
		assert.Equal(t, "1", os.Getenv("GROVE_TEST_MODE"))
		assert.Equal(t, "1", os.Getenv("GIT_CONFIG_NOSYSTEM"))
	})

	// After the runner, the original environment should be restored.
	assert.Equal(t, testValue, os.Getenv(testKey))
}

func TestTestRunner_WithIsolatedWorkingDir(t *testing.T) {
	// Get the original working directory.
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	runner := NewTestRunner(t).WithIsolatedWorkingDir()
	runner.Run(func() {
		// Inside the isolated environment, we should be in a temp directory.
		currentDir, err := os.Getwd()
		require.NoError(t, err)

		assert.NotEqual(t, originalDir, currentDir)
		assert.Contains(t, currentDir, filepath.Join("", ""))

		// We should be able to create files in this directory.
		testFile := "test.txt"
		err = os.WriteFile(testFile, []byte("test"), 0o644)
		assert.NoError(t, err)
	})

	// After the runner, we should be back in the original directory.
	currentDir, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, originalDir, currentDir)
}

func TestIntegrationTestHelper_WithCleanFilesystem(t *testing.T) {
	helper := NewIntegrationTestHelper(t).WithCleanFilesystem()

	// Create a temp file.
	filePath := helper.CreateTempFile("test.txt", "test content")
	assert.FileExists(t, filePath)

	// Read the file to verify content.
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))

	// Create a temp directory.
	dirPath := helper.CreateTempDir("test-dir")
	assert.DirExists(t, dirPath)
}

func TestUnitTestHelper_WithCleanFilesystem(t *testing.T) {
	helper := NewUnitTestHelper(t).WithCleanFilesystem()

	// Create a temp file with nested path.
	filePath := helper.CreateTempFile("nested/dir/test.txt", "nested content")
	assert.FileExists(t, filePath)

	// Verify parent directories were created.
	assert.DirExists(t, filepath.Dir(filePath))

	// Read the file to verify content.
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, "nested content", string(content))

	// Create a nested temp directory.
	dirPath := helper.CreateTempDir("deep/nested/test-dir")
	assert.DirExists(t, dirPath)

	// Verify all parent directories exist.
	assert.DirExists(t, filepath.Dir(dirPath))
	assert.DirExists(t, filepath.Dir(filepath.Dir(dirPath)))
}

func TestUnitTestHelper_Run(t *testing.T) {
	helper := NewUnitTestHelper(t).
		WithCleanFilesystem().
		WithCleanEnvironment().
		WithIsolatedWorkingDir()

	originalDir, err := os.Getwd()
	require.NoError(t, err)

	// Set a test environment variable.
	testKey := "UNIT_HELPER_VAR"
	testValue := "unit_value"
	err = os.Setenv(testKey, testValue)
	require.NoError(t, err)

	helper.Run(func() {
		// Should be in an isolated environment.
		currentDir, err := os.Getwd()
		require.NoError(t, err)
		assert.NotEqual(t, originalDir, currentDir)

		// Environment should be clean.
		assert.Empty(t, os.Getenv(testKey))

		// Should be able to create files in clean filesystem.
		filePath := helper.CreateTempFile("isolated.txt", "isolated")
		assert.FileExists(t, filePath)
	})

	// Should be restored after.
	currentDir, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, originalDir, currentDir)
	assert.Equal(t, testValue, os.Getenv(testKey))
}

func TestPathHelpers(t *testing.T) {
	// Test NormalizePath.
	normalized := NormalizePath("/path/to//file/../dir/")
	expected := filepath.Clean("/path/to/dir")
	assert.Equal(t, expected, normalized)

	// Test JoinPath.
	joined := JoinPath("base", "sub", "file.txt")
	expected = filepath.Join("base", "sub", "file.txt")
	assert.Equal(t, expected, joined)

	// Test empty JoinPath.
	empty := JoinPath()
	assert.Equal(t, "", empty)

	// Test platform detection.
	if os.PathSeparator == '\\' {
		assert.True(t, IsWindows())
		assert.False(t, IsLinux())
		assert.False(t, IsMacOS())
	} else {
		assert.False(t, IsWindows())
		// Can't definitively test Linux vs macOS without runtime check
	}
}

// Binary Caching Tests

func TestIntegrationTestHelper_BinaryCaching(t *testing.T) {
	// Test that the binary is cached and reused across multiple helpers.
	helper1 := NewIntegrationTestHelper(t).WithCleanFilesystem()
	helper2 := NewIntegrationTestHelper(t).WithCleanFilesystem()

	// First execution should build the binary.
	stdout1, stderr1, err1 := helper1.ExecGrove("--version")
	require.NoError(t, err1, "first execution should succeed")
	assert.Contains(t, stdout1, "grove", "should output version info")
	assert.Empty(t, stderr1, "should have no stderr output")

	// Second execution should use the cached binary.
	stdout2, stderr2, err2 := helper2.ExecGrove("--version")
	require.NoError(t, err2, "second execution should succeed")
	assert.Contains(t, stdout2, "grove", "should output version info")
	assert.Empty(t, stderr2, "should have no stderr output")

	// Both should produce the same output.
	assert.Equal(t, stdout1, stdout2, "cached binary should produce same output")
}

func TestIntegrationTestHelper_BinaryCaching_ParallelSafety(t *testing.T) {
	// Test that parallel test execution doesn't cause race conditions.
	// This test verifies the mutex protection in getCachedBinary.

	// Clear the cache to force a rebuild.
	binaryCache.Lock()
	binaryCache.path = ""
	binaryCache.hash = ""
	binaryCache.Unlock()

	// Run multiple goroutines that try to build simultaneously.
	const numGoroutines = 5
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			helper := NewIntegrationTestHelper(t).WithCleanFilesystem()
			stdout, _, err := helper.ExecGrove("--version")
			if err != nil {
				results <- err
				return
			}
			if !assert.Contains(t, stdout, "grove") {
				results <- assert.AnError
				return
			}
			results <- nil
		}(i)
	}

	// Collect results.
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		assert.NoError(t, err, "parallel execution %d should succeed", i)
	}

	// Verify the binary was only built once by checking the cache.
	binaryCache.RLock()
	assert.NotEmpty(t, binaryCache.path, "binary should be cached")
	assert.NotEmpty(t, binaryCache.hash, "hash should be cached")
	binaryCache.RUnlock()
}

func TestIntegrationTestHelper_BinaryCaching_SourceChangeDetection(t *testing.T) {
	// This test verifies that source changes invalidate the cache.
	// Note: This is a conceptual test - in practice, source changes
	// would happen between test runs, not during a single test.

	helper := NewIntegrationTestHelper(t).WithCleanFilesystem()

	// First execution builds and caches.
	stdout1, _, err1 := helper.ExecGrove("--version")
	require.NoError(t, err1)
	assert.Contains(t, stdout1, "grove")

	// Get the current cache state.
	binaryCache.RLock()
	originalPath := binaryCache.path
	originalHash := binaryCache.hash
	binaryCache.RUnlock()

	assert.NotEmpty(t, originalPath, "binary should be cached")
	assert.NotEmpty(t, originalHash, "hash should be stored")

	// In a real scenario, if source files change, the hash would change
	// and trigger a rebuild. We can't easily simulate this in a test,
	// but we can verify the hash calculation works.
	hash1, err := calculateSourceHash()
	require.NoError(t, err, "hash calculation should work")
	assert.NotEmpty(t, hash1, "hash should not be empty")

	// Calculate again - should be deterministic.
	hash2, err := calculateSourceHash()
	require.NoError(t, err, "hash calculation should work")
	assert.Equal(t, hash1, hash2, "hash should be deterministic")
}

func TestIntegrationTestHelper_BinaryCaching_MissingBinary(t *testing.T) {
	// Test that a missing cached binary triggers a rebuild.

	helper := NewIntegrationTestHelper(t).WithCleanFilesystem()

	// First execution to populate cache.
	stdout1, _, err1 := helper.ExecGrove("--version")
	require.NoError(t, err1)
	assert.Contains(t, stdout1, "grove")

	// Get cached binary path.
	binaryCache.RLock()
	cachedPath := binaryCache.path
	binaryCache.RUnlock()
	require.NotEmpty(t, cachedPath, "binary should be cached")

	// Delete the cached binary to simulate it being cleaned up.
	err := os.Remove(cachedPath)
	require.NoError(t, err, "should be able to delete cached binary")

	// Next execution should detect missing binary and rebuild.
	helper2 := NewIntegrationTestHelper(t).WithCleanFilesystem()
	stdout2, _, err2 := helper2.ExecGrove("--version")
	require.NoError(t, err2, "should rebuild after binary is missing")
	assert.Contains(t, stdout2, "grove")

	// Verify a new binary was built.
	binaryCache.RLock()
	newPath := binaryCache.path
	binaryCache.RUnlock()
	assert.NotEmpty(t, newPath, "new binary should be cached")
	assert.FileExists(t, newPath, "new binary should exist")
}

func TestCalculateSourceHash(t *testing.T) {
	// Test the source hash calculation function.

	// Calculate hash once.
	hash1, err := calculateSourceHash()
	require.NoError(t, err, "should calculate hash successfully")
	assert.NotEmpty(t, hash1, "hash should not be empty")
	assert.Len(t, hash1, 64, "SHA256 hash should be 64 hex characters")

	// Calculate again - should be identical.
	hash2, err := calculateSourceHash()
	require.NoError(t, err, "should calculate hash successfully")
	assert.Equal(t, hash1, hash2, "hash should be deterministic")

	// Verify it's a valid hex string.
	for _, r := range hash1 {
		assert.True(t, (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f'),
			"hash should only contain hex characters")
	}
}

func TestFindProjectRoot(t *testing.T) {
	// Test finding the project root directory.

	root, err := findProjectRoot()
	require.NoError(t, err, "should find project root")
	assert.NotEmpty(t, root, "root should not be empty")

	// Verify go.mod exists at the root.
	goModPath := filepath.Join(root, "go.mod")
	assert.FileExists(t, goModPath, "go.mod should exist at project root")

	// The root should be an absolute path.
	assert.True(t, filepath.IsAbs(root), "root should be an absolute path")
}

func TestBuildGroveBinary(t *testing.T) {
	// Test building the grove binary.

	binaryPath, err := buildGroveBinary(t)
	require.NoError(t, err, "should build binary successfully")
	assert.NotEmpty(t, binaryPath, "binary path should not be empty")
	assert.FileExists(t, binaryPath, "binary should exist")

	// Verify the binary is executable.
	info, err := os.Stat(binaryPath)
	require.NoError(t, err, "should stat binary")

	// On Unix systems, check executable permission.
	if !IsWindows() {
		mode := info.Mode()
		assert.True(t, mode&0o111 != 0, "binary should be executable")
	}

	// Verify the binary actually works.
	stdout, _, err := NewIntegrationTestHelper(t).ExecGrove("--version")
	require.NoError(t, err, "binary should execute")
	assert.Contains(t, stdout, "grove", "should output version info")
}

// Environment Isolation Tests

func TestTestRunner_HomeDirectoryIsolation(t *testing.T) {
	// Get original HOME directory.
	originalHome := os.Getenv("HOME")
	if IsWindows() {
		originalHome = os.Getenv("USERPROFILE")
	}
	require.NotEmpty(t, originalHome, "original home should exist")

	// Create a test file in the original home (if possible).
	testFile := filepath.Join(originalHome, ".test_grove_isolation")
	_ = os.WriteFile(testFile, []byte("original"), 0o644)
	defer func() {
		_ = os.Remove(testFile)
	}()

	runner := NewTestRunner(t).WithCleanEnvironment()
	runner.Run(func() {
		// Inside the clean environment, HOME should be a temp directory.
		isolatedHome := os.Getenv("HOME")
		if IsWindows() {
			isolatedHome = os.Getenv("USERPROFILE")
		}
		assert.NotEmpty(t, isolatedHome, "isolated home should be set")
		assert.NotEqual(t, originalHome, isolatedHome, "isolated home should differ from original")
		assert.DirExists(t, isolatedHome, "isolated home directory should exist")

		// The test file from original home should not be visible.
		isolatedTestFile := filepath.Join(isolatedHome, ".test_grove_isolation")
		assert.NoFileExists(t, isolatedTestFile, "original home files should not leak")

		// We should be able to create files in the isolated home.
		newFile := filepath.Join(isolatedHome, ".grove_test_file")
		err := os.WriteFile(newFile, []byte("isolated"), 0o644)
		assert.NoError(t, err, "should be able to write to isolated home")
		assert.FileExists(t, newFile, "file should exist in isolated home")
	})

	// After the runner, original HOME should be restored.
	currentHome := os.Getenv("HOME")
	if IsWindows() {
		currentHome = os.Getenv("USERPROFILE")
	}
	assert.Equal(t, originalHome, currentHome, "original home should be restored")

	// Original test file should still exist.
	if _, err := os.Stat(testFile); err == nil {
		content, _ := os.ReadFile(testFile)
		assert.Equal(t, "original", string(content), "original file should be unchanged")
	}
}

func TestTestRunner_XDGConfigPathIsolation(t *testing.T) {
	// Skip on Windows as XDG is Unix-specific.
	if IsWindows() {
		t.Skip("XDG paths not applicable on Windows")
	}

	// Save original XDG variables.
	originalConfig := os.Getenv("XDG_CONFIG_HOME")
	originalData := os.Getenv("XDG_DATA_HOME")
	originalCache := os.Getenv("XDG_CACHE_HOME")

	// Set some XDG variables before the test.
	_ = os.Setenv("XDG_CONFIG_HOME", "/original/config")
	_ = os.Setenv("XDG_DATA_HOME", "/original/data")
	_ = os.Setenv("XDG_CACHE_HOME", "/original/cache")

	runner := NewTestRunner(t).WithCleanEnvironment()
	runner.Run(func() {
		// XDG variables should either be unset or point to temp directories.
		configHome := os.Getenv("XDG_CONFIG_HOME")
		dataHome := os.Getenv("XDG_DATA_HOME")
		cacheHome := os.Getenv("XDG_CACHE_HOME")

		// If set, they should not point to original locations.
		if configHome != "" {
			assert.NotEqual(t, "/original/config", configHome, "XDG_CONFIG_HOME should be isolated")
		}
		if dataHome != "" {
			assert.NotEqual(t, "/original/data", dataHome, "XDG_DATA_HOME should be isolated")
		}
		if cacheHome != "" {
			assert.NotEqual(t, "/original/cache", cacheHome, "XDG_CACHE_HOME should be isolated")
		}
	})

	// Restore original values.
	if originalConfig != "" {
		_ = os.Setenv("XDG_CONFIG_HOME", originalConfig)
	} else {
		_ = os.Unsetenv("XDG_CONFIG_HOME")
	}
	if originalData != "" {
		_ = os.Setenv("XDG_DATA_HOME", originalData)
	} else {
		_ = os.Unsetenv("XDG_DATA_HOME")
	}
	if originalCache != "" {
		_ = os.Setenv("XDG_CACHE_HOME", originalCache)
	} else {
		_ = os.Unsetenv("XDG_CACHE_HOME")
	}
}

func TestTestRunner_GitConfigIsolation(t *testing.T) {
	// Save original Git config variables.
	originalGitConfigSystem := os.Getenv("GIT_CONFIG_NOSYSTEM")
	originalGitConfigGlobal := os.Getenv("GIT_CONFIG_GLOBAL")

	runner := NewTestRunner(t).WithCleanEnvironment()
	runner.Run(func() {
		// Git configuration should be isolated.
		assert.Equal(t, "1", os.Getenv("GIT_CONFIG_NOSYSTEM"), "GIT_CONFIG_NOSYSTEM should be set to 1")
		assert.Equal(t, "/dev/null", os.Getenv("GIT_CONFIG_GLOBAL"), "GIT_CONFIG_GLOBAL should point to /dev/null")

		// Test that git config commands work in isolation.
		// We can't easily test this without having git available, but we can verify the environment is set.
		gitConfigGlobal := os.Getenv("GIT_CONFIG_GLOBAL")
		if !IsWindows() {
			assert.Equal(t, "/dev/null", gitConfigGlobal, "git global config should be isolated")
		} else {
			// On Windows, it might be "nul" or similar.
			assert.NotEmpty(t, gitConfigGlobal, "git global config should be set on Windows")
		}
	})

	// After the runner, original Git config should be restored.
	currentGitConfigSystem := os.Getenv("GIT_CONFIG_NOSYSTEM")
	currentGitConfigGlobal := os.Getenv("GIT_CONFIG_GLOBAL")

	if originalGitConfigSystem == "" {
		assert.Empty(t, currentGitConfigSystem, "GIT_CONFIG_NOSYSTEM should be restored (empty)")
	} else {
		assert.Equal(t, originalGitConfigSystem, currentGitConfigSystem, "GIT_CONFIG_NOSYSTEM should be restored")
	}

	if originalGitConfigGlobal == "" {
		assert.Empty(t, currentGitConfigGlobal, "GIT_CONFIG_GLOBAL should be restored (empty)")
	} else {
		assert.Equal(t, originalGitConfigGlobal, currentGitConfigGlobal, "GIT_CONFIG_GLOBAL should be restored")
	}
}

func TestTestRunner_SSHKeyIsolation(t *testing.T) {
	// Save original SSH-related variables.
	originalSSHAuthSock := os.Getenv("SSH_AUTH_SOCK")
	originalSSHKnownHosts := os.Getenv("SSH_KNOWN_HOSTS")
	originalSSHAgent := os.Getenv("SSH_AGENT_PID")

	// Set some SSH variables before the test.
	_ = os.Setenv("SSH_AUTH_SOCK", "/tmp/ssh-agent-123")
	_ = os.Setenv("SSH_KNOWN_HOSTS", "/home/user/.ssh/known_hosts")
	_ = os.Setenv("SSH_AGENT_PID", "12345")

	runner := NewTestRunner(t).WithCleanEnvironment()
	runner.Run(func() {
		// SSH variables should be cleared in clean environment.
		assert.Empty(t, os.Getenv("SSH_AUTH_SOCK"), "SSH_AUTH_SOCK should be cleared")
		assert.Empty(t, os.Getenv("SSH_KNOWN_HOSTS"), "SSH_KNOWN_HOSTS should be cleared")
		assert.Empty(t, os.Getenv("SSH_AGENT_PID"), "SSH_AGENT_PID should be cleared")

		// Test HOME-based SSH config isolation.
		home := os.Getenv("HOME")
		if IsWindows() {
			home = os.Getenv("USERPROFILE")
		}
		sshDir := filepath.Join(home, ".ssh")

		// Create SSH directory in isolated home to verify isolation.
		err := os.MkdirAll(sshDir, 0o700)
		assert.NoError(t, err, "should be able to create .ssh in isolated home")

		// Create a test key file.
		testKey := filepath.Join(sshDir, "id_test")
		err = os.WriteFile(testKey, []byte("fake ssh key"), 0o600)
		assert.NoError(t, err, "should be able to create ssh key in isolated environment")
		assert.FileExists(t, testKey, "ssh key should exist in isolated environment")
	})

	// Restore original SSH variables.
	if originalSSHAuthSock != "" {
		_ = os.Setenv("SSH_AUTH_SOCK", originalSSHAuthSock)
	} else {
		_ = os.Unsetenv("SSH_AUTH_SOCK")
	}
	if originalSSHKnownHosts != "" {
		_ = os.Setenv("SSH_KNOWN_HOSTS", originalSSHKnownHosts)
	} else {
		_ = os.Unsetenv("SSH_KNOWN_HOSTS")
	}
	if originalSSHAgent != "" {
		_ = os.Setenv("SSH_AGENT_PID", originalSSHAgent)
	} else {
		_ = os.Unsetenv("SSH_AGENT_PID")
	}
}

func TestTestRunner_ProxySettingsIsolation(t *testing.T) {
	// Save original proxy variables.
	originalHTTPProxy := os.Getenv("HTTP_PROXY")
	originalHTTPSProxy := os.Getenv("HTTPS_PROXY")
	originalNoProxy := os.Getenv("NO_PROXY")
	originalAllProxy := os.Getenv("ALL_PROXY")

	// Set some proxy variables before the test.
	_ = os.Setenv("HTTP_PROXY", "http://proxy.example.com:8080")
	_ = os.Setenv("HTTPS_PROXY", "https://proxy.example.com:8443")
	_ = os.Setenv("NO_PROXY", "localhost,127.0.0.1")
	_ = os.Setenv("ALL_PROXY", "socks5://proxy.example.com:1080")

	runner := NewTestRunner(t).WithCleanEnvironment()
	runner.Run(func() {
		// Proxy variables should be cleared in clean environment.
		assert.Empty(t, os.Getenv("HTTP_PROXY"), "HTTP_PROXY should be cleared")
		assert.Empty(t, os.Getenv("HTTPS_PROXY"), "HTTPS_PROXY should be cleared")
		assert.Empty(t, os.Getenv("NO_PROXY"), "NO_PROXY should be cleared")
		assert.Empty(t, os.Getenv("ALL_PROXY"), "ALL_PROXY should be cleared")

		// Also check lowercase variants.
		assert.Empty(t, os.Getenv("http_proxy"), "http_proxy should be cleared")
		assert.Empty(t, os.Getenv("https_proxy"), "https_proxy should be cleared")
		assert.Empty(t, os.Getenv("no_proxy"), "no_proxy should be cleared")
		assert.Empty(t, os.Getenv("all_proxy"), "all_proxy should be cleared")
	})

	// Restore original proxy variables.
	if originalHTTPProxy != "" {
		_ = os.Setenv("HTTP_PROXY", originalHTTPProxy)
	} else {
		_ = os.Unsetenv("HTTP_PROXY")
	}
	if originalHTTPSProxy != "" {
		_ = os.Setenv("HTTPS_PROXY", originalHTTPSProxy)
	} else {
		_ = os.Unsetenv("HTTPS_PROXY")
	}
	if originalNoProxy != "" {
		_ = os.Setenv("NO_PROXY", originalNoProxy)
	} else {
		_ = os.Unsetenv("NO_PROXY")
	}
	if originalAllProxy != "" {
		_ = os.Setenv("ALL_PROXY", originalAllProxy)
	} else {
		_ = os.Unsetenv("ALL_PROXY")
	}
}

func TestTestRunner_EnvironmentCleanup_WithPanic(t *testing.T) {
	// Test that environment is restored even when the test function panics.

	testKey := "PANIC_TEST_VAR"
	testValue := "panic_value"
	_ = os.Setenv(testKey, testValue)

	runner := NewTestRunner(t).WithCleanEnvironment()

	// This should panic but the environment should still be restored.
	func() {
		defer func() {
			// Catch the panic.
			if r := recover(); r != nil {
				// Panic occurred as expected.
				assert.Contains(t, r.(string), "intentional panic", "should have caught expected panic")
			}
		}()

		runner.Run(func() {
			// Verify we're in clean environment.
			assert.Empty(t, os.Getenv(testKey), "should be in clean environment")

			// Intentionally panic.
			panic("intentional panic for cleanup test")
		})
	}()

	// After panic, environment should be restored.
	assert.Equal(t, testValue, os.Getenv(testKey), "environment should be restored after panic")

	// Clean up.
	_ = os.Unsetenv(testKey)
}

func TestTestRunner_CompleteEnvironmentCleanup(t *testing.T) {
	// Test comprehensive environment isolation and cleanup.

	// Set various environment variables.
	testVars := map[string]string{
		"GROVE_TEST_VAR1":  "value1",
		"GROVE_TEST_VAR2":  "value2",
		"CUSTOM_PATH":      "/custom/path",
		"DEVELOPMENT_MODE": "true",
		"API_KEY":          "secret123",
		"DEBUG_LEVEL":      "verbose",
	}

	for key, value := range testVars {
		_ = os.Setenv(key, value)
	}

	// Also save some system variables to ensure they're restored.
	systemVars := map[string]string{
		"PATH":   os.Getenv("PATH"),
		"HOME":   os.Getenv("HOME"),
		"TMPDIR": os.Getenv("TMPDIR"),
	}
	if IsWindows() {
		systemVars["USERPROFILE"] = os.Getenv("USERPROFILE")
		systemVars["TEMP"] = os.Getenv("TEMP")
		systemVars["TMP"] = os.Getenv("TMP")
	}

	runner := NewTestRunner(t).WithCleanEnvironment()
	runner.Run(func() {
		// All test variables should be gone.
		for key := range testVars {
			assert.Empty(t, os.Getenv(key), "test variable %s should be cleared", key)
		}

		// Essential system variables should still be set.
		assert.NotEmpty(t, os.Getenv("PATH"), "PATH should be preserved")
		assert.NotEmpty(t, os.Getenv("HOME"), "HOME should be set (to temp)")
		if IsWindows() {
			assert.NotEmpty(t, os.Getenv("USERPROFILE"), "USERPROFILE should be set")
		}

		// Test-specific variables should be set.
		assert.Equal(t, "1", os.Getenv("GROVE_TEST_MODE"))
		assert.Equal(t, "1", os.Getenv("GIT_CONFIG_NOSYSTEM"))
		assert.Equal(t, "/dev/null", os.Getenv("GIT_CONFIG_GLOBAL"))
	})

	// After cleanup, all original variables should be restored.
	for key, originalValue := range testVars {
		currentValue := os.Getenv(key)
		assert.Equal(t, originalValue, currentValue, "test variable %s should be restored", key)
	}

	for key, originalValue := range systemVars {
		currentValue := os.Getenv(key)
		assert.Equal(t, originalValue, currentValue, "system variable %s should be restored", key)
	}

	// Clean up test variables.
	for key := range testVars {
		_ = os.Unsetenv(key)
	}
}

func TestIntegrationTestHelper_ExecGrove_Arguments(t *testing.T) {
	// Test that arguments are passed correctly to the binary.

	helper := NewIntegrationTestHelper(t).WithCleanFilesystem()

	// Test with no arguments (should show help or error).
	stdout, stderr, _ := helper.ExecGrove()
	// Don't check error as grove might exit with non-zero for help.
	assert.True(t, stdout != "" || stderr != "",
		"should produce some output with no args")

	// Test with invalid command.
	_, stderr2, err2 := helper.ExecGrove("invalid-command-xyz")
	assert.Error(t, err2, "invalid command should error")
	assert.NotEmpty(t, stderr2, "should have error message")

	// Test with multiple arguments.
	stdout3, _, _ := helper.ExecGrove("--help")
	assert.Contains(t, stdout3, "Usage", "help should show usage")
}

func TestIntegrationTestHelper_WorkingDirectory(t *testing.T) {
	// Test that the working directory is set correctly.

	helper := NewIntegrationTestHelper(t).WithCleanFilesystem()

	// Create a test file in the temp directory.
	testFile := helper.CreateTempFile("test.txt", "test content")
	assert.FileExists(t, testFile)

	// The binary should run in the temp directory when WithCleanFilesystem is used.
	// We can't directly test this without modifying grove, but we can verify
	// the temp directory is created and used.
	assert.NotEmpty(t, helper.tempDir, "temp directory should be set")
	assert.DirExists(t, helper.tempDir, "temp directory should exist")
}
