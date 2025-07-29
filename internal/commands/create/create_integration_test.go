//go:build integration
// +build integration

package create

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateUniqueBranchName creates a unique branch name using timestamp and random suffix
func generateUniqueBranchName(prefix string) string {
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	random, _ := rand.Int(rand.Reader, big.NewInt(10000))
	return fmt.Sprintf("%s-%d-%d", prefix, timestamp, random.Int64())
}

func TestCreateCommand_Integration_BasicWorktreeCreation(t *testing.T) {
	// Set up test repository
	testDir := setupTestRepository(t)
	defer func() {
		_ = os.RemoveAll(testDir)
	}()

	// Change to test directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.Chdir(testDir))

	// Set up git helper for unique branch names
	gitExec := git.DefaultExecutor
	uniqueBranch := generateUniqueBranchName("test-feature")

	// Create a feature branch
	_, err = gitExec.Execute("checkout", "-b", uniqueBranch)
	require.NoError(t, err)

	// Switch back to main
	_, err = gitExec.Execute("checkout", "main")
	require.NoError(t, err)

	// Set up create service components
	branchResolver := NewBranchResolver(gitExec)
	pathGenerator := NewPathGenerator()
	worktreeCreator := NewWorktreeCreator(gitExec)
	fileManager := NewFileManager(gitExec)

	service := NewCreateService(branchResolver, pathGenerator, worktreeCreator, fileManager)

	// Test creating worktree for existing branch
	options := &CreateOptions{
		BranchName: uniqueBranch,
		CopyFiles:  false,
	}

	result, err := service.Create(options)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uniqueBranch, result.BranchName)
	assert.False(t, result.WasCreated)
	assert.DirExists(t, result.WorktreePath)

	// Verify the worktree was created correctly
	worktrees, err := gitExec.Execute("worktree", "list")
	require.NoError(t, err)
	assert.Contains(t, worktrees, result.WorktreePath)
	assert.Contains(t, worktrees, uniqueBranch)
}

func TestCreateCommand_Integration_NewBranchCreation(t *testing.T) {
	// Set up test repository
	testDir := setupTestRepository(t)
	defer func() {
		_ = os.RemoveAll(testDir)
	}()

	// Change to test directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.Chdir(testDir))

	// Set up create service components
	gitExec := git.DefaultExecutor
	uniqueBranch := generateUniqueBranchName("new-feature")
	
	branchResolver := NewBranchResolver(gitExec)
	pathGenerator := NewPathGenerator()
	worktreeCreator := NewWorktreeCreator(gitExec)
	fileManager := NewFileManager(gitExec)

	service := NewCreateService(branchResolver, pathGenerator, worktreeCreator, fileManager)

	// Test creating worktree for new branch
	options := &CreateOptions{
		BranchName: uniqueBranch,
		BaseBranch: "main",
		CopyFiles:  false,
	}

	result, err := service.Create(options)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uniqueBranch, result.BranchName)
	assert.True(t, result.WasCreated)
	assert.DirExists(t, result.WorktreePath)

	// Verify the branch and worktree were created correctly
	branches, err := gitExec.Execute("branch", "--list", uniqueBranch)
	require.NoError(t, err)
	assert.Contains(t, branches, uniqueBranch)

	worktrees, err := gitExec.Execute("worktree", "list")
	require.NoError(t, err)
	assert.Contains(t, worktrees, result.WorktreePath)
	assert.Contains(t, worktrees, uniqueBranch)
}

