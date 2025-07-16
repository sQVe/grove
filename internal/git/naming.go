package git

import (
	"regexp"
	"strings"
)

// BranchToDirectoryName converts a branch name to a filesystem-safe directory name.
// This function handles common filesystem-unsafe characters by replacing them with safe alternatives.
//
// Examples:
//   - "fix/123" -> "fix-123"
//   - "feature/user/auth" -> "feature-user-auth"
//   - "bugfix/issue#456" -> "bugfix-issue-456"
//   - "hotfix/v1.2.3" -> "hotfix-v1.2.3"
func BranchToDirectoryName(branchName string) string {
	if branchName == "" {
		return ""
	}

	// Replace filesystem-unsafe characters with safe alternatives
	dirName := branchName

	// Replace forward slashes with hyphens (most common case)
	dirName = strings.ReplaceAll(dirName, "/", "-")

	// Replace backslashes with hyphens (Windows compatibility)
	dirName = strings.ReplaceAll(dirName, "\\", "-")

	// Replace other problematic characters
	dirName = strings.ReplaceAll(dirName, ":", "-")
	dirName = strings.ReplaceAll(dirName, "*", "-")
	dirName = strings.ReplaceAll(dirName, "?", "-")
	dirName = strings.ReplaceAll(dirName, "\"", "-")
	dirName = strings.ReplaceAll(dirName, "<", "-")
	dirName = strings.ReplaceAll(dirName, ">", "-")
	dirName = strings.ReplaceAll(dirName, "|", "-")
	dirName = strings.ReplaceAll(dirName, "#", "-")

	// Remove or replace spaces and tabs
	dirName = strings.ReplaceAll(dirName, " ", "-")
	dirName = strings.ReplaceAll(dirName, "\t", "-")

	// Replace multiple consecutive hyphens with single hyphen
	multiHyphen := regexp.MustCompile(`-+`)
	dirName = multiHyphen.ReplaceAllString(dirName, "-")

	// Remove leading/trailing hyphens
	dirName = strings.Trim(dirName, "-")

	// Handle edge case where the result might be empty
	if dirName == "" {
		return "worktree"
	}

	return dirName
}

// IsValidDirectoryName checks if a directory name is filesystem-safe.
// This function validates that the name doesn't contain problematic characters
// that could cause issues on various filesystems.
func IsValidDirectoryName(name string) bool {
	if name == "" {
		return false
	}

	// Check for problematic characters
	problematicChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "#"}
	for _, char := range problematicChars {
		if strings.Contains(name, char) {
			return false
		}
	}

	// Check for leading/trailing spaces or dots (problematic on some filesystems)
	if strings.HasPrefix(name, " ") || strings.HasSuffix(name, " ") ||
		strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") {
		return false
	}

	// Check for reserved names on Windows
	reservedNames := []string{
		"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4",
		"COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5",
		"LPT6", "LPT7", "LPT8", "LPT9",
	}

	upperName := strings.ToUpper(name)
	for _, reserved := range reservedNames {
		if upperName == reserved {
			return false
		}
	}

	return true
}

// NormalizeBranchName ensures a branch name is valid for Git operations.
// This function primarily validates that the branch name follows Git's naming conventions.
func NormalizeBranchName(branchName string) string {
	if branchName == "" {
		return ""
	}

	// Git branch names cannot start with a hyphen
	if strings.HasPrefix(branchName, "-") {
		branchName = "branch" + branchName
	}

	// Git branch names cannot end with a dot
	branchName = strings.TrimSuffix(branchName, ".")

	return branchName
}
