package switchcmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/errors"
)

var (
	// Styling for output messages
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6")) // Blue
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#059669")) // Green
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#D97706")) // Yellow
	boldStyle    = lipgloss.NewStyle().Bold(true)
)

// InstallShellIntegrationOptions contains options for shell integration installation
type InstallShellIntegrationOptions struct {
	Shell  string
	Manual bool
	Force  bool
	DryRun bool
}

// NewInstallShellIntegrationCmd creates the install-shell-integration command
func NewInstallShellIntegrationCmd() *cobra.Command {
	options := &InstallShellIntegrationOptions{}

	cmd := &cobra.Command{
		Use:   "install-shell-integration",
		Short: "Install shell integration for grove switch command",
		Long: `Install shell integration that allows 'grove switch' to change your current directory.

This command will:
1. Detect your current shell (bash, zsh, fish, or PowerShell)
2. Generate appropriate shell functions
3. Either modify your shell configuration file or provide manual instructions

Examples:
  grove switch install-shell-integration                # Auto-detect and install
  grove switch install-shell-integration --manual       # Show manual instructions
  grove switch install-shell-integration --shell=zsh    # Install for specific shell
  grove switch install-shell-integration --dry-run      # Preview changes without installing`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstallShellIntegration(options)
		},
	}

	cmd.Flags().StringVar(&options.Shell, "shell", "", "Target shell (bash, zsh, fish, powershell)")
	cmd.Flags().BoolVar(&options.Manual, "manual", false, "Show manual installation instructions instead of automatic installation")
	cmd.Flags().BoolVar(&options.Force, "force", false, "Force installation even if integration already exists")
	cmd.Flags().BoolVar(&options.DryRun, "dry-run", false, "Preview changes without actually installing")

	return cmd
}

// runInstallShellIntegration executes the shell integration installation
func runInstallShellIntegration(options *InstallShellIntegrationOptions) error {
	var shellInfo ShellInfo
	var err error

	// Detect or validate shell
	if options.Shell != "" {
		shellType := ShellType(options.Shell)
		switch shellType {
		case ShellBash, ShellZsh, ShellFish, ShellPowerShell:
			shellInfo = ShellInfo{
				Type:     shellType,
				Detected: true,
			}
			// Set config paths based on shell type
			switch shellType {
			case ShellBash:
				shellInfo.ConfigPaths = getBashConfigPaths()
			case ShellZsh:
				shellInfo.ConfigPaths = getZshConfigPaths()
			case ShellFish:
				shellInfo.ConfigPaths = getFishConfigPaths()
			case ShellPowerShell:
				shellInfo.ConfigPaths = getPowerShellConfigPaths()
			}
		default:
			return errors.NewGroveError(
				errors.ErrCodeConfigInvalid,
				fmt.Sprintf("unsupported shell: %s. Supported shells: bash, zsh, fish, powershell", options.Shell),
				nil,
			).WithContext("shell", options.Shell)
		}
	} else {
		shellInfo = DetectShell()
		if !shellInfo.Detected {
			return errors.NewGroveError(
				errors.ErrCodeConfigInvalid,
				"could not detect shell. Please specify one with --shell flag",
				nil,
			).WithContext("shell_env", os.Getenv("SHELL"))
		}
	}

	fmt.Printf("%s Detected shell: %s\n", infoStyle.Render("ℹ"), boldStyle.Render(string(shellInfo.Type)))

	// Check if already installed
	if !options.Force {
		installed, err := IsShellIntegrationInstalled(shellInfo.Type)
		if err == nil && installed {
			fmt.Printf("%s Grove shell integration is already installed for %s\n",
				warningStyle.Render("⚠"), shellInfo.Type)
			fmt.Printf("  Use %s to reinstall\n", boldStyle.Render("--force"))
			return nil
		}
	}

	// Generate shell integration code
	integrationCode, err := GenerateShellIntegration(shellInfo.Type)
	if err != nil {
		return errors.NewGroveError(
			errors.ErrCodeConfigInvalid,
			"failed to generate shell integration",
			err,
		).WithContext("shell_type", string(shellInfo.Type))
	}

	// Handle manual installation
	if options.Manual {
		return showManualInstructions(shellInfo, integrationCode)
	}

	// Handle dry run
	if options.DryRun {
		return showDryRun(shellInfo, integrationCode)
	}

	// Automatic installation
	return performAutomaticInstallation(shellInfo, integrationCode, options.Force)
}

// showManualInstructions displays manual installation instructions
func showManualInstructions(shellInfo ShellInfo, integrationCode string) error {
	fmt.Printf("\n%s Manual Installation Instructions\n", boldStyle.Render("📋"))
	fmt.Printf("Add the following code to your shell configuration file:\n\n")

	configPath, _ := GetShellConfigPath(shellInfo.Type)
	fmt.Printf("%s Configuration file: %s\n\n", infoStyle.Render("📁"), configPath)

	// Display the integration code
	fmt.Println(lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(1).Render(integrationCode))

	fmt.Printf("\n%s After adding the code, restart your shell or run:\n", boldStyle.Render("🔄"))

	switch shellInfo.Type {
	case ShellBash:
		fmt.Printf("  source ~/.bashrc\n")
	case ShellZsh:
		fmt.Printf("  source ~/.zshrc\n")
	case ShellFish:
		fmt.Printf("  source ~/.config/fish/config.fish\n")
	case ShellPowerShell:
		fmt.Printf("  . $PROFILE\n")
	}

	return nil
}