func TestCreateCommand_Integration_FileCopying(t *testing.T) {
	// Set up test repository with files to copy
	testDir := setupTestRepositoryWithFiles(t)
	defer func() {
		_ = os.RemoveAll(testDir)
	}()

	// Change to test directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.Chdir(testDir))

	// Set up create service components
	gitExec := git.DefaultExecutor
	uniqueBranch := generateUniqueBranchName("feature-with-files")
	
	branchResolver := NewBranchResolver(gitExec)
	pathGenerator := NewPathGenerator()
	worktreeCreator := NewWorktreeCreator(gitExec)
	fileManager := NewFileManager(gitExec)

	service := NewCreateService(branchResolver, pathGenerator, worktreeCreator, fileManager)

	// Create a feature branch first
	_, err = gitExec.Execute("checkout", "-b", uniqueBranch)
	require.NoError(t, err)
	_, err = gitExec.Execute("checkout", "main")
	require.NoError(t, err)

	// Test creating worktree with file copying
	options := &CreateOptions{
		BranchName:   uniqueBranch,
		CopyFiles:    true,
		CopyPatterns: []string{".env*", ".vscode/*"}, // Specify patterns to copy
	}

	result, err := service.Create(options)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.DirExists(t, result.WorktreePath)

	// Verify files were copied
	envFile := filepath.Join(result.WorktreePath, ".env")
	assert.FileExists(t, envFile)

	envLocalFile := filepath.Join(result.WorktreePath, ".env.local")
	assert.FileExists(t, envLocalFile)

	vscodeSettings := filepath.Join(result.WorktreePath, ".vscode", "settings.json")
	assert.FileExists(t, vscodeSettings)

	// Verify that README.md exists (part of git branch) but wasn't copied from source worktree
	// Git worktree creation includes all files from the branch, regardless of copy patterns
	readmeFile := filepath.Join(result.WorktreePath, "README.md")
	assert.FileExists(t, readmeFile) // It exists because it's part of the git branch

	// Verify file contents match
	expectedEnv := "DATABASE_URL=postgres://localhost"
	actualEnv, err := os.ReadFile(envFile)
	require.NoError(t, err)
	assert.Equal(t, expectedEnv, string(actualEnv))
}

func TestCreateCommand_Integration_PathGeneration(t *testing.T) {
	// Set up test repository
	testDir := setupTestRepository(t)
	defer func() {
		_ = os.RemoveAll(testDir)
	}()

	// Change to test directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.Chdir(testDir))

	// Initialize Grove configuration
	cfg := &config.Config{
		Worktree: struct {
			NamingPattern    string                 `mapstructure:"naming_pattern"`
			CleanupThreshold time.Duration          `mapstructure:"cleanup_threshold"`
			BasePath         string                 `mapstructure:"base_path"`
			AutoTrackRemote  bool                   `mapstructure:"auto_track_remote"`
			CopyFiles        config.CopyFilesConfig `mapstructure:"copy_files"`
		}{
			BasePath: filepath.Join(testDir, "worktrees"),
		},
	}

	// Set up viper configuration for the test
	viper.Set("worktree.base_path", cfg.Worktree.BasePath)

	// Set up create service components
	gitExec := git.DefaultExecutor
	branchResolver := NewBranchResolver(gitExec)
	pathGenerator := NewPathGenerator()
	worktreeCreator := NewWorktreeCreator(gitExec)
	fileManager := NewFileManager(gitExec)

	service := NewCreateService(branchResolver, pathGenerator, worktreeCreator, fileManager)

	// Create a feature branch with special characters
	branchName := "feature/complex-branch-name"
	_, err = gitExec.Execute("checkout", "-b", branchName)
	require.NoError(t, err)
	_, err = gitExec.Execute("checkout", "main")
	require.NoError(t, err)

	// Test creating worktree with automatic path generation
	options := &CreateOptions{
		BranchName: branchName,
		CopyFiles:  false,
	}

	result, err := service.Create(options)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.DirExists(t, result.WorktreePath)

	// Verify path was sanitized properly
	expectedBasename := "feature-complex-branch-name"
	assert.Contains(t, result.WorktreePath, expectedBasename)
	assert.Contains(t, result.WorktreePath, cfg.Worktree.BasePath)
}

func TestCreateCommand_Integration_CollisionResolution(t *testing.T) {
	// Set up test repository
	testDir := setupTestRepository(t)
	defer func() {
		_ = os.RemoveAll(testDir)
	}()

	// Change to test directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.Chdir(testDir))

	// Initialize Grove configuration
	cfg := &config.Config{
		Worktree: struct {
			NamingPattern    string                 `mapstructure:"naming_pattern"`
			CleanupThreshold time.Duration          `mapstructure:"cleanup_threshold"`
			BasePath         string                 `mapstructure:"base_path"`
			AutoTrackRemote  bool                   `mapstructure:"auto_track_remote"`
			CopyFiles        config.CopyFilesConfig `mapstructure:"copy_files"`
		}{
			BasePath: filepath.Join(testDir, "worktrees"),
		},
	}

	// Set up viper configuration for the test
	viper.Set("worktree.base_path", cfg.Worktree.BasePath)

	// Set up create service components
	gitExec := git.DefaultExecutor
	uniqueBranch := generateUniqueBranchName("feature-branch")
	
	// Create a directory that would conflict
	// Use git.BranchToDirectoryName to match what the path generator will do
	dirName := git.BranchToDirectoryName(uniqueBranch)
	conflictPath := filepath.Join(cfg.Worktree.BasePath, dirName)
	require.NoError(t, os.MkdirAll(conflictPath, 0o755))

	branchResolver := NewBranchResolver(gitExec)
	pathGenerator := NewPathGenerator()
	worktreeCreator := NewWorktreeCreator(gitExec)
	fileManager := NewFileManager(gitExec)

	service := NewCreateService(branchResolver, pathGenerator, worktreeCreator, fileManager)

	// Create a feature branch
	_, err = gitExec.Execute("checkout", "-b", uniqueBranch)
	require.NoError(t, err)
	_, err = gitExec.Execute("checkout", "main")
	require.NoError(t, err)

	// Test creating worktree with path collision
	options := &CreateOptions{
		BranchName: uniqueBranch,
		CopyFiles:  false,
	}

	result, err := service.Create(options)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.DirExists(t, result.WorktreePath)

	// Verify a unique path was generated
	assert.NotEqual(t, conflictPath, result.WorktreePath)
	assert.Contains(t, result.WorktreePath, cfg.Worktree.BasePath)

	// Should contain a suffix like -2, -3, etc.
	basename := filepath.Base(result.WorktreePath)
	assert.True(t, strings.HasPrefix(basename, dirName+"-") || basename == dirName+"-1", 
		"basename %s should have collision suffix starting with %s-", basename, dirName)
	assert.NotEqual(t, dirName, basename)
}

