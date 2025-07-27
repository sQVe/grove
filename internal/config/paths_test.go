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
	originalGROVE_CONFIG := os.Getenv("GROVE_CONFIG")
	originalHOME := os.Getenv("HOME")
	originalUSERPROFILE := os.Getenv("USERPROFILE")
	originalAPPDATA := os.Getenv("APPDATA")
	originalXDG_CONFIG_HOME := os.Getenv("XDG_CONFIG_HOME")

	defer func() {
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

	testConfigPath := "/tmp/test-config"
	require.NoError(t, os.Setenv("GROVE_CONFIG", filepath.Join(testConfigPath, "config.toml")))

	paths := GetConfigPaths()
	assert.NotEmpty(t, paths)
	assert.Equal(t, testConfigPath, paths[0])

	require.NoError(t, os.Unsetenv("GROVE_CONFIG"))
	paths = GetConfigPaths()
	assert.NotEmpty(t, paths)

	cwd, _ := os.Getwd()
	assert.Contains(t, paths, cwd)
}

func TestGetUserConfigDir(t *testing.T) {
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

	switch runtime.GOOS {
	case osWindows:
		require.NoError(t, os.Setenv("APPDATA", `C:\Users\test\AppData\Roaming`))
		require.NoError(t, os.Unsetenv("HOME"))
		require.NoError(t, os.Unsetenv("USERPROFILE"))
		require.NoError(t, os.Unsetenv("XDG_CONFIG_HOME"))

		configDir := getUserConfigDir()
		assert.Equal(t, `C:\Users\test\AppData\Roaming\grove`, configDir)

		require.NoError(t, os.Unsetenv("APPDATA"))
		require.NoError(t, os.Setenv("USERPROFILE", `C:\Users\test`))

		configDir = getUserConfigDir()
		assert.Equal(t, `C:\Users\test\AppData\Roaming\grove`, configDir)

	case osDarwin:
		require.NoError(t, os.Setenv("HOME", "/Users/test"))
		require.NoError(t, os.Unsetenv("APPDATA"))
		require.NoError(t, os.Unsetenv("USERPROFILE"))
		require.NoError(t, os.Unsetenv("XDG_CONFIG_HOME"))

		configDir := getUserConfigDir()
		assert.Equal(t, "/Users/test/Library/Application Support/grove", configDir)

	default:
		require.NoError(t, os.Setenv("XDG_CONFIG_HOME", "/home/test/.config"))
		require.NoError(t, os.Unsetenv("HOME"))
		require.NoError(t, os.Unsetenv("APPDATA"))
		require.NoError(t, os.Unsetenv("USERPROFILE"))

		configDir := getUserConfigDir()
		assert.Equal(t, "/home/test/.config/grove", configDir)

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

	require.NoError(t, os.Setenv("APPDATA", `C:\Users\test\AppData\Roaming`))
	require.NoError(t, os.Unsetenv("USERPROFILE"))

	configDir := getWindowsConfigDir()
	assert.Equal(t, `C:\Users\test\AppData\Roaming\grove`, configDir)

	require.NoError(t, os.Unsetenv("APPDATA"))
	require.NoError(t, os.Setenv("USERPROFILE", `C:\Users\test`))

	configDir = getWindowsConfigDir()
	assert.Equal(t, `C:\Users\test\AppData\Roaming\grove`, configDir)

	require.NoError(t, os.Unsetenv("APPDATA"))
	require.NoError(t, os.Unsetenv("USERPROFILE"))

	configDir = getWindowsConfigDir()
	assert.Equal(t, "", configDir)
}

func TestGetMacOSConfigDir(t *testing.T) {
	if runtime.GOOS != osDarwin {
		t.Skip("macOS-specific test")
	}

	originalHOME := os.Getenv("HOME")

	defer func() {
		if originalHOME != "" {
			_ = os.Setenv("HOME", originalHOME)
		} else {
			_ = os.Unsetenv("HOME")
		}
	}()

	require.NoError(t, os.Setenv("HOME", "/Users/test"))

	configDir := getMacOSConfigDir()
	assert.Equal(t, "/Users/test/Library/Application Support/grove", configDir)

	require.NoError(t, os.Unsetenv("HOME"))

	configDir = getMacOSConfigDir()
	assert.Equal(t, "", configDir)
}

func TestGetLinuxConfigDir(t *testing.T) {
	if runtime.GOOS == osWindows || runtime.GOOS == osDarwin {
		t.Skip("Linux-specific test")
	}

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

	require.NoError(t, os.Setenv("XDG_CONFIG_HOME", "/home/test/.config"))
	require.NoError(t, os.Unsetenv("HOME"))

	configDir := getLinuxConfigDir()
	assert.Equal(t, "/home/test/.config/grove", configDir)

	require.NoError(t, os.Unsetenv("XDG_CONFIG_HOME"))
	require.NoError(t, os.Setenv("HOME", "/home/test"))

	configDir = getLinuxConfigDir()
	assert.Equal(t, "/home/test/.config/grove", configDir)

	require.NoError(t, os.Unsetenv("XDG_CONFIG_HOME"))
	require.NoError(t, os.Unsetenv("HOME"))

	configDir = getLinuxConfigDir()
	assert.Equal(t, "", configDir)
}

func TestGetHomeDir(t *testing.T) {
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

	require.NoError(t, os.Setenv("HOME", "/home/test"))
	require.NoError(t, os.Unsetenv("USERPROFILE"))

	homeDir := getHomeDir()
	assert.Equal(t, "/home/test", homeDir)

	require.NoError(t, os.Unsetenv("HOME"))
	require.NoError(t, os.Setenv("USERPROFILE", "/Users/test"))

	homeDir = getHomeDir()
	assert.Equal(t, "/Users/test", homeDir)

	require.NoError(t, os.Unsetenv("HOME"))
	require.NoError(t, os.Unsetenv("USERPROFILE"))

	homeDir = getHomeDir()
	assert.Equal(t, "", homeDir)
}

func TestGetDefaultConfigPath(t *testing.T) {
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

	require.NoError(t, os.Unsetenv("HOME"))
	require.NoError(t, os.Unsetenv("USERPROFILE"))
	require.NoError(t, os.Unsetenv("APPDATA"))
	require.NoError(t, os.Unsetenv("XDG_CONFIG_HOME"))

	configPath := GetDefaultConfigPath()
	assert.Equal(t, "", configPath)
}

func TestEnsureConfigDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grove-config-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

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

	err = EnsureConfigDir()
	require.NoError(t, err)

	assert.DirExists(t, testConfigDir)

	err = EnsureConfigDir()
	require.NoError(t, err)
}

