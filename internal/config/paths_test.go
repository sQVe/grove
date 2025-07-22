//go:build !integration
// +build !integration

package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfigPaths(t *testing.T) {
	// Save original environment
	originalGROVE_CONFIG := os.Getenv("GROVE_CONFIG")
	originalHOME := os.Getenv("HOME")
	originalUSERPROFILE := os.Getenv("USERPROFILE")
	originalAPPDATA := os.Getenv("APPDATA")
	originalXDG_CONFIG_HOME := os.Getenv("XDG_CONFIG_HOME")

	defer func() {
		// Restore original environment
		if originalGROVE_CONFIG != "" {
			_ = os.Setenv("GROVE_CONFIG", originalGROVE_CONFIG)
		} else {
			_ = os.Unsetenv("GROVE_CONFIG")
		}
		if originalHOME != "" {
			_ = os.Setenv("HOME", originalHOME)
		} else {
			_ = os.Unsetenv("HOME")
		}
		if originalUSERPROFILE != "" {
			_ = os.Setenv("USERPROFILE", originalUSERPROFILE)
		} else {
			_ = os.Unsetenv("USERPROFILE")
		}
		if originalAPPDATA != "" {
			_ = os.Setenv("APPDATA", originalAPPDATA)
		} else {
			_ = os.Unsetenv("APPDATA")
		}
		if originalXDG_CONFIG_HOME != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", originalXDG_CONFIG_HOME)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	// Test with GROVE_CONFIG environment variable
	testConfigPath := "/tmp/test-config"
	require.NoError(t, os.Setenv("GROVE_CONFIG", filepath.Join(testConfigPath, "config.toml")))

	paths := GetConfigPaths()
	assert.NotEmpty(t, paths)
	assert.Equal(t, testConfigPath, paths[0])

	// Test without GROVE_CONFIG
	require.NoError(t, os.Unsetenv("GROVE_CONFIG"))
	paths = GetConfigPaths()
	assert.NotEmpty(t, paths)

	// Current directory should be in the paths
	cwd, _ := os.Getwd()
	assert.Contains(t, paths, cwd)
}

func TestGetUserConfigDir(t *testing.T) {
	// Save original environment
	originalHOME := os.Getenv("HOME")
	originalUSERPROFILE := os.Getenv("USERPROFILE")
	originalAPPDATA := os.Getenv("APPDATA")
	originalXDG_CONFIG_HOME := os.Getenv("XDG_CONFIG_HOME")

	defer func() {
		// Restore original environment
		if originalHOME != "" {
			_ = os.Setenv("HOME", originalHOME)
		} else {
			_ = os.Unsetenv("HOME")
		}
		if originalUSERPROFILE != "" {
			_ = os.Setenv("USERPROFILE", originalUSERPROFILE)
		} else {
			_ = os.Unsetenv("USERPROFILE")
		}
		if originalAPPDATA != "" {
			_ = os.Setenv("APPDATA", originalAPPDATA)
		} else {
			_ = os.Unsetenv("APPDATA")
		}
		if originalXDG_CONFIG_HOME != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", originalXDG_CONFIG_HOME)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	switch runtime.GOOS {
	case osWindows:
		// Test Windows path
		require.NoError(t, os.Setenv("APPDATA", `C:\Users\test\AppData\Roaming`))
		require.NoError(t, os.Unsetenv("HOME"))
		require.NoError(t, os.Unsetenv("USERPROFILE"))
		require.NoError(t, os.Unsetenv("XDG_CONFIG_HOME"))

		configDir := getUserConfigDir()
		assert.Equal(t, `C:\Users\test\AppData\Roaming\grove`, configDir)

		// Test fallback to USERPROFILE
		require.NoError(t, os.Unsetenv("APPDATA"))
		require.NoError(t, os.Setenv("USERPROFILE", `C:\Users\test`))

		configDir = getUserConfigDir()
		assert.Equal(t, `C:\Users\test\AppData\Roaming\grove`, configDir)

	case osDarwin:
		// Test macOS path
		require.NoError(t, os.Setenv("HOME", "/Users/test"))
		require.NoError(t, os.Unsetenv("APPDATA"))
		require.NoError(t, os.Unsetenv("USERPROFILE"))
		require.NoError(t, os.Unsetenv("XDG_CONFIG_HOME"))

		configDir := getUserConfigDir()
		assert.Equal(t, "/Users/test/Library/Application Support/grove", configDir)

	default:
		// Test Linux path with XDG_CONFIG_HOME
		require.NoError(t, os.Setenv("XDG_CONFIG_HOME", "/home/test/.config"))
		require.NoError(t, os.Unsetenv("HOME"))
		require.NoError(t, os.Unsetenv("APPDATA"))
		require.NoError(t, os.Unsetenv("USERPROFILE"))

		configDir := getUserConfigDir()
		assert.Equal(t, "/home/test/.config/grove", configDir)

		// Test Linux path with HOME fallback
		require.NoError(t, os.Unsetenv("XDG_CONFIG_HOME"))
		require.NoError(t, os.Setenv("HOME", "/home/test"))

		configDir = getUserConfigDir()
		assert.Equal(t, "/home/test/.config/grove", configDir)
	}
}

func TestGetWindowsConfigDir(t *testing.T) {
	if runtime.GOOS != osWindows {
		t.Skip("Windows-specific test")
	}

	// Save original environment
	originalAPPDATA := os.Getenv("APPDATA")
	originalUSERPROFILE := os.Getenv("USERPROFILE")

	defer func() {
		if originalAPPDATA != "" {
			_ = os.Setenv("APPDATA", originalAPPDATA)
		} else {
			_ = os.Unsetenv("APPDATA")
		}
		if originalUSERPROFILE != "" {
			_ = os.Setenv("USERPROFILE", originalUSERPROFILE)
		} else {
			_ = os.Unsetenv("USERPROFILE")
		}
	}()

	// Test with APPDATA
	require.NoError(t, os.Setenv("APPDATA", `C:\Users\test\AppData\Roaming`))
	require.NoError(t, os.Unsetenv("USERPROFILE"))

	configDir := getWindowsConfigDir()
	assert.Equal(t, `C:\Users\test\AppData\Roaming\grove`, configDir)

	// Test with USERPROFILE fallback
	require.NoError(t, os.Unsetenv("APPDATA"))
	require.NoError(t, os.Setenv("USERPROFILE", `C:\Users\test`))

	configDir = getWindowsConfigDir()
	assert.Equal(t, `C:\Users\test\AppData\Roaming\grove`, configDir)

	// Test with no environment variables
	require.NoError(t, os.Unsetenv("APPDATA"))
	require.NoError(t, os.Unsetenv("USERPROFILE"))

	configDir = getWindowsConfigDir()
	assert.Equal(t, "", configDir)
}

func TestGetMacOSConfigDir(t *testing.T) {
	if runtime.GOOS != osDarwin {
		t.Skip("macOS-specific test")
	}

	// Save original environment
	originalHOME := os.Getenv("HOME")

	defer func() {
		if originalHOME != "" {
			_ = os.Setenv("HOME", originalHOME)
		} else {
			_ = os.Unsetenv("HOME")
		}
	}()

	// Test with HOME
	require.NoError(t, os.Setenv("HOME", "/Users/test"))

	configDir := getMacOSConfigDir()
	assert.Equal(t, "/Users/test/Library/Application Support/grove", configDir)

	// Test with no HOME
	require.NoError(t, os.Unsetenv("HOME"))

	configDir = getMacOSConfigDir()
	assert.Equal(t, "", configDir)
}

func TestGetLinuxConfigDir(t *testing.T) {
	if runtime.GOOS == osWindows || runtime.GOOS == osDarwin {
		t.Skip("Linux-specific test")
	}

	// Save original environment
	originalXDG_CONFIG_HOME := os.Getenv("XDG_CONFIG_HOME")
	originalHOME := os.Getenv("HOME")

	defer func() {
		if originalXDG_CONFIG_HOME != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", originalXDG_CONFIG_HOME)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
		if originalHOME != "" {
			_ = os.Setenv("HOME", originalHOME)
		} else {
			_ = os.Unsetenv("HOME")
		}
	}()

	// Test with XDG_CONFIG_HOME
	require.NoError(t, os.Setenv("XDG_CONFIG_HOME", "/home/test/.config"))
	require.NoError(t, os.Unsetenv("HOME"))

	configDir := getLinuxConfigDir()
	assert.Equal(t, "/home/test/.config/grove", configDir)

	// Test with HOME fallback
	require.NoError(t, os.Unsetenv("XDG_CONFIG_HOME"))
	require.NoError(t, os.Setenv("HOME", "/home/test"))

	configDir = getLinuxConfigDir()
	assert.Equal(t, "/home/test/.config/grove", configDir)

	// Test with no environment variables
	require.NoError(t, os.Unsetenv("XDG_CONFIG_HOME"))
	require.NoError(t, os.Unsetenv("HOME"))

	configDir = getLinuxConfigDir()
	assert.Equal(t, "", configDir)
}

func TestGetHomeDir(t *testing.T) {
	// Save original environment
	originalHOME := os.Getenv("HOME")
	originalUSERPROFILE := os.Getenv("USERPROFILE")

	defer func() {
		if originalHOME != "" {
			_ = os.Setenv("HOME", originalHOME)
		} else {
			_ = os.Unsetenv("HOME")
		}
		if originalUSERPROFILE != "" {
			_ = os.Setenv("USERPROFILE", originalUSERPROFILE)
		} else {
			_ = os.Unsetenv("USERPROFILE")
		}
	}()

	// Test with HOME
	require.NoError(t, os.Setenv("HOME", "/home/test"))
	require.NoError(t, os.Unsetenv("USERPROFILE"))

	homeDir := getHomeDir()
	assert.Equal(t, "/home/test", homeDir)

	// Test with USERPROFILE fallback
	require.NoError(t, os.Unsetenv("HOME"))
	require.NoError(t, os.Setenv("USERPROFILE", "/Users/test"))

	homeDir = getHomeDir()
	assert.Equal(t, "/Users/test", homeDir)

	// Test with no environment variables
	require.NoError(t, os.Unsetenv("HOME"))
	require.NoError(t, os.Unsetenv("USERPROFILE"))

	homeDir = getHomeDir()
	assert.Equal(t, "", homeDir)
}

func TestGetDefaultConfigPath(t *testing.T) {
	// Save original environment
	originalHOME := os.Getenv("HOME")
	originalUSERPROFILE := os.Getenv("USERPROFILE")
	originalAPPDATA := os.Getenv("APPDATA")
	originalXDG_CONFIG_HOME := os.Getenv("XDG_CONFIG_HOME")

	defer func() {
		if originalHOME != "" {
			_ = os.Setenv("HOME", originalHOME)
		} else {
			_ = os.Unsetenv("HOME")
		}
		if originalUSERPROFILE != "" {
			_ = os.Setenv("USERPROFILE", originalUSERPROFILE)
		} else {
			_ = os.Unsetenv("USERPROFILE")
		}
		if originalAPPDATA != "" {
			_ = os.Setenv("APPDATA", originalAPPDATA)
		} else {
			_ = os.Unsetenv("APPDATA")
		}
		if originalXDG_CONFIG_HOME != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", originalXDG_CONFIG_HOME)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	// Test with valid home directory
	switch runtime.GOOS {
	case osWindows:
		require.NoError(t, os.Setenv("APPDATA", `C:\Users\test\AppData\Roaming`))
		require.NoError(t, os.Unsetenv("HOME"))
		require.NoError(t, os.Unsetenv("USERPROFILE"))
		require.NoError(t, os.Unsetenv("XDG_CONFIG_HOME"))

		configPath := GetDefaultConfigPath()
		assert.Equal(t, `C:\Users\test\AppData\Roaming\grove\config.toml`, configPath)
	case osDarwin:
		require.NoError(t, os.Setenv("HOME", "/Users/test"))
		require.NoError(t, os.Unsetenv("APPDATA"))
		require.NoError(t, os.Unsetenv("USERPROFILE"))
		require.NoError(t, os.Unsetenv("XDG_CONFIG_HOME"))

		configPath := GetDefaultConfigPath()
		assert.Equal(t, "/Users/test/Library/Application Support/grove/config.toml", configPath)
	default:
		require.NoError(t, os.Setenv("XDG_CONFIG_HOME", "/home/test/.config"))
		require.NoError(t, os.Unsetenv("HOME"))
		require.NoError(t, os.Unsetenv("APPDATA"))
		require.NoError(t, os.Unsetenv("USERPROFILE"))

		configPath := GetDefaultConfigPath()
		assert.Equal(t, "/home/test/.config/grove/config.toml", configPath)
	}

	// Test with no config directory
	require.NoError(t, os.Unsetenv("HOME"))
	require.NoError(t, os.Unsetenv("USERPROFILE"))
	require.NoError(t, os.Unsetenv("APPDATA"))
	require.NoError(t, os.Unsetenv("XDG_CONFIG_HOME"))

	configPath := GetDefaultConfigPath()
	assert.Equal(t, "", configPath)
}

func TestEnsureConfigDir(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "grove-config-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Save original environment
	originalHOME := os.Getenv("HOME")
	originalUSERPROFILE := os.Getenv("USERPROFILE")
	originalAPPDATA := os.Getenv("APPDATA")
	originalXDG_CONFIG_HOME := os.Getenv("XDG_CONFIG_HOME")

	defer func() {
		if originalHOME != "" {
			_ = os.Setenv("HOME", originalHOME)
		} else {
			_ = os.Unsetenv("HOME")
		}
		if originalUSERPROFILE != "" {
			_ = os.Setenv("USERPROFILE", originalUSERPROFILE)
		} else {
			_ = os.Unsetenv("USERPROFILE")
		}
		if originalAPPDATA != "" {
			_ = os.Setenv("APPDATA", originalAPPDATA)
		} else {
			_ = os.Unsetenv("APPDATA")
		}
		if originalXDG_CONFIG_HOME != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", originalXDG_CONFIG_HOME)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	// Set up test environment
	testConfigDir := filepath.Join(tmpDir, "grove")
	switch runtime.GOOS {
	case osWindows:
		require.NoError(t, os.Setenv("APPDATA", tmpDir))
		require.NoError(t, os.Unsetenv("HOME"))
		require.NoError(t, os.Unsetenv("USERPROFILE"))
		require.NoError(t, os.Unsetenv("XDG_CONFIG_HOME"))
	case osDarwin:
		require.NoError(t, os.Setenv("HOME", tmpDir))
		require.NoError(t, os.Unsetenv("APPDATA"))
		require.NoError(t, os.Unsetenv("USERPROFILE"))
		require.NoError(t, os.Unsetenv("XDG_CONFIG_HOME"))
		testConfigDir = filepath.Join(tmpDir, "Library", "Application Support", "grove")
	default:
		require.NoError(t, os.Setenv("XDG_CONFIG_HOME", tmpDir))
		require.NoError(t, os.Unsetenv("HOME"))
		require.NoError(t, os.Unsetenv("APPDATA"))
		require.NoError(t, os.Unsetenv("USERPROFILE"))
	}

	// Test creating config directory
	err = EnsureConfigDir()
	require.NoError(t, err)

	// Verify directory was created
	assert.DirExists(t, testConfigDir)

	// Test that it doesn't fail if directory already exists
	err = EnsureConfigDir()
	require.NoError(t, err)
}

func TestGetConfigFilePath(t *testing.T) {
	// Save original environment
	originalHOME := os.Getenv("HOME")
	originalUSERPROFILE := os.Getenv("USERPROFILE")
	originalAPPDATA := os.Getenv("APPDATA")
	originalXDG_CONFIG_HOME := os.Getenv("XDG_CONFIG_HOME")

	defer func() {
		if originalHOME != "" {
			_ = os.Setenv("HOME", originalHOME)
		} else {
			_ = os.Unsetenv("HOME")
		}
		if originalUSERPROFILE != "" {
			_ = os.Setenv("USERPROFILE", originalUSERPROFILE)
		} else {
			_ = os.Unsetenv("USERPROFILE")
		}
		if originalAPPDATA != "" {
			_ = os.Setenv("APPDATA", originalAPPDATA)
		} else {
			_ = os.Unsetenv("APPDATA")
		}
		if originalXDG_CONFIG_HOME != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", originalXDG_CONFIG_HOME)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	// Test with specific filename
	if runtime.GOOS == "linux" {
		require.NoError(t, os.Setenv("XDG_CONFIG_HOME", "/home/test/.config"))
		require.NoError(t, os.Unsetenv("HOME"))
		require.NoError(t, os.Unsetenv("APPDATA"))
		require.NoError(t, os.Unsetenv("USERPROFILE"))

		configPath := GetConfigFilePath("custom.toml")
		assert.Equal(t, "/home/test/.config/grove/custom.toml", configPath)
	}

	// Test with empty filename (should default to config.toml)
	configPath := GetConfigFilePath("")
	if runtime.GOOS == "linux" {
		assert.Equal(t, "/home/test/.config/grove/config.toml", configPath)
	}

	// Test with no config directory
	require.NoError(t, os.Unsetenv("HOME"))
	require.NoError(t, os.Unsetenv("USERPROFILE"))
	require.NoError(t, os.Unsetenv("APPDATA"))
	require.NoError(t, os.Unsetenv("XDG_CONFIG_HOME"))

	configPath = GetConfigFilePath("test.toml")
	assert.Equal(t, "test.toml", configPath)
}

func TestConfigExists(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "grove-config-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Save original environment
	originalGROVE_CONFIG := os.Getenv("GROVE_CONFIG")

	defer func() {
		if originalGROVE_CONFIG != "" {
			_ = os.Setenv("GROVE_CONFIG", originalGROVE_CONFIG)
		} else {
			_ = os.Unsetenv("GROVE_CONFIG")
		}
	}()

	// Set up test environment
	require.NoError(t, os.Setenv("GROVE_CONFIG", filepath.Join(tmpDir, "config.toml")))

	// Test when config doesn't exist
	exists, path := ConfigExists()
	assert.False(t, exists)
	assert.Empty(t, path)

	// Create a config file
	configPath := filepath.Join(tmpDir, "config.toml")
	err = os.WriteFile(configPath, []byte("test"), 0o644)
	require.NoError(t, err)

	// Test when config exists
	exists, path = ConfigExists()
	assert.True(t, exists)
	assert.Equal(t, configPath, path)

	// Test with JSON config
	_ = os.Remove(configPath)
	jsonConfigPath := filepath.Join(tmpDir, "config.json")
	err = os.WriteFile(jsonConfigPath, []byte("{}"), 0o644)
	require.NoError(t, err)

	exists, path = ConfigExists()
	assert.True(t, exists)
	assert.Equal(t, jsonConfigPath, path)
}

func TestListConfigPaths(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "grove-config-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Save original environment
	originalGROVE_CONFIG := os.Getenv("GROVE_CONFIG")

	defer func() {
		if originalGROVE_CONFIG != "" {
			_ = os.Setenv("GROVE_CONFIG", originalGROVE_CONFIG)
		} else {
			_ = os.Unsetenv("GROVE_CONFIG")
		}
	}()

	// Set up test environment
	require.NoError(t, os.Setenv("GROVE_CONFIG", filepath.Join(tmpDir, "config.toml")))

	// Create a config file
	configPath := filepath.Join(tmpDir, "config.toml")
	err = os.WriteFile(configPath, []byte("test"), 0o644)
	require.NoError(t, err)

	// Test listing config paths
	pathInfos := ListConfigPaths()
	assert.NotEmpty(t, pathInfos)

	// First path should be from GROVE_CONFIG and should exist
	assert.Equal(t, tmpDir, pathInfos[0].Path)
	assert.Equal(t, 1, pathInfos[0].Priority)
	assert.True(t, pathInfos[0].Exists)
	assert.Equal(t, configPath, pathInfos[0].File)

	// Other paths should have higher priority numbers
	for i := 1; i < len(pathInfos); i++ {
		assert.Greater(t, pathInfos[i].Priority, pathInfos[i-1].Priority)
	}
}

func TestCheckConfigExistsInPath(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "grove-config-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Test when no config exists
	exists, path := checkConfigExistsInPath(tmpDir)
	assert.False(t, exists)
	assert.Empty(t, path)

	// Create a TOML config file
	configPath := filepath.Join(tmpDir, "config.toml")
	err = os.WriteFile(configPath, []byte("test"), 0o644)
	require.NoError(t, err)

	// Test when config exists
	exists, path = checkConfigExistsInPath(tmpDir)
	assert.True(t, exists)
	assert.Equal(t, configPath, path)

	// Test priority order (TOML should be found first)
	yamlPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(yamlPath, []byte("test"), 0o644)
	require.NoError(t, err)

	exists, path = checkConfigExistsInPath(tmpDir)
	assert.True(t, exists)
	assert.Equal(t, configPath, path) // Should still return TOML path
}
