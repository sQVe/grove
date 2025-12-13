package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/fs"
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
	Fixed       bool
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

	// Phase 3: Config validation
	detectConfigIssues(bareDir, result)

	// Handle fix mode (Phase 4)
	if fix {
		fixIssues(bareDir, result)

		// Re-run detection after fixes to get current state
		result = &DoctorResult{}
		detectGitIssues(bareDir, result)
		detectConfigIssues(bareDir, result)
	}

	// Output results
	if jsonOutput {
		return outputJSONResult(result)
	}

	if perf {
		if err := outputPerfAnalysis(bareDir); err != nil {
			return err
		}
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
		// Note: gitdir file contains path to .git FILE (e.g., /path/worktree/.git)
		// We need to check the parent directory (the actual worktree)
		gitFilePath := strings.TrimSpace(string(content))
		worktreeDir := filepath.Dir(gitFilePath)
		if _, err := os.Stat(worktreeDir); os.IsNotExist(err) {
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

// Phase 3: Config validation

func detectConfigIssues(bareDir string, result *DoctorResult) {
	workspaceRoot := filepath.Dir(bareDir)

	// Check .grove.toml syntax
	detectInvalidToml(workspaceRoot, result)

	// Check hook commands
	detectInvalidHooks(workspaceRoot, result)

	// Check stale lock files
	detectStaleLockFiles(workspaceRoot, result)
}

func detectInvalidToml(workspaceRoot string, result *DoctorResult) {
	tomlPath := filepath.Join(workspaceRoot, ".grove.toml")

	// Check if file exists
	if _, err := os.Stat(tomlPath); os.IsNotExist(err) {
		return
	}

	// Try to parse the TOML file
	content, err := os.ReadFile(tomlPath) //nolint:gosec // Path derived from validated workspace
	if err != nil {
		logger.Debug("Failed to read .grove.toml: %v", err)

		return
	}

	// Attempt to decode - we don't care about the result, just whether it parses
	var dummy interface{}
	if _, err := toml.Decode(string(content), &dummy); err != nil {
		result.Issues = append(result.Issues, Issue{
			Category:    CategoryConfig,
			Severity:    SeverityError,
			Message:     "Invalid .grove.toml",
			Path:        ".grove.toml",
			Details:     []string{err.Error()},
			AutoFixable: false,
		})
	}
}

func detectInvalidHooks(workspaceRoot string, result *DoctorResult) {
	tomlPath := filepath.Join(workspaceRoot, ".grove.toml")

	// Check if file exists
	if _, err := os.Stat(tomlPath); os.IsNotExist(err) {
		return
	}

	// Parse config to get hooks
	content, err := os.ReadFile(tomlPath) //nolint:gosec // Path derived from validated workspace
	if err != nil {
		return
	}

	var cfg struct {
		Hooks struct {
			Add []string `toml:"add"`
		} `toml:"hooks"`
	}

	if _, err := toml.Decode(string(content), &cfg); err != nil {
		// Invalid TOML - already reported by detectInvalidToml
		return
	}

	// Check each hook command
	for _, cmd := range cfg.Hooks.Add {
		// Extract the executable (first word)
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			continue
		}

		executable := parts[0]

		// Check if executable exists in PATH
		if _, err := exec.LookPath(executable); err != nil {
			result.Issues = append(result.Issues, Issue{
				Category:    CategoryConfig,
				Severity:    SeverityWarning,
				Message:     "Hook command not found",
				Path:        executable,
				Details:     []string{"Ensure " + executable + " is in PATH"},
				AutoFixable: false,
			})
		}
	}
}

func detectStaleLockFiles(workspaceRoot string, result *DoctorResult) {
	lockPath := filepath.Join(workspaceRoot, ".grove-convert.lock")

	if _, err := os.Stat(lockPath); err == nil {
		result.Issues = append(result.Issues, Issue{
			Category:    CategoryConfig,
			Severity:    SeverityWarning,
			Message:     "Stale lock file",
			Path:        ".grove-convert.lock",
			Details:     []string{"May block grove operations"},
			FixHint:     "rm " + lockPath,
			AutoFixable: true,
		})
	}
}

// Phase 4: Fix capability

func fixIssues(bareDir string, result *DoctorResult) {
	workspaceRoot := filepath.Dir(bareDir)

	for i := range result.Issues {
		issue := &result.Issues[i]
		if !issue.AutoFixable {
			continue
		}

		var err error

		switch issue.Message {
		case "Stale lock file":
			err = fixStaleLockFile(workspaceRoot, issue)
		case "Stale worktree entry":
			err = fixStaleWorktreeEntry(bareDir, issue)
		case "Broken .git pointer":
			err = fixBrokenGitPointer(bareDir, workspaceRoot, issue)
		}

		if err != nil {
			logger.Warning("Failed to fix %s: %v", issue.Message, err)
		} else {
			issue.Fixed = true
			fmt.Printf("Fixed: %s (%s)\n", issue.Message, issue.Path)
		}
	}
}

func fixStaleLockFile(workspaceRoot string, issue *Issue) error {
	lockPath := filepath.Join(workspaceRoot, issue.Path)

	return os.Remove(lockPath)
}

func fixStaleWorktreeEntry(bareDir string, issue *Issue) error {
	worktreeDir := filepath.Join(bareDir, "worktrees", issue.Path)

	return os.RemoveAll(worktreeDir)
}

func fixBrokenGitPointer(bareDir, workspaceRoot string, issue *Issue) error {
	worktreePath := filepath.Join(workspaceRoot, issue.Path)
	gitFile := filepath.Join(worktreePath, ".git")

	// Find the gitdir for this worktree
	worktreeName := filepath.Base(issue.Path)
	gitdirPath := filepath.Join(bareDir, "worktrees", worktreeName)

	// Verify the gitdir exists in .bare/worktrees
	if _, err := os.Stat(gitdirPath); os.IsNotExist(err) {
		return fmt.Errorf("cannot fix: gitdir not found at %s", gitdirPath)
	}

	// Calculate relative path from worktree to gitdir
	relPath, err := filepath.Rel(worktreePath, gitdirPath)
	if err != nil {
		return fmt.Errorf("cannot compute relative path: %w", err)
	}

	content := fmt.Sprintf("gitdir: %s\n", relPath)

	return os.WriteFile(gitFile, []byte(content), fs.FileGit) //nolint:gosec // Git files need 0644 permissions
}

// Phase 5: JSON output

type jsonIssue struct {
	Category    string   `json:"category"`
	Severity    string   `json:"severity"`
	Message     string   `json:"message"`
	Path        string   `json:"path,omitempty"`
	Details     []string `json:"details,omitempty"`
	FixHint     string   `json:"fixHint,omitempty"`
	AutoFixable bool     `json:"autoFixable"`
	Fixed       bool     `json:"fixed"`
}

type jsonSummary struct {
	Errors      int `json:"errors"`
	Warnings    int `json:"warnings"`
	AutoFixable int `json:"autoFixable"`
}

type jsonResult struct {
	Issues  []jsonIssue `json:"issues"`
	Summary jsonSummary `json:"summary"`
}

func outputJSONResult(result *DoctorResult) error {
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

	// Convert to JSON-friendly structure
	jsonRes := jsonResult{
		Issues: make([]jsonIssue, 0, len(result.Issues)),
		Summary: jsonSummary{
			Errors:      result.Errors,
			Warnings:    result.Warnings,
			AutoFixable: result.AutoFixable,
		},
	}

	for _, issue := range result.Issues {
		jsonRes.Issues = append(jsonRes.Issues, jsonIssue{
			Category:    categoryToString(issue.Category),
			Severity:    severityToString(issue.Severity),
			Message:     issue.Message,
			Path:        issue.Path,
			Details:     issue.Details,
			FixHint:     issue.FixHint,
			AutoFixable: issue.AutoFixable,
			Fixed:       issue.Fixed,
		})
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(jsonRes); err != nil {
		return err
	}

	// Return error to set exit code 1 if there are errors
	if result.Errors > 0 {
		return fmt.Errorf("found %d errors", result.Errors)
	}

	return nil
}

func categoryToString(c Category) string {
	switch c {
	case CategoryGit:
		return "git"
	case CategoryConfig:
		return "config"
	default:
		return "unknown"
	}
}

func severityToString(s Severity) string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return "unknown"
	}
}

// Phase 6: Performance analysis

func outputPerfAnalysis(bareDir string) error {
	workspaceRoot := filepath.Dir(bareDir)

	fmt.Println("Disk Usage")
	fmt.Println()

	// Get worktrees
	worktrees, err := git.ListWorktrees(bareDir)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Calculate size for each worktree
	for _, worktreePath := range worktrees {
		size, err := calculateDirSize(worktreePath)
		if err != nil {
			logger.Debug("Failed to calculate size for %s: %v", worktreePath, err)

			continue
		}

		relPath, _ := filepath.Rel(workspaceRoot, worktreePath)
		fmt.Printf("  %s  %s\n", formatSize(size), relPath)
	}

	// Calculate .bare size
	bareSize, err := calculateDirSize(bareDir)
	if err == nil {
		fmt.Printf("  %s  .bare (shared)\n", formatSize(bareSize))
	}

	return nil
}

func calculateDirSize(path string) (int64, error) {
	var size int64

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			size += info.Size()
		}

		return nil
	})

	return size, err
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%6.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%6.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%6.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%6d B ", bytes)
	}
}