func TestGetConfigFilePath(t *testing.T) {
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

	if runtime.GOOS == "linux" {
		require.NoError(t, os.Setenv("XDG_CONFIG_HOME", "/home/test/.config"))
		require.NoError(t, os.Unsetenv("HOME"))
		require.NoError(t, os.Unsetenv("APPDATA"))
		require.NoError(t, os.Unsetenv("USERPROFILE"))

		configPath := GetConfigFilePath("custom.toml")
		assert.Equal(t, "/home/test/.config/grove/custom.toml", configPath)
	}

	configPath := GetConfigFilePath("")
	if runtime.GOOS == "linux" {
		assert.Equal(t, "/home/test/.config/grove/config.toml", configPath)
	}

	require.NoError(t, os.Unsetenv("HOME"))
	require.NoError(t, os.Unsetenv("USERPROFILE"))
	require.NoError(t, os.Unsetenv("APPDATA"))
	require.NoError(t, os.Unsetenv("XDG_CONFIG_HOME"))

	configPath = GetConfigFilePath("test.toml")
	assert.Equal(t, "test.toml", configPath)
}

func TestConfigExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grove-config-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	originalGROVE_CONFIG := os.Getenv("GROVE_CONFIG")

	defer func() {
		if originalGROVE_CONFIG != "" {
			_ = os.Setenv("GROVE_CONFIG", originalGROVE_CONFIG)
		} else {
			_ = os.Unsetenv("GROVE_CONFIG")
		}
	}()

	require.NoError(t, os.Setenv("GROVE_CONFIG", filepath.Join(tmpDir, "config.toml")))

	exists, path := ConfigExists()
	assert.False(t, exists)
	assert.Empty(t, path)

	configPath := filepath.Join(tmpDir, "config.toml")
	err = os.WriteFile(configPath, []byte("test"), 0o644)
	require.NoError(t, err)

	exists, path = ConfigExists()
	assert.True(t, exists)
	assert.Equal(t, configPath, path)

	_ = os.Remove(configPath)
	jsonConfigPath := filepath.Join(tmpDir, "config.json")
	err = os.WriteFile(jsonConfigPath, []byte("{}"), 0o644)
	require.NoError(t, err)

	exists, path = ConfigExists()
	assert.True(t, exists)
	assert.Equal(t, jsonConfigPath, path)
}

func TestListConfigPaths(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grove-config-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	originalGROVE_CONFIG := os.Getenv("GROVE_CONFIG")

	defer func() {
		if originalGROVE_CONFIG != "" {
			_ = os.Setenv("GROVE_CONFIG", originalGROVE_CONFIG)
		} else {
			_ = os.Unsetenv("GROVE_CONFIG")
		}
	}()

	require.NoError(t, os.Setenv("GROVE_CONFIG", filepath.Join(tmpDir, "config.toml")))

	configPath := filepath.Join(tmpDir, "config.toml")
	err = os.WriteFile(configPath, []byte("test"), 0o644)
	require.NoError(t, err)

	pathInfos := ListConfigPaths()
	assert.NotEmpty(t, pathInfos)

	assert.Equal(t, tmpDir, pathInfos[0].Path)
	assert.Equal(t, 1, pathInfos[0].Priority)
	assert.True(t, pathInfos[0].Exists)
	assert.Equal(t, configPath, pathInfos[0].File)

	for i := 1; i < len(pathInfos); i++ {
		assert.Greater(t, pathInfos[i].Priority, pathInfos[i-1].Priority)
	}
}

func TestCheckConfigExistsInPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grove-config-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	exists, path := checkConfigExistsInPath(tmpDir)
	assert.False(t, exists)
	assert.Empty(t, path)

	configPath := filepath.Join(tmpDir, "config.toml")
	err = os.WriteFile(configPath, []byte("test"), 0o644)
	require.NoError(t, err)

	exists, path = checkConfigExistsInPath(tmpDir)
	assert.True(t, exists)
	assert.Equal(t, configPath, path)

	yamlPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(yamlPath, []byte("test"), 0o644)
	require.NoError(t, err)

	exists, path = checkConfigExistsInPath(tmpDir)
	assert.True(t, exists)
	assert.Equal(t, configPath, path)
}
