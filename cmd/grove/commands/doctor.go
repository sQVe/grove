package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/workspace"
)

// Severity represents the severity level of a doctor issue
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityError
)

// Category represents the category of a doctor issue
type Category int

const (
	CategoryGit Category = iota
	CategoryConfig
)

// Issue represents a single diagnostic issue found by doctor
type Issue struct {
	Category    Category
	Severity    Severity
	Message     string
	Path        string
	Details     []string
	FixHint     string
	AutoFixable bool
}

// DoctorResult contains all issues found and summary counts
type DoctorResult struct {
	Issues      []Issue
	Errors      int
	Warnings    int
	AutoFixable int
}

// NewDoctorCmd creates the doctor command
func NewDoctorCmd() *cobra.Command {
	var fix bool
	var jsonOutput bool
	var perf bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose workspace issues",
		Long: `Diagnose workspace configuration and health issues.

Examples:
  grove doctor           # Quick health check
  grove doctor --fix     # Auto-fix safe issues
  grove doctor --json    # Machine-readable output
  grove doctor --perf    # Disk space analysis`,
		Args: cobra.NoArgs,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(fix, jsonOutput, perf)
		},
	}

	cmd.Flags().BoolVar(&fix, "fix", false, "Auto-fix safe issues")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().BoolVar(&perf, "perf", false, "Disk space analysis")
	cmd.Flags().BoolP("help", "h", false, "Help for doctor")

	return cmd
}

func runDoctor(fix, jsonOutput, perf bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Verify we're in a grove workspace
	bareDir, err := workspace.FindBareDir(cwd)
	if err != nil {
		return err
	}

	// Gather issues
	result := &DoctorResult{}

	// Phase 2: Git detection
	detectGitIssues(bareDir, result)

	// TODO: Phase 3 - Config validation

	// Handle fix mode (Phase 4)
	if fix {
		// TODO: Implement auto-fix logic
		_ = fix // Use parameter to satisfy linter until Phase 4
	}

	// Output results
	if jsonOutput {
		// TODO: Implement JSON output (Phase 5)
		_ = jsonOutput // Use parameter to satisfy linter until Phase 5
	}

	if perf {
		// TODO: Implement disk space analysis (Phase 6)
		_ = perf // Use parameter to satisfy linter until Phase 6
	}

	// Output human-readable format
	return outputDoctorResult(result)
}

func detectGitIssues(bareDir string, result *DoctorResult) {
	// Get workspace root (parent of .bare)
	workspaceRoot := filepath.Dir(bareDir)

	// Check all worktrees for broken .git pointers
	detectBrokenGitPointers(workspaceRoot, bareDir, result)

	// Check for stale worktree entries in .bare/worktrees
	detectStaleWorktreeEntries(bareDir, result)
}

func detectBrokenGitPointers(workspaceRoot, bareDir string, result *DoctorResult) {
	// Get list of worktrees from git
	worktrees, err := git.ListWorktrees(bareDir)
	if err != nil {
		logger.Debug("Failed to list worktrees: %v", err)

		return
	}

	for _, worktreePath := range worktrees {
		// Check if .git file exists
		gitFile := filepath.Join(worktreePath, ".git")
		info, err := os.Stat(gitFile)

		if os.IsNotExist(err) {
			// .git file missing
			relPath, _ := filepath.Rel(workspaceRoot, worktreePath)
			result.Issues = append(result.Issues, Issue{
				Category:    CategoryGit,
				Severity:    SeverityError,
				Message:     "Broken .git pointer",
				Path:        relPath,
				FixHint:     "grove doctor --fix",
				AutoFixable: true,
			})

			continue
		}

		if err != nil {
			logger.Debug("Failed to stat .git file in %s: %v", worktreePath, err)

			continue
		}

		// If it's a directory, it's a regular repo, not a worktree - skip
		if info.IsDir() {
			continue
		}

		// Read and validate .git file content
		gitdir, err := git.GetWorktreeGitDir(worktreePath)
		if err != nil {
			// Invalid .git file format
			relPath, _ := filepath.Rel(workspaceRoot, worktreePath)
			result.Issues = append(result.Issues, Issue{
				Category:    CategoryGit,
				Severity:    SeverityError,
				Message:     "Broken .git pointer",
				Path:        relPath,
				Details:     []string{err.Error()},
				FixHint:     "grove doctor --fix",
				AutoFixable: true,
			})

			continue
		}

		// Check if the gitdir target exists
		if gitdir != "" {
			if _, err := os.Stat(gitdir); os.IsNotExist(err) {
				relPath, _ := filepath.Rel(workspaceRoot, worktreePath)
				result.Issues = append(result.Issues, Issue{
					Category:    CategoryGit,
					Severity:    SeverityError,
					Message:     "Broken .git pointer",
					Path:        relPath,
					Details:     []string{"gitdir target does not exist"},
					FixHint:     "grove doctor --fix",
					AutoFixable: true,
				})
			}
		}
	}
}