func TestCreateCommand_Integration_ConfigurationIntegration(t *testing.T) {
	// Set up test repository
	testDir := setupTestRepository(t)
	defer func() {
		_ = os.RemoveAll(testDir)
	}()

	// Change to test directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.Chdir(testDir))

	// Initialize Grove configuration with custom settings
	customBasePath := filepath.Join(testDir, "custom-worktrees")
	cfg := &config.Config{
		Worktree: struct {
			NamingPattern    string                 `mapstructure:"naming_pattern"`
			CleanupThreshold time.Duration          `mapstructure:"cleanup_threshold"`
			BasePath         string                 `mapstructure:"base_path"`
			AutoTrackRemote  bool                   `mapstructure:"auto_track_remote"`
			CopyFiles        config.CopyFilesConfig `mapstructure:"copy_files"`
		}{
			BasePath: customBasePath,
			CopyFiles: config.CopyFilesConfig{
				Patterns:   []string{".env", ".config/"},
				OnConflict: "overwrite",
			},
		},
		Create: struct {
			DefaultBaseBranch  string `mapstructure:"default_base_branch"`
			PromptForNewBranch bool   `mapstructure:"prompt_for_new_branch"`
			AutoCreateParents  bool   `mapstructure:"auto_create_parents"`
		}{
			DefaultBaseBranch:  "main",
			PromptForNewBranch: false,
			AutoCreateParents:  true,
		},
	}

	// Set up viper configuration for the test
	viper.Set("worktree.base_path", cfg.Worktree.BasePath)

	// Set up create service components
	gitExec := git.DefaultExecutor
	uniqueBranch := generateUniqueBranchName("config-test")
	
	branchResolver := NewBranchResolver(gitExec)
	pathGenerator := NewPathGenerator()
	worktreeCreator := NewWorktreeCreator(gitExec)
	fileManager := NewFileManager(gitExec)

	service := NewCreateService(branchResolver, pathGenerator, worktreeCreator, fileManager)

	// Test creating worktree with configuration integration
	options := &CreateOptions{
		BranchName: uniqueBranch,
		BaseBranch: "main",
		CopyFiles:  false,
	}

	result, err := service.Create(options)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.DirExists(t, result.WorktreePath)

	// Verify custom base path was used
	assert.Contains(t, result.WorktreePath, customBasePath)

	// Verify branch was created with default base
	branches, err := gitExec.Execute("branch", "--list", uniqueBranch)
	require.NoError(t, err)
	assert.Contains(t, branches, uniqueBranch)
}

