package create

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/sqve/grove/internal/git"
)

var (
	primaryColor = lipgloss.Color("#8B5CF6") // Purple - for highlights.
	successColor = lipgloss.Color("#059669") // Green - for success messages.
	mutedColor   = lipgloss.Color("#9CA3AF") // Gray - for progress messages.
)

var (
	successStyle = lipgloss.NewStyle().Foreground(successColor).Bold(true)
	primaryStyle = lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
	mutedStyle   = lipgloss.NewStyle().Foreground(mutedColor)
	boldStyle    = lipgloss.NewStyle().Bold(true)
)

type progressIndicator struct {
	message string
	start   time.Time
}

func (p *progressIndicator) show() {
	fmt.Fprintf(os.Stderr, "%s %s...\n", mutedStyle.Render("→"), p.message)
}

func (p *progressIndicator) complete() {
	elapsed := time.Since(p.start)
	// Move cursor up one line and clear it, then write success message.
	fmt.Fprintf(os.Stderr, "\033[1A\033[2K")
	if elapsed > time.Second {
		fmt.Fprintf(os.Stderr, "%s %s (%s)\n", successStyle.Render("✓"), p.message, mutedStyle.Render(elapsed.Round(time.Millisecond).String()))
	} else {
		fmt.Fprintf(os.Stderr, "%s %s\n", successStyle.Render("✓"), p.message)
	}
}

func (p *progressIndicator) fail() {
	elapsed := time.Since(p.start)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#DC2626")).Bold(true) // Red.
	// Move cursor up one line and clear it, then write failure message.
	fmt.Fprintf(os.Stderr, "\033[1A\033[2K")
	if elapsed > time.Second {
		fmt.Fprintf(os.Stderr, "%s %s (%s)\n", errorStyle.Render("✗"), p.message, mutedStyle.Render(elapsed.Round(time.Millisecond).String()))
	} else {
		fmt.Fprintf(os.Stderr, "%s %s\n", errorStyle.Render("✗"), p.message)
	}
}

func startProgress(message string) *progressIndicator {
	p := &progressIndicator{message: message, start: time.Now()}
	p.show()
	return p
}

func displaySuccess(result *CreateResult) {
	fmt.Println()

	fmt.Printf("%s Worktree created successfully!\n", successStyle.Render("✅"))
	fmt.Println()

	if result.WasCreated {
		fmt.Printf("  %s %s %s\n",
			boldStyle.Render("Branch:"),
			primaryStyle.Render(result.BranchName),
			mutedStyle.Render("(created)"))
	} else {
		fmt.Printf("  %s %s\n",
			boldStyle.Render("Branch:"),
			primaryStyle.Render(result.BranchName))
	}

	fmt.Printf("  %s %s\n",
		boldStyle.Render("Path:"),
		result.WorktreePath)

	if result.BaseBranch != "" {
		fmt.Printf("  %s %s\n",
			boldStyle.Render("Base:"),
			result.BaseBranch)
	}

	if result.CopiedFiles > 0 {
		fmt.Printf("  %s %d files copied\n",
			boldStyle.Render("Files:"),
			result.CopiedFiles)
	}

	fmt.Println()

	fmt.Printf("%s Next steps:\n", boldStyle.Render("→"))
	fmt.Printf("  %s\n", fmt.Sprintf("cd %s", mutedStyle.Render(result.WorktreePath)))
}

func NewCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [branch-name|url] [path]",
		Short: "Create a new Git worktree from a branch or URL",
		Long: `Create a new Git worktree from an existing or new branch with intelligent automation.

Basic usage:
  grove create feature-branch              # Create worktree for existing branch
  grove create feature-branch ./custom     # Create at specific path
  grove create new-feature                 # Creates new branch automatically if it doesn't exist
  grove create --base main new-feature     # Create new branch from specific base branch

URL and remote branch support:
  grove create https://github.com/owner/repo/pull/123
  grove create https://gitlab.com/owner/repo/-/merge_requests/456
  grove create https://bitbucket.org/owner/repo/pull-requests/789
  grove create https://github.com/owner/repo/tree/feature-branch
  grove create origin/feature-branch
  grove create upstream/hotfix-123

Supported platforms: GitHub, GitLab, Bitbucket, Azure DevOps, Codeberg, Gitea

File copying options:
  grove create feature-branch --copy-env               # Copy environment files (.env*, *.local.*)
  grove create feature-branch --copy ".env*,.vscode/"  # Copy specific patterns
  grove create feature-branch --no-copy                # Skip all file copying

File copying patterns support glob syntax. Common patterns:
  .env*             # All environment files
  .vscode/          # VS Code settings
  .idea/            # IntelliJ IDEA settings
  *.local.*         # Local configuration files
  .gitignore.local  # Local gitignore

By default, files are copied from the base branch's worktree when --base is specified,
or from the default branch worktree otherwise. Use --no-copy to disable, --copy-env for quick 
environment setup, or --copy with custom patterns for specific files.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := ValidateCreateArgs(args); err != nil {
				return err
			}

			noCopy, _ := cmd.Flags().GetBool("no-copy")
			copyEnv, _ := cmd.Flags().GetBool("copy-env")
			copyPatterns, _ := cmd.Flags().GetString("copy")

			return ValidateFlags(noCopy, copyEnv, copyPatterns)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			options, err := ParseCreateOptions(cmd, args)
			if err != nil {
				return err
			}

			var currentProgress *progressIndicator

			options.ProgressCallback = func(message string) {
				if currentProgress != nil {
					currentProgress.complete()
				}
				currentProgress = startProgress(message)
			}

			// Create service with dependencies.
			service := NewCreateService(
				NewBranchResolver(git.DefaultExecutor),
				NewPathGenerator(),
				NewWorktreeCreator(git.DefaultExecutor),
				NewFileManager(git.DefaultExecutor),
			)

			// Execute the create operation.
			result, err := service.Create(&options)
			if err != nil {
				// Mark current progress as failed.
				if currentProgress != nil {
					currentProgress.fail()
				}
				// Show error immediately after the failed step.
				fmt.Fprintf(os.Stderr, "\n%s %s\n",
					lipgloss.NewStyle().Foreground(lipgloss.Color("#DC2626")).Bold(true).Render("Error:"),
					err.Error())

				// Add space after error (tip is already included in error message for some errors).
				fmt.Fprintf(os.Stderr, "\n")

				// Don't show usage for operational errors (command was correct).
				cmd.SilenceUsage = true
				return err
			}

			// Mark final progress as completed.
			if currentProgress != nil {
				currentProgress.complete()
			}

			// Display success information.
			displaySuccess(result)

			return nil
		},
	}

	cmd.Flags().String("base", "", "Base branch for new branch creation (default: current branch)")
	cmd.Flags().Bool("copy-env", false, "Copy common environment files (.env*, *.local.*, docker-compose.override.yml)")
	cmd.Flags().String("copy", "", "Comma-separated glob patterns to copy (supports .env*,.vscode/,.idea/ etc.)")
	cmd.Flags().Bool("no-copy", false, "Skip all file copying (overrides config and other copy flags)")

	RegisterCreateCompletion(cmd)

	return cmd
}
