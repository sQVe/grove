package switchcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sqve/grove/internal/errors"
)

// ShellType represents the type of shell
type ShellType string

const (
	ShellBash       ShellType = "bash"
	ShellZsh        ShellType = "zsh"
	ShellFish       ShellType = "fish"
	ShellPowerShell ShellType = "powershell"
	ShellUnknown    ShellType = "unknown"

	// OS identifiers
	osWindows = "windows"
)

// ShellInfo contains information about the detected shell
type ShellInfo struct {
	Type        ShellType
	ConfigPaths []string
	Detected    bool
}

// DetectShell detects the current shell environment
func DetectShell() ShellInfo {
	// Check SHELL environment variable first
	shellPath := os.Getenv("SHELL")
	if shellPath != "" {
		shellName := filepath.Base(shellPath)
		switch {
		case strings.Contains(shellName, "bash"):
			return ShellInfo{
				Type:        ShellBash,
				ConfigPaths: getBashConfigPaths(),
				Detected:    true,
			}
		case strings.Contains(shellName, "zsh"):
			return ShellInfo{
				Type:        ShellZsh,
				ConfigPaths: getZshConfigPaths(),
				Detected:    true,
			}
		case strings.Contains(shellName, "fish"):
			return ShellInfo{
				Type:        ShellFish,
				ConfigPaths: getFishConfigPaths(),
				Detected:    true,
			}
		}
	}

	// Check for PowerShell on Windows
	if runtime.GOOS == osWindows {
		if os.Getenv("PSModulePath") != "" {
			return ShellInfo{
				Type:        ShellPowerShell,
				ConfigPaths: getPowerShellConfigPaths(),
				Detected:    true,
			}
		}
	}

	// Fallback detection by checking PS1 or other indicators
	ps1 := os.Getenv("PS1")
	if strings.Contains(ps1, "zsh") {
		return ShellInfo{
			Type:        ShellZsh,
			ConfigPaths: getZshConfigPaths(),
			Detected:    true,
		}
	}

	return ShellInfo{
		Type:     ShellUnknown,
		Detected: false,
	}
}

// GenerateShellIntegration generates shell-specific integration code
func GenerateShellIntegration(shellType ShellType) (string, error) {
	switch shellType {
	case ShellBash, ShellZsh:
		return generateBashZshIntegration(), nil
	case ShellFish:
		return generateFishIntegration(), nil
	case ShellPowerShell:
		return generatePowerShellIntegration(), nil
	default:
		return "", fmt.Errorf("unsupported shell type: %s", shellType)
	}
}

// generateBashZshIntegration generates shell integration for bash and zsh
func generateBashZshIntegration() string {
	return `# Grove shell integration
grove() {
    if [[ "$1" == "switch" ]]; then
        local target
        # Safely escape the worktree name to prevent shell injection
        local escaped_name
        printf -v escaped_name '%q' "$2"
        target=$(command grove switch --get-path "$escaped_name" 2>/dev/null)
        if [[ $? -eq 0 ]] && [[ -n "$target" ]]; then
            cd "$target" || return 1
        else
            # Show error message by running the command normally
            command grove switch "$@"
        fi
    else
        command grove "$@"
    fi
}`
}

// generateFishIntegration generates shell integration for fish
func generateFishIntegration() string {
	return `# Grove shell integration
function grove
    if test "$argv[1]" = "switch"
        # Safely quote the worktree name to prevent shell injection
        set -l escaped_name (string escape -- $argv[2])
        set target (command grove switch --get-path $escaped_name 2>/dev/null)
        if test $status -eq 0 -a -n "$target"
            cd "$target"
        else
            # Show error message by running the command normally
            command grove switch $argv[2..-1]
        end
    else
        command grove $argv
    end
end`
}

// generatePowerShellIntegration generates shell integration for PowerShell
func generatePowerShellIntegration() string {
	return `# Grove shell integration
function grove {
    if ($args[0] -eq "switch") {
        # Safely escape the worktree name to prevent command injection
        $escapedName = $args[1] -replace "'", "''"
        $target = & grove switch --get-path "'$escapedName'" 2>$null
        if ($LASTEXITCODE -eq 0 -and $target) {
            Set-Location $target
        } else {
            # Show error message by running the command normally
            & grove switch $args[1..$args.Length]
        }
    } else {
        & grove @args
    }
}`
}

// Shell configuration file path helpers
func getBashConfigPaths() []string {
	homeDir, _ := os.UserHomeDir()
	return []string{
		filepath.Join(homeDir, ".bashrc"),
		filepath.Join(homeDir, ".bash_profile"),
		filepath.Join(homeDir, ".profile"),
	}
}