func TestCreateCommand_Integration_ErrorHandling(t *testing.T) {
	// Set up test repository
	testDir := setupTestRepository(t)
	defer func() {
		_ = os.RemoveAll(testDir)
	}()

	// Change to test directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.Chdir(testDir))

	// Initialize Grove configuration
	_ = &config.Config{
		Worktree: struct {
			NamingPattern    string                 `mapstructure:"naming_pattern"`
			CleanupThreshold time.Duration          `mapstructure:"cleanup_threshold"`
			BasePath         string                 `mapstructure:"base_path"`
			AutoTrackRemote  bool                   `mapstructure:"auto_track_remote"`
			CopyFiles        config.CopyFilesConfig `mapstructure:"copy_files"`
		}{
			BasePath: filepath.Join(testDir, "worktrees"),
		},
	}

	// Set up create service components
	gitExec := git.DefaultExecutor
	branchResolver := NewBranchResolver(gitExec)
	pathGenerator := NewPathGenerator()
	worktreeCreator := NewWorktreeCreator(gitExec)
	fileManager := NewFileManager(gitExec)

	service := NewCreateService(branchResolver, pathGenerator, worktreeCreator, fileManager)

	// Test creating worktree with invalid branch name (should cause validation error)
	options := &CreateOptions{
		BranchName: "invalid..branch..name", // Invalid branch name with consecutive dots
		CopyFiles:  false,
	}

	result, err := service.Create(options)

	require.Error(t, err)
	assert.Nil(t, result)

	// Should be a Grove error indicating validation failure
	assert.Contains(t, err.Error(), "invalid branch name")
}

func TestCreateCommand_Integration_PerformanceRequirements(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Set up test repository
	testDir := setupTestRepository(t)
	defer func() {
		_ = os.RemoveAll(testDir)
	}()

	// Change to test directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.Chdir(testDir))

	// Initialize Grove configuration
	_ = &config.Config{
		Worktree: struct {
			NamingPattern    string                 `mapstructure:"naming_pattern"`
			CleanupThreshold time.Duration          `mapstructure:"cleanup_threshold"`
			BasePath         string                 `mapstructure:"base_path"`
			AutoTrackRemote  bool                   `mapstructure:"auto_track_remote"`
			CopyFiles        config.CopyFilesConfig `mapstructure:"copy_files"`
		}{
			BasePath: filepath.Join(testDir, "worktrees"),
		},
	}

	// Set up create service components
	gitExec := git.DefaultExecutor
	branchResolver := NewBranchResolver(gitExec)
	pathGenerator := NewPathGenerator()
	worktreeCreator := NewWorktreeCreator(gitExec)
	fileManager := NewFileManager(gitExec)

	service := NewCreateService(branchResolver, pathGenerator, worktreeCreator, fileManager)

	// Create a feature branch
	_, err = gitExec.Execute("checkout", "-b", "performance-test")
	require.NoError(t, err)
	_, err = gitExec.Execute("checkout", "main")
	require.NoError(t, err)

	// Test performance requirements (< 5 seconds for local branches)
	start := time.Now()

	options := &CreateOptions{
		BranchName: "performance-test",
		CopyFiles:  false,
	}

	result, err := service.Create(options)

	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.DirExists(t, result.WorktreePath)

	// Performance requirement: < 5 seconds for local branches
	assert.True(t, elapsed.Seconds() < 5.0, "Create operation took %v, should be < 5 seconds", elapsed)
}

// Helper functions for integration tests

func setupTestRepository(t *testing.T) string {
	testDir := t.TempDir()

	// Initialize git repository
	gitExec := git.DefaultExecutor

	// Change to test directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.Chdir(testDir))

	// Initialize repository
	_, err = gitExec.Execute("init")
	require.NoError(t, err)

	// Configure git user (required for commits)
	_, err = gitExec.Execute("config", "user.name", "Test User")
	require.NoError(t, err)
	_, err = gitExec.Execute("config", "user.email", "test@example.com")
	require.NoError(t, err)

	// Create initial commit
	initialFile := filepath.Join(testDir, "README.md")
	require.NoError(t, os.WriteFile(initialFile, []byte("# Test Repository"), 0o644))

	_, err = gitExec.Execute("add", "README.md")
	require.NoError(t, err)
	_, err = gitExec.Execute("commit", "-m", "Initial commit")
	require.NoError(t, err)

	return testDir
}

func setupTestRepositoryWithFiles(t *testing.T) string {
	testDir := setupTestRepository(t)

	// Change to test directory for file creation
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.Chdir(testDir))

	// Create test files for copying
	testFiles := map[string]string{
		".env":                  "DATABASE_URL=postgres://localhost",
		".env.local":            "DEBUG=true",
		".vscode/settings.json": `{"editor.tabSize": 2}`,
		".vscode/launch.json":   `{"version": "0.2.0"}`,
		"src/main.go":           "package main\n\nfunc main() {}\n",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(testDir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o644))
	}

	// Add files to git (but don't track .env files in real scenario)
	gitExec := git.DefaultExecutor
	_, err = gitExec.Execute("add", "src/")
	require.NoError(t, err)
	_, err = gitExec.Execute("commit", "-m", "Add source files")
	require.NoError(t, err)

	return testDir
}
