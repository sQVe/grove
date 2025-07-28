package create

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
	"github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/validation"
)

type CreateServiceImpl struct {
	branchResolver  BranchResolver
	pathGenerator   PathGenerator
	worktreeCreator WorktreeCreator
	fileManager     FileManager
	logger          *logger.Logger
}

func NewCreateService(
	branchResolver BranchResolver,
	pathGenerator PathGenerator,
	worktreeCreator WorktreeCreator,
	fileManager FileManager,
) *CreateServiceImpl {
	return &CreateServiceImpl{
		branchResolver:  branchResolver,
		pathGenerator:   pathGenerator,
		worktreeCreator: worktreeCreator,
		fileManager:     fileManager,
		logger:          logger.WithComponent("create_service"),
	}
}

func (s *CreateServiceImpl) Create(options *CreateOptions) (*CreateResult, error) {
	s.logger.DebugOperation("starting create workflow",
		"branch", options.BranchName,
		"path", options.WorktreePath,
		"copy_files", options.CopyFiles,
	)

	if err := s.validateOptions(options); err != nil {
		return nil, &errors.GroveError{
			Code:      errors.ErrCodeConfigInvalid,
			Message:   "invalid create options",
			Cause:     err,
			Operation: "option_validation",
		}
	}

	// Notify about input processing
	inputInfo, err := s.classifyInput(options.BranchName)
	if err != nil {
		return nil, &errors.GroveError{
			Code:      errors.ErrCodeConfigInvalid,
			Message:   "failed to process input",
			Cause:     err,
			Operation: "input_classification",
			Context: map[string]interface{}{
				"input": options.BranchName,
			},
		}
	}

	// Provide informative progress for different input types
	if options.ProgressCallback != nil {
		switch inputInfo.Type {
		case InputTypeURL:
			options.ProgressCallback(fmt.Sprintf("Parsing URL: %s", inputInfo.OriginalName))
		case InputTypeRemoteBranch:
			options.ProgressCallback(fmt.Sprintf("Resolving remote branch: %s", inputInfo.OriginalName))
		case InputTypeBranch:
			options.ProgressCallback(fmt.Sprintf("Resolving branch: %s", inputInfo.OriginalName))
		}
	}

	branchInfo, err := s.resolveBranchInfo(inputInfo, options)
	if err != nil {
		return nil, &errors.GroveError{
			Code:      errors.ErrCodeGitOperation,
			Message:   "failed to resolve branch information",
			Cause:     err,
			Operation: "branch_resolution",
		}
	}

	worktreePath := options.WorktreePath
	if worktreePath == "" {
		if options.ProgressCallback != nil {
			options.ProgressCallback("Generating worktree path")
		}
		worktreePath, err = s.pathGenerator.GeneratePath(branchInfo.Name, "")
		if err != nil {
			return nil, &errors.GroveError{
				Code:      errors.ErrCodeFileSystem,
				Message:   "failed to generate worktree path",
				Cause:     err,
				Operation: "path_generation",
			}
		}
	}

	worktreeOptions := WorktreeOptions{
		TrackRemote: s.shouldTrackRemote(branchInfo, options),
		BaseBranch:  options.BaseBranch,
	}

	if options.ProgressCallback != nil {
		options.ProgressCallback(fmt.Sprintf("Creating worktree at %s", worktreePath))
	}

	if err := s.worktreeCreator.CreateWorktreeWithProgress(branchInfo.Name, worktreePath, worktreeOptions, options.ProgressCallback); err != nil {
		return nil, &errors.GroveError{
			Code:      errors.ErrCodeGitOperation,
			Message:   "failed to create worktree",
			Cause:     err,
			Operation: "worktree_creation",
			Context: map[string]interface{}{
				"branch": branchInfo.Name,
				"path":   worktreePath,
			},
		}
	}

	result := &CreateResult{
		WorktreePath: worktreePath,
		BranchName:   branchInfo.Name,
		WasCreated:   !branchInfo.Exists,
		BaseBranch:   options.BaseBranch,
	}

	if options.CopyFiles {
		if options.ProgressCallback != nil {
			options.ProgressCallback("Copying files to new worktree")
		}
		copiedFiles, err := s.handleFileCopying(options, worktreePath)
		if err != nil {
			// File copying failure is not critical - worktree was created successfully.
			// Only log at debug level to avoid cluttering user output.
			s.logger.DebugOperation("file copying failed", "error", err.Error())
		} else {
			result.CopiedFiles = copiedFiles
		}
	}

	s.logger.DebugOperation("worktree created successfully",
		"branch", result.BranchName,
		"path", result.WorktreePath,
		"was_created", result.WasCreated,
		"copied_files", result.CopiedFiles,
	)

	return result, nil
}