func detectStaleWorktreeEntries(bareDir string, result *DoctorResult) {
	// Check .bare/worktrees directory for orphaned entries
	worktreesDir := filepath.Join(bareDir, "worktrees")

	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		// No worktrees directory is fine
		if os.IsNotExist(err) {
			return
		}
		logger.Debug("Failed to read worktrees directory: %v", err)

		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		worktreeName := entry.Name()
		gitdirFile := filepath.Join(worktreesDir, worktreeName, "gitdir")

		// Read the gitdir file to find the worktree path
		content, err := os.ReadFile(gitdirFile) //nolint:gosec // Path derived from validated workspace
		if err != nil {
			// No gitdir file means stale entry
			result.Issues = append(result.Issues, Issue{
				Category:    CategoryGit,
				Severity:    SeverityError,
				Message:     "Stale worktree entry",
				Path:        worktreeName,
				Details:     []string{"missing gitdir file"},
				FixHint:     "grove doctor --fix",
				AutoFixable: true,
			})

			continue
		}

		// Check if the worktree directory exists
		worktreePath := strings.TrimSpace(string(content))
		if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
			result.Issues = append(result.Issues, Issue{
				Category:    CategoryGit,
				Severity:    SeverityError,
				Message:     "Stale worktree entry",
				Path:        worktreeName,
				Details:     []string{"worktree directory does not exist"},
				FixHint:     "grove doctor --fix",
				AutoFixable: true,
			})
		}
	}
}

func outputDoctorResult(result *DoctorResult) error {
	// Count issues by severity
	for _, issue := range result.Issues {
		switch issue.Severity {
		case SeverityError:
			result.Errors++
		case SeverityWarning:
			result.Warnings++
		}
		if issue.AutoFixable {
			result.AutoFixable++
		}
	}

	// If no issues, report clean
	if len(result.Issues) == 0 {
		if config.IsPlain() {
			fmt.Println("[ok] No issues found")
		} else {
			fmt.Println("✓ No issues found")
		}

		return nil
	}

	// Group issues by category
	gitIssues := filterIssuesByCategory(result.Issues, CategoryGit)
	configIssues := filterIssuesByCategory(result.Issues, CategoryConfig)

	// Output git issues
	if len(gitIssues) > 0 {
		outputCategoryIssues("Git Issues", gitIssues)
	}

	// Output config issues
	if len(configIssues) > 0 {
		outputCategoryIssues("Configuration", configIssues)
	}

	// Output summary
	fmt.Println()
	fmt.Printf("Summary: %d errors, %d warnings (%d auto-fixable)\n",
		result.Errors, result.Warnings, result.AutoFixable)

	// Return error to set exit code 1 if there are errors
	if result.Errors > 0 {
		return fmt.Errorf("found %d errors", result.Errors)
	}

	return nil
}

func filterIssuesByCategory(issues []Issue, category Category) []Issue {
	var filtered []Issue

	for _, issue := range issues {
		if issue.Category == category {
			filtered = append(filtered, issue)
		}
	}

	return filtered
}

func outputCategoryIssues(categoryName string, issues []Issue) {
	// Count errors and warnings in this category
	var errors, warnings int

	for _, issue := range issues {
		switch issue.Severity {
		case SeverityError:
			errors++
		case SeverityWarning:
			warnings++
		}
	}

	// Print category header
	var countParts []string
	if errors > 0 {
		countParts = append(countParts, fmt.Sprintf("%d errors", errors))
	}
	if warnings > 0 {
		countParts = append(countParts, fmt.Sprintf("%d warnings", warnings))
	}

	fmt.Printf("%s (%s)\n", categoryName, strings.Join(countParts, ", "))

	// Print each issue
	for _, issue := range issues {
		symbol := getIssueSymbol(issue.Severity)
		if issue.Path != "" {
			fmt.Printf("  %s %s in %s\n", symbol, issue.Message, issue.Path)
		} else {
			fmt.Printf("  %s %s\n", symbol, issue.Message)
		}

		// Print details as sub-items
		for _, detail := range issue.Details {
			if config.IsPlain() {
				fmt.Printf("    > %s\n", detail)
			} else {
				fmt.Printf("    ↳ %s\n", detail)
			}
		}
	}

	// Print auto-fix hint if any issues are auto-fixable
	hasAutoFixable := false

	for _, issue := range issues {
		if issue.AutoFixable {
			hasAutoFixable = true

			break
		}
	}

	if hasAutoFixable {
		if config.IsPlain() {
			fmt.Println("  -> Run: grove doctor --fix")
		} else {
			fmt.Println("  → Run: grove doctor --fix")
		}
	}
}

func getIssueSymbol(severity Severity) string {
	if config.IsPlain() {
		switch severity {
		case SeverityError:
			return "[x]"
		case SeverityWarning:
			return "[!]"
		case SeverityInfo:
			return "[i]"
		default:
			return "[-]"
		}
	}

	switch severity {
	case SeverityError:
		return "✗"
	case SeverityWarning:
		return "⚠"
	case SeverityInfo:
		return "→"
	default:
		return "•"
	}
}