// showDryRun shows what would be done without actually doing it
func showDryRun(shellInfo ShellInfo, integrationCode string) error {
	fmt.Printf("\n%s Dry Run - No changes will be made\n", boldStyle.Render("🔍"))

	configPath, err := GetShellConfigPath(shellInfo.Type)
	if err != nil {
		return err
	}

	fmt.Printf("%s Target file: %s\n", infoStyle.Render("📁"), configPath)

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("%s File does not exist - would be created\n", warningStyle.Render("⚠"))
	} else {
		fmt.Printf("%s File exists - would create backup: %s.grove-backup-%d\n",
			infoStyle.Render("ℹ"), configPath, time.Now().Unix())
	}

	fmt.Printf("\n%s Code to be added:\n", boldStyle.Render("📝"))
	fmt.Println(lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(1).Render(integrationCode))

	return nil
}

// performAutomaticInstallation performs the automatic installation
func performAutomaticInstallation(shellInfo ShellInfo, integrationCode string, force bool) error {
	configPath, err := GetShellConfigPath(shellInfo.Type)
	if err != nil {
		return err
	}

	fmt.Printf("%s Installing shell integration for %s\n", infoStyle.Render("🔧"), shellInfo.Type)
	fmt.Printf("  Target: %s\n", configPath)

	// Check if config file exists and create backup
	if _, err := os.Stat(configPath); err == nil {
		backupPath := fmt.Sprintf("%s.grove-backup-%d", configPath, time.Now().Unix())
		fmt.Printf("  Creating backup: %s\n", backupPath)

		if err := copyFile(configPath, backupPath); err != nil {
			return errors.NewGroveError(
				errors.ErrCodeFileSystem,
				"failed to create backup",
				err,
			).WithContext("source", configPath).WithContext("backup", backupPath)
		}
	} else if !os.IsNotExist(err) {
		return errors.NewGroveError(
			errors.ErrCodeFileSystem,
			"failed to check config file",
			err,
		).WithContext("config_path", configPath)
	}

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return errors.NewGroveError(
			errors.ErrCodeFileSystem,
			"failed to create config directory",
			err,
		).WithContext("directory", filepath.Dir(configPath))
	}

	// Use atomic file operations to prevent corruption
	if err := atomicAppendToFile(configPath, integrationCode); err != nil {
		return errors.NewGroveError(
			errors.ErrCodeFileSystem,
			"failed to install integration code",
			err,
		).WithContext("config_path", configPath)
	}

	fmt.Printf("%s Shell integration installed successfully!\n", successStyle.Render("✅"))
	fmt.Printf("\n%s Next steps:\n", boldStyle.Render("🚀"))
	fmt.Printf("  1. Restart your shell or run: ")

	switch shellInfo.Type {
	case ShellBash:
		fmt.Printf("source ~/.bashrc\n")
	case ShellZsh:
		fmt.Printf("source ~/.zshrc\n")
	case ShellFish:
		fmt.Printf("source ~/.config/fish/config.fish\n")
	case ShellPowerShell:
		fmt.Printf(". $PROFILE\n")
	}

	fmt.Printf("  2. Test with: %s\n", boldStyle.Render("grove switch <worktree-name>"))

	return nil
}

// atomicAppendToFile atomically appends content to a file using temp file + rename
func atomicAppendToFile(configPath, integrationCode string) error {
	// Create temporary file in the same directory to ensure atomic rename works
	tempFile, err := os.CreateTemp(filepath.Dir(configPath), ".grove-install-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempPath := tempFile.Name()

	// Ensure cleanup on error
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	// Copy existing content if file exists
	if existingFile, err := os.Open(configPath); err == nil {
		defer func() { _ = existingFile.Close() }()
		if _, err := io.Copy(tempFile, existingFile); err != nil {
			return fmt.Errorf("failed to copy existing content: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read existing config: %w", err)
	}

	// Add newlines before and after for clean separation
	integrationWithNewlines := fmt.Sprintf("\n%s\n", integrationCode)

	// Append integration code
	if _, err := tempFile.WriteString(integrationWithNewlines); err != nil {
		return fmt.Errorf("failed to write integration code: %w", err)
	}

	// Ensure all data is written
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temporary file: %w", err)
	}

	// Close temp file before rename
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Atomically replace the config file
	if err := os.Rename(tempPath, configPath); err != nil {
		return fmt.Errorf("failed to replace config file: %w", err)
	}

	return nil
}

// copyFile copies a file from src to dst using efficient I/O
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = sourceFile.Close() }()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = destFile.Close() }()

	// Use io.Copy for efficient copying instead of line-by-line
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return destFile.Sync()
}

