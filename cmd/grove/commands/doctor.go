package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
	_, err = workspace.FindBareDir(cwd)
	if err != nil {
		return err
	}

	// Gather issues (Phase 2-3 will populate this)
	result := &DoctorResult{}

	// TODO: Phase 2 - Git detection
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
		fmt.Println("✓ No issues found")

		return nil
	}

	// TODO: Group and display issues by category

	return nil
}
