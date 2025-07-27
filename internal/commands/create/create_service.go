package create

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
	"github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/logger"
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
		Force:       options.Force,
	}

	if err := s.worktreeCreator.CreateWorktree(branchInfo.Name, worktreePath, worktreeOptions); err != nil {
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
		copiedFiles, err := s.handleFileCopying(options, worktreePath)
		if err != nil {
			// File copying failure is not critical - worktree was created successfully.
			s.logger.Warn("file copying failed", "error", err.Error())
		} else {
			result.CopiedFiles = copiedFiles
		}
	}

	s.logger.InfoOperation("worktree created successfully",
		"branch", result.BranchName,
		"path", result.WorktreePath,
		"was_created", result.WasCreated,
		"copied_files", result.CopiedFiles,
	)

	return result, nil
}

func (s *CreateServiceImpl) validateOptions(options *CreateOptions) error {
	if options.BranchName == "" {
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
	if isURL(input) {
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
		if len(parts) == 2 {
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
	switch inputInfo.Type {
	case InputTypeURL:
		// For URLs, we need to handle remote setup and branch resolution.
		// This is a simplified implementation - full URL handling would require more complexity.
		return s.branchResolver.ResolveBranch(inputInfo.ProcessedName, options.BaseBranch, options.CreateBranch)

	case InputTypeRemoteBranch:
		return s.branchResolver.ResolveRemoteBranch(inputInfo.OriginalName)

	case InputTypeBranch:
		return s.branchResolver.ResolveBranch(inputInfo.ProcessedName, options.BaseBranch, options.CreateBranch)

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
	sourceWorktree := options.SourceWorktree
	if sourceWorktree == "" {
		var err error
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