func getZshConfigPaths() []string {
	homeDir, _ := os.UserHomeDir()
	return []string{
		filepath.Join(homeDir, ".zshrc"),
		filepath.Join(homeDir, ".zprofile"),
	}
}

func getFishConfigPaths() []string {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "fish")
	return []string{
		filepath.Join(configDir, "config.fish"),
	}
}

func getPowerShellConfigPaths() []string {
	if runtime.GOOS == osWindows {
		homeDir, _ := os.UserHomeDir()
		return []string{
			filepath.Join(homeDir, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
			filepath.Join(homeDir, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
		}
	}

	// PowerShell on Unix systems
	homeDir, _ := os.UserHomeDir()
	return []string{
		filepath.Join(homeDir, ".config", "powershell", "Microsoft.PowerShell_profile.ps1"),
	}
}

// validateConfigPath ensures the config path is safe and within expected directories
func validateConfigPath(configPath string) error {
	// Resolve any symlinks to get the actual path
	resolvedPath, err := filepath.EvalSymlinks(configPath)
	if err != nil && !os.IsNotExist(err) {
		return errors.NewGroveError(
			errors.ErrCodeFileSystem,
			"failed to resolve config path",
			err,
		).WithContext("path", configPath)
	}

	// Use resolved path if available, otherwise use original
	pathToCheck := configPath
	if err == nil {
		pathToCheck = resolvedPath
	}

	// Clean and make absolute
	absPath, err := filepath.Abs(filepath.Clean(pathToCheck))
	if err != nil {
		return errors.NewGroveError(
			errors.ErrCodeFileSystem,
			"failed to get absolute path",
			err,
		).WithContext("path", pathToCheck)
	}

	// Get user home directory for validation
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return errors.NewGroveError(
			errors.ErrCodeFileSystem,
			"failed to get user home directory",
			err,
		)
	}

	// Ensure path is within user's home directory or system config directories
	homeAbsPath, err := filepath.Abs(homeDir)
	if err != nil {
		return errors.NewGroveError(
			errors.ErrCodeFileSystem,
			"failed to get absolute home path",
			err,
		).WithContext("home", homeDir)
	}

	// Check if path is within home directory
	if !strings.HasPrefix(absPath, homeAbsPath+string(filepath.Separator)) && absPath != homeAbsPath {
		// Allow some system config directories on Unix
		allowedPrefixes := []string{"/etc/", "/usr/local/etc/"}
		if runtime.GOOS != osWindows {
			for _, prefix := range allowedPrefixes {
				if strings.HasPrefix(absPath, prefix) {
					return nil // Allow system config directories
				}
			}
		}

		return errors.NewGroveError(
			errors.ErrCodeFileSystem,
			"config path is outside user home directory and not in allowed system directories",
			nil,
		).WithContext("path", absPath).WithContext("home", homeAbsPath)
	}

	return nil
}

// GetShellConfigPath returns the primary configuration file path for a shell
func GetShellConfigPath(shellType ShellType) (string, error) {
	var paths []string

	switch shellType {
	case ShellBash:
		paths = getBashConfigPaths()
	case ShellZsh:
		paths = getZshConfigPaths()
	case ShellFish:
		paths = getFishConfigPaths()
	case ShellPowerShell:
		paths = getPowerShellConfigPaths()
	default:
		return "", errors.NewGroveError(
			errors.ErrCodeConfigInvalid,
			fmt.Sprintf("unsupported shell type: %s", shellType),
			nil,
		).WithContext("shell_type", string(shellType))
	}

	// Return the first existing config file, or the primary one if none exist
	for _, path := range paths {
		if err := validateConfigPath(path); err != nil {
			continue // Skip invalid paths
		}
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Return primary config path even if it doesn't exist, but validate it first
	if len(paths) > 0 {
		primaryPath := paths[0]
		if err := validateConfigPath(primaryPath); err != nil {
			return "", err
		}
		return primaryPath, nil
	}

	return "", errors.NewGroveError(
		errors.ErrCodeConfigInvalid,
		fmt.Sprintf("no configuration paths available for shell: %s", shellType),
		nil,
	).WithContext("shell_type", string(shellType))
}

// IsShellIntegrationInstalled checks if grove shell integration is already installed
func IsShellIntegrationInstalled(shellType ShellType) (bool, error) {
	configPath, err := GetShellConfigPath(shellType)
	if err != nil {
		return false, err
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	// Check for grove shell integration comment marker
	return strings.Contains(string(content), "# Grove shell integration"), nil
}
