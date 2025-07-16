package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetConfigPaths returns a list of paths where Grove looks for config files
// The order is important - first paths have higher precedence
func GetConfigPaths() []string {
	var paths []string

	// 1. Environment variable override (highest precedence)
	if envPath := os.Getenv("GROVE_CONFIG"); envPath != "" {
		paths = append(paths, filepath.Dir(envPath))
	}

	// 2. Current directory (project-specific config)
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, cwd)
	}

	// 3. User config directory (platform-specific)
	if userConfigDir := getUserConfigDir(); userConfigDir != "" {
		paths = append(paths, userConfigDir)
	}

	// 4. Home directory (fallback)
	if homeDir := getHomeDir(); homeDir != "" {
		paths = append(paths, homeDir)
	}

	return paths
}

// getUserConfigDir returns the user's config directory based on platform
func getUserConfigDir() string {
	switch runtime.GOOS {
	case "windows":
		return getWindowsConfigDir()
	case "darwin":
		return getMacOSConfigDir()
	default:
		return getLinuxConfigDir()
	}
}

// getWindowsConfigDir returns the Windows config directory
func getWindowsConfigDir() string {
	if appData := os.Getenv("APPDATA"); appData != "" {
		return filepath.Join(appData, "grove")
	}
	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
		return filepath.Join(userProfile, "AppData", "Roaming", "grove")
	}
	return ""
}

// getMacOSConfigDir returns the macOS config directory
func getMacOSConfigDir() string {
	if homeDir := getHomeDir(); homeDir != "" {
		return filepath.Join(homeDir, "Library", "Application Support", "grove")
	}
	return ""
}

// getLinuxConfigDir returns the Linux config directory following XDG Base Directory specification
func getLinuxConfigDir() string {
	// Follow XDG Base Directory specification
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "grove")
	}
	if homeDir := getHomeDir(); homeDir != "" {
		return filepath.Join(homeDir, ".config", "grove")
	}
	return ""
}

// getHomeDir returns the user's home directory
func getHomeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
		return userProfile
	}
	return ""
}

// GetDefaultConfigPath returns the default config file path for the current platform
func GetDefaultConfigPath() string {
	configDir := getUserConfigDir()
	if configDir == "" {
		return ""
	}
	return filepath.Join(configDir, "config.toml")
}

// EnsureConfigDir creates the config directory if it doesn't exist
func EnsureConfigDir() error {
	configDir := getUserConfigDir()
	if configDir == "" {
		return nil // No config directory available
	}

	return os.MkdirAll(configDir, 0755)
}

// GetConfigFilePath returns the path to a specific config file
func GetConfigFilePath(filename string) string {
	if filename == "" {
		filename = "config.toml"
	}

	configDir := getUserConfigDir()
	if configDir == "" {
		return filename
	}

	return filepath.Join(configDir, filename)
}

// ConfigExists checks if a config file exists in any of the search paths
func ConfigExists() (bool, string) {
	paths := GetConfigPaths()
	filenames := []string{"config.toml", "config.yaml", "config.yml", "config.json"}

	for _, path := range paths {
		for _, filename := range filenames {
			configPath := filepath.Join(path, filename)
			if _, err := os.Stat(configPath); err == nil {
				return true, configPath
			}
		}
	}

	return false, ""
}

// ListConfigPaths returns all possible config paths with their priority
func ListConfigPaths() []ConfigPathInfo {
	paths := GetConfigPaths()
	result := make([]ConfigPathInfo, len(paths))

	for i, path := range paths {
		exists, configFile := checkConfigExistsInPath(path)
		result[i] = ConfigPathInfo{
			Path:     path,
			Priority: i + 1,
			Exists:   exists,
			File:     configFile,
		}
	}

	return result
}

// ConfigPathInfo contains information about a config path
type ConfigPathInfo struct {
	Path     string
	Priority int
	Exists   bool
	File     string
}

// checkConfigExistsInPath checks if a config file exists in a specific path
func checkConfigExistsInPath(path string) (bool, string) {
	filenames := []string{"config.toml", "config.yaml", "config.yml", "config.json"}

	for _, filename := range filenames {
		configPath := filepath.Join(path, filename)
		if _, err := os.Stat(configPath); err == nil {
			return true, configPath
		}
	}

	return false, ""
}
