package switchcmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sqve/grove/internal/completion"
	"github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/git"
)

// SwitchService provides worktree switching functionality with path resolution and validation.
// It handles exact and fuzzy matching of worktree names, validates paths for security,
// and integrates with Grove's existing completion and git execution systems.
type SwitchService interface {
	// Switch performs a complete worktree switch operation with the given options.
	// It returns a SwitchResult containing the path, command, and mode used.
	Switch(ctx context.Context, worktreeName string, options SwitchOptions) (*SwitchResult, error)
	
	// GetWorktreePath resolves a worktree name to its absolute file system path.
	// Supports both exact matching and fuzzy matching using completion logic.
	// Returns an error if the worktree is not found or path is invalid.
	GetWorktreePath(worktreeName string) (string, error)
	
	// ListWorktrees returns all available worktrees using Grove's completion system.
	// This provides access to cached worktree information for performance.
	ListWorktrees() ([]completion.WorktreeInfo, error)
	
	// GenerateShellIntegration creates shell-specific integration code for the given shell type.
	// Supported shells: bash, zsh, fish, powershell
	GenerateShellIntegration(shell string) (string, error)
}

// SwitchResult contains the outcome of a worktree switch operation.
type SwitchResult struct {
	// Path is the absolute file system path to the target worktree
	Path         string
	// WorktreeName is the resolved name of the worktree that was switched to
	WorktreeName string
	// Mode indicates which execution mode was used (auto, eval, subshell, etc.)
	Mode         SwitchMode
	// Command contains any shell command that should be executed (for eval mode)
	Command      string
}

// switchService implements SwitchService using Grove's git execution and completion systems.
type switchService struct {
	executor git.GitExecutor
}

func NewSwitchService(executor git.GitExecutor) SwitchService {
	return &switchService{
		executor: executor,
	}
}

func (s *switchService) Switch(ctx context.Context, worktreeName string, options SwitchOptions) (*SwitchResult, error) {
	// Core switching logic will be implemented in upcoming tasks
	return nil, nil
}

func (s *switchService) GetWorktreePath(worktreeName string) (string, error) {
	worktrees, err := s.ListWorktrees()
	if err != nil {
		return "", err
	}

	// Single iteration to collect names and check for exact match
	var exactMatch *completion.WorktreeInfo
	worktreeNames := make([]string, len(worktrees))

	for i, worktree := range worktrees {
		name := worktree.Name()
		worktreeNames[i] = name
		if name == worktreeName {
			exactMatch = &worktrees[i]
		}
	}

	if exactMatch != nil {
		return s.validateAndCleanPath(exactMatch.Path)
	}

	// Try fuzzy matching using existing completion logic
	matches := completion.FilterCompletions(worktreeNames, worktreeName)

	if len(matches) == 0 {
		return "", errors.ErrSourceWorktreeNotFound(worktreeName).
			WithContext("available_worktrees", worktreeNames).
			WithContext("suggestion", "Run 'grove list' to see available worktrees")
	}

	if len(matches) == 1 {
		for _, worktree := range worktrees {
			if worktree.Name() == matches[0] {
				return s.validateAndCleanPath(worktree.Path)
			}
		}
	}

	return "", errors.NewGroveError(
		errors.ErrCodeConfigInvalid,
		fmt.Sprintf("Ambiguous worktree name '%s'", worktreeName),
		nil,
	).WithContext("matches", matches).
		WithContext("suggestion", "Be more specific or use the full worktree name")
}

func (s *switchService) validateAndCleanPath(path string) (string, error) {
	// Clean the path to resolve any .. or . elements and prevent directory traversal
	cleanPath := filepath.Clean(path)

	// Check for directory traversal attempts in the original path
	if strings.Contains(path, "..") {
		return "", errors.ErrPathTraversal(path)
	}

	// Convert to absolute path for consistency
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", errors.ErrDirectoryAccess(path, err)
	}

	// Verify the cleaned absolute path doesn't contain traversal elements
	if strings.Contains(absPath, "..") {
		return "", errors.ErrPathTraversal(path)
	}

	return absPath, nil
}

func (s *switchService) ListWorktrees() ([]completion.WorktreeInfo, error) {
	// Create a completion context to leverage existing worktree parsing logic
	ctx := &completion.CompletionContext{
		Executor: s.executor,
	}

	return completion.GetWorktreeInfo(ctx)
}

func (s *switchService) GenerateShellIntegration(shell string) (string, error) {
	// Shell integration generation will be implemented in later tasks
	return "", nil
}