func (s *CreateServiceImpl) validateOptions(options *CreateOptions) error {
	if options == nil {
		return fmt.Errorf("options cannot be nil")
	}

	if strings.TrimSpace(options.BranchName) == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	if options.WorktreePath != "" {
		if strings.Contains(options.WorktreePath, "..") {
			return fmt.Errorf("path cannot contain '..' components")
		}
	}

	return nil
}

type InputType int

const (
	InputTypeBranch InputType = iota
	InputTypeURL
	InputTypeRemoteBranch
)

type InputInfo struct {
	Type          InputType
	OriginalName  string
	ProcessedName string
	URLInfo       *URLBranchInfo
	RemoteName    string
}

func (s *CreateServiceImpl) classifyInput(input string) (*InputInfo, error) {
	if validation.IsURL(input) {
		urlInfo, err := s.branchResolver.ResolveURL(input)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve URL: %w", err)
		}

		return &InputInfo{
			Type:          InputTypeURL,
			OriginalName:  input,
			ProcessedName: urlInfo.BranchName,
			URLInfo:       urlInfo,
		}, nil
	}

	if strings.Contains(input, "/") && !strings.HasPrefix(input, "/") && !strings.HasSuffix(input, "/") {
		parts := strings.SplitN(input, "/", 2)
		if len(parts) == 2 && s.branchResolver.RemoteExists(parts[0]) {
			return &InputInfo{
				Type:          InputTypeRemoteBranch,
				OriginalName:  input,
				ProcessedName: parts[1], // Branch name without remote prefix.
				RemoteName:    parts[0], // Remote name.
			}, nil
		}
	}

	return &InputInfo{
		Type:          InputTypeBranch,
		OriginalName:  input,
		ProcessedName: input,
	}, nil
}

func (s *CreateServiceImpl) resolveBranchInfo(inputInfo *InputInfo, options *CreateOptions) (*BranchInfo, error) {
	// For worktree creation, default to creating missing branches automatically
	// The --create flag is now optional - we always attempt to create missing branches
	shouldCreateBranch := true

	switch inputInfo.Type {
	case InputTypeURL:
		// For URLs, we need to handle remote setup and branch resolution.
		// This is a simplified implementation - full URL handling would require more complexity.
		return s.branchResolver.ResolveBranch(inputInfo.ProcessedName, options.BaseBranch, shouldCreateBranch)

	case InputTypeRemoteBranch:
		return s.branchResolver.ResolveRemoteBranch(inputInfo.OriginalName)

	case InputTypeBranch:
		return s.branchResolver.ResolveBranch(inputInfo.ProcessedName, options.BaseBranch, shouldCreateBranch)

	default:
		return nil, fmt.Errorf("unknown input type: %d", inputInfo.Type)
	}
}

func (s *CreateServiceImpl) shouldTrackRemote(branchInfo *BranchInfo, options *CreateOptions) bool {
	if branchInfo.IsRemote {
		return true
	}

	autoTrack := viper.GetBool("worktree.auto_track_remote")
	return autoTrack
}

func (s *CreateServiceImpl) handleFileCopying(options *CreateOptions, targetWorktree string) (int, error) {
	var sourceWorktree string
	var err error
	
	// If base branch is specified, try to find worktree for that branch first
	if options.BaseBranch != "" {
		sourceWorktree, err = s.fileManager.FindWorktreeByBranch(options.BaseBranch)
		if err != nil {
			s.logger.DebugOperation("failed to find worktree for base branch, falling back to auto-discovery", 
				"base_branch", options.BaseBranch, "error", err.Error())
		}
	}
	
	// Fall back to default branch worktree if base branch worktree not found
	if sourceWorktree == "" {
		sourceWorktree, err = s.fileManager.DiscoverSourceWorktree()
		if err != nil {
			return 0, fmt.Errorf("failed to discover source worktree: %w", err)
		}
	}

	patterns := options.CopyPatterns
	if len(patterns) == 0 {
		if options.CopyEnv {
			patterns = []string{".env*", "*.local", "local.*"}
		} else {
			patterns = viper.GetStringSlice("worktree.copy_files.patterns")
		}
	}

	if len(patterns) == 0 {
		return 0, nil
	}

	conflictStrategy := ConflictPrompt
	if configStrategy := viper.GetString("worktree.copy_files.on_conflict"); configStrategy != "" {
		conflictStrategy = ConflictStrategy(configStrategy)
	}

	copyOptions := CopyOptions{
		ConflictStrategy: conflictStrategy,
		DryRun:           false,
	}

	// Count files before copying to track success.
	// This is a simplified approach - a more complete implementation would track actual copied files.
	totalPatterns := len(patterns)

	if err := s.fileManager.CopyFiles(sourceWorktree, targetWorktree, patterns, copyOptions); err != nil {
		return 0, err
	}

	// Return an estimate of copied files (this could be improved with actual counting).
	return totalPatterns, nil
}
