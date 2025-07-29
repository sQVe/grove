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

type SwitchService interface {
	Switch(ctx context.Context, worktreeName string, options SwitchOptions) (*SwitchResult, error)
	GetWorktreePath(worktreeName string) (string, error)
	ListWorktrees() ([]completion.WorktreeInfo, error)
	GenerateShellIntegration(shell string) (string, error)
}

type SwitchResult struct {
	Path         string
	WorktreeName string
	Mode         SwitchMode
	Command      string
}

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
