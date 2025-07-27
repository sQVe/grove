package config

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	osWindows = "windows"
	osDarwin  = "darwin"
)

// The order is important - first paths have higher precedence.
func GetConfigPaths() []string {
	var paths []string

	// 1. Environment variable override (highest precedence).
	if envPath := os.Getenv("GROVE_CONFIG"); envPath != "" {
		paths = append(paths, filepath.Dir(envPath))
	}

	// 2. Current directory (project-specific config).
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, cwd)
	}

	// 3. User config directory (platform-specific).
	if userConfigDir := getUserConfigDir(); userConfigDir != "" {
		paths = append(paths, userConfigDir)
	}

	// 4. Home directory (fallback).
	if homeDir := getHomeDir(); homeDir != "" {
		paths = append(paths, homeDir)
	}

	return paths
}

func getUserConfigDir() string {
	switch runtime.GOOS {
	case osWindows:
		return getWindowsConfigDir()
	case osDarwin:
		return getMacOSConfigDir()
	default:
		return getLinuxConfigDir()
	}
}

func getWindowsConfigDir() string {
	if appData := os.Getenv("APPDATA"); appData != "" {
		return filepath.Join(appData, "grove")
	}
	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
		return filepath.Join(userProfile, "AppData", "Roaming", "grove")
	}
	return ""
}

func getMacOSConfigDir() string {
	if homeDir := getHomeDir(); homeDir != "" {
		return filepath.Join(homeDir, "Library", "Application Support", "grove")
	}
	return ""
}

func getLinuxConfigDir() string {
	// Follow XDG Base Directory specification.
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "grove")
	}
	if homeDir := getHomeDir(); homeDir != "" {
		return filepath.Join(homeDir, ".config", "grove")
	}
	return ""
}

func getHomeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
		return userProfile
	}
	return ""
}

func GetDefaultConfigPath() string {
	configDir := getUserConfigDir()
	if configDir == "" {
		return ""
	}
	return filepath.Join(configDir, "config.toml")
}

func EnsureConfigDir() error {
	configDir := getUserConfigDir()
	if configDir == "" {
		return nil // No config directory available
	}

	return os.MkdirAll(configDir, 0o755)
}

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

func ConfigExists() (exists bool, configPath string) {
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

type ConfigPathInfo struct {
	Path     string
	Priority int
	Exists   bool
	File     string
}

func checkConfigExistsInPath(path string) (exists bool, configPath string) {
	filenames := []string{"config.toml", "config.yaml", "config.yml", "config.json"}

	for _, filename := range filenames {
		configPath := filepath.Join(path, filename)
		if _, err := os.Stat(configPath); err == nil {
			return true, configPath
		}
	}

	return false, ""
}
