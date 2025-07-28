//go:build integration
// +build integration

package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sqve/grove/internal/commands/create"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCommand_Integration_BasicWorktreeCreation(t *testing.T) {
	// Set up test repository
	testDir := setupTestRepository(t)
	defer testutils.CleanupTestDirectory(t, testDir)

	// Change to test directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.Chdir(testDir))

	// Create a feature branch
	gitExec := git.NewGitExecutor()
	_, err = gitExec.Execute("checkout", "-b", "feature-branch")
	require.NoError(t, err)

	// Switch back to main
	_, err = gitExec.Execute("checkout", "main")
	require.NoError(t, err)

	// Initialize Grove configuration
	cfg := &config.Config{
		Worktree: config.WorktreeConfig{
			BasePath: filepath.Join(testDir, "worktrees"),
		},
	}

	// Set up create service components
	branchResolver := create.NewBranchResolver(gitExec)
	pathGenerator := create.NewPathGenerator(cfg)
	worktreeCreator := create.NewWorktreeCreator(gitExec)
	fileManager := create.NewFileManager(gitExec)
	
	service := create.NewCreateService(branchResolver, pathGenerator, worktreeCreator, fileManager)

	// Test creating worktree for existing branch
	options := &create.CreateOptions{
		BranchName: "feature-branch",
		CopyFiles:  false,
	}

	result, err := service.Create(options)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "feature-branch", result.BranchName)
	assert.False(t, result.WasCreated)
	assert.DirExists(t, result.WorktreePath)

	// Verify the worktree was created correctly
	worktrees, err := gitExec.Execute("worktree", "list")
	require.NoError(t, err)
	assert.Contains(t, worktrees, result.WorktreePath)
	assert.Contains(t, worktrees, "feature-branch")
}

func TestCreateCommand_Integration_NewBranchCreation(t *testing.T) {
	// Set up test repository
	testDir := setupTestRepository(t)
	defer testutils.CleanupTestDirectory(t, testDir)

	// Change to test directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.Chdir(testDir))

	// Initialize Grove configuration
	cfg := &config.Config{
		Worktree: config.WorktreeConfig{
			BasePath: filepath.Join(testDir, "worktrees"),
		},
	}

	// Set up create service components
	gitExec := git.NewGitExecutor()
	branchResolver := create.NewBranchResolver(gitExec)
	pathGenerator := create.NewPathGenerator(cfg)
	worktreeCreator := create.NewWorktreeCreator(gitExec)
	fileManager := create.NewFileManager(gitExec)
	
	service := create.NewCreateService(branchResolver, pathGenerator, worktreeCreator, fileManager)

	// Test creating worktree for new branch
	options := &create.CreateOptions{
		BranchName:   "new-feature",
		CreateBranch: true,
		BaseBranch:   "main",
		CopyFiles:    false,
	}

	result, err := service.Create(options)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "new-feature", result.BranchName)
	assert.True(t, result.WasCreated)
	assert.DirExists(t, result.WorktreePath)

	// Verify the branch and worktree were created correctly
	branches, err := gitExec.Execute("branch", "--list", "new-feature")
	require.NoError(t, err)
	assert.Contains(t, branches, "new-feature")

	worktrees, err := gitExec.Execute("worktree", "list")
	require.NoError(t, err)
	assert.Contains(t, worktrees, result.WorktreePath)
	assert.Contains(t, worktrees, "new-feature")
}

func TestCreateCommand_Integration_FileCopying(t *testing.T) {
	// Set up test repository with files to copy
	testDir := setupTestRepositoryWithFiles(t)
	defer testutils.CleanupTestDirectory(t, testDir)

	// Change to test directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.Chdir(testDir))

	// Initialize Grove configuration with file copying
	cfg := &config.Config{
		Worktree: config.WorktreeConfig{
			BasePath: filepath.Join(testDir, "worktrees"),
			CopyFiles: config.CopyFilesConfig{
				Patterns:         []string{".env*", ".vscode/"},
				SourceWorktree:   "main",
				OnConflict:       "skip",
			},
		},
	}

	// Set up create service components
	gitExec := git.NewGitExecutor()
	branchResolver := create.NewBranchResolver(gitExec)
	pathGenerator := create.NewPathGenerator(cfg)
	worktreeCreator := create.NewWorktreeCreator(gitExec)
	fileManager := create.NewFileManager(gitExec)
	
	service := create.NewCreateService(branchResolver, pathGenerator, worktreeCreator, fileManager)

	// Create a feature branch first
	_, err = gitExec.Execute("checkout", "-b", "feature-with-files")
	require.NoError(t, err)
	_, err = gitExec.Execute("checkout", "main")
	require.NoError(t, err)

	// Test creating worktree with file copying
	options := &create.CreateOptions{
		BranchName: "feature-with-files",
		CopyFiles:  true,
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

	// Verify excluded files were not copied
	readmeFile := filepath.Join(result.WorktreePath, "README.md")
	assert.NoFileExists(t, readmeFile)

	// Verify file contents match
	expectedEnv := "DATABASE_URL=postgres://localhost"
	actualEnv, err := os.ReadFile(envFile)
	require.NoError(t, err)
	assert.Equal(t, expectedEnv, string(actualEnv))
}

func TestCreateCommand_Integration_PathGeneration(t *testing.T) {
	// Set up test repository
	testDir := setupTestRepository(t)
	defer testutils.CleanupTestDirectory(t, testDir)

	// Change to test directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.Chdir(testDir))

	// Initialize Grove configuration
	cfg := &config.Config{
		Worktree: config.WorktreeConfig{
			BasePath: filepath.Join(testDir, "worktrees"),
		},
	}

	// Set up create service components
	gitExec := git.NewGitExecutor()
	branchResolver := create.NewBranchResolver(gitExec)
	pathGenerator := create.NewPathGenerator(cfg)
	worktreeCreator := create.NewWorktreeCreator(gitExec)
	fileManager := create.NewFileManager(gitExec)
	
	service := create.NewCreateService(branchResolver, pathGenerator, worktreeCreator, fileManager)

	// Create a feature branch with special characters
	branchName := "feature/complex-branch-name"
	_, err = gitExec.Execute("checkout", "-b", branchName)
	require.NoError(t, err)
	_, err = gitExec.Execute("checkout", "main")
	require.NoError(t, err)

	// Test creating worktree with automatic path generation
	options := &create.CreateOptions{
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
	defer testutils.CleanupTestDirectory(t, testDir)

	// Change to test directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.Chdir(testDir))

	// Initialize Grove configuration
	cfg := &config.Config{
		Worktree: config.WorktreeConfig{
			BasePath: filepath.Join(testDir, "worktrees"),
		},
	}

	// Create a directory that would conflict
	conflictPath := filepath.Join(cfg.Worktree.BasePath, "feature-branch")
	require.NoError(t, os.MkdirAll(conflictPath, 0755))

	// Set up create service components
	gitExec := git.NewGitExecutor()
	branchResolver := create.NewBranchResolver(gitExec)
	pathGenerator := create.NewPathGenerator(cfg)
	worktreeCreator := create.NewWorktreeCreator(gitExec)
	fileManager := create.NewFileManager(gitExec)
	
	service := create.NewCreateService(branchResolver, pathGenerator, worktreeCreator, fileManager)

	// Create a feature branch
	_, err = gitExec.Execute("checkout", "-b", "feature-branch")
	require.NoError(t, err)
	_, err = gitExec.Execute("checkout", "main")
	require.NoError(t, err)

	// Test creating worktree with path collision
	options := &create.CreateOptions{
		BranchName: "feature-branch",
		CopyFiles:  false,
	}

	result, err := service.Create(options)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.DirExists(t, result.WorktreePath)

	// Verify a unique path was generated
	assert.NotEqual(t, conflictPath, result.WorktreePath)
	assert.Contains(t, result.WorktreePath, "feature-branch")
	
	// Should contain a suffix like -2, -3, etc.
	basename := filepath.Base(result.WorktreePath)
	assert.True(t, strings.HasPrefix(basename, "feature-branch"))
	assert.NotEqual(t, "feature-branch", basename)
}

func TestCreateCommand_Integration_ConfigurationIntegration(t *testing.T) {
	// Set up test repository
	testDir := setupTestRepository(t)
	defer testutils.CleanupTestDirectory(t, testDir)

	// Change to test directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.Chdir(testDir))

	// Initialize Grove configuration with custom settings
	customBasePath := filepath.Join(testDir, "custom-worktrees")
	cfg := &config.Config{
		Worktree: config.WorktreeConfig{
			BasePath:    customBasePath,
			NamingStyle: "branch",
			CopyFiles: config.CopyFilesConfig{
				Patterns:   []string{".env", ".config/"},
				OnConflict: "overwrite",
			},
		},
		Create: config.CreateConfig{
			DefaultBaseBranch:    "main",
			PromptForNewBranch:   false,
			AutoCreateParents:    true,
		},
	}

	// Set up create service components
	gitExec := git.NewGitExecutor()
	branchResolver := create.NewBranchResolver(gitExec)
	pathGenerator := create.NewPathGenerator(cfg)
	worktreeCreator := create.NewWorktreeCreator(gitExec)
	fileManager := create.NewFileManager(gitExec)
	
	service := create.NewCreateService(branchResolver, pathGenerator, worktreeCreator, fileManager)

	// Test creating worktree with configuration integration
	options := &create.CreateOptions{
		BranchName:   "config-test",
		CreateBranch: true,
		CopyFiles:    false,
	}

	result, err := service.Create(options)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.DirExists(t, result.WorktreePath)

	// Verify custom base path was used
	assert.Contains(t, result.WorktreePath, customBasePath)
	
	// Verify branch was created with default base
	branches, err := gitExec.Execute("branch", "--list", "config-test")
	require.NoError(t, err)
	assert.Contains(t, branches, "config-test")
}

func TestCreateCommand_Integration_ErrorHandling(t *testing.T) {
	// Set up test repository
	testDir := setupTestRepository(t)
	defer testutils.CleanupTestDirectory(t, testDir)

	// Change to test directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.Chdir(testDir))

	// Initialize Grove configuration
	cfg := &config.Config{
		Worktree: config.WorktreeConfig{
			BasePath: filepath.Join(testDir, "worktrees"),
		},
	}

	// Set up create service components
	gitExec := git.NewGitExecutor()
	branchResolver := create.NewBranchResolver(gitExec)
	pathGenerator := create.NewPathGenerator(cfg)
	worktreeCreator := create.NewWorktreeCreator(gitExec)
	fileManager := create.NewFileManager(gitExec)
	
	service := create.NewCreateService(branchResolver, pathGenerator, worktreeCreator, fileManager)

	// Test creating worktree for nonexistent branch without new flag
	options := &create.CreateOptions{
		BranchName:   "nonexistent-branch",
		CreateBranch: false,
		CopyFiles:    false,
	}

	result, err := service.Create(options)

	require.Error(t, err)
	assert.Nil(t, result)
	
	// Should be a Grove error with appropriate code
	assert.Contains(t, err.Error(), "branch not found")
}

func TestCreateCommand_Integration_PerformanceRequirements(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Set up test repository
	testDir := setupTestRepository(t)
	defer testutils.CleanupTestDirectory(t, testDir)

	// Change to test directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.Chdir(testDir))

	// Initialize Grove configuration
	cfg := &config.Config{
		Worktree: config.WorktreeConfig{
			BasePath: filepath.Join(testDir, "worktrees"),
		},
	}

	// Set up create service components
	gitExec := git.NewGitExecutor()
	branchResolver := create.NewBranchResolver(gitExec)
	pathGenerator := create.NewPathGenerator(cfg)
	worktreeCreator := create.NewWorktreeCreator(gitExec)
	fileManager := create.NewFileManager(gitExec)
	
	service := create.NewCreateService(branchResolver, pathGenerator, worktreeCreator, fileManager)

	// Create a feature branch
	_, err = gitExec.Execute("checkout", "-b", "performance-test")
	require.NoError(t, err)
	_, err = gitExec.Execute("checkout", "main")
	require.NoError(t, err)

	// Test performance requirements (< 5 seconds for local branches)
	start := testutils.StartTimer()

	options := &create.CreateOptions{
		BranchName: "performance-test",
		CopyFiles:  false,
	}

	result, err := service.Create(options)

	elapsed := testutils.StopTimer(start)

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
	gitExec := git.NewGitExecutor()
	
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
	require.NoError(t, os.WriteFile(initialFile, []byte("# Test Repository"), 0644))
	
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
		"src/main.go":          "package main\n\nfunc main() {}\n",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(testDir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))
	}

	// Add files to git (but don't track .env files in real scenario)
	gitExec := git.NewGitExecutor()
	_, err = gitExec.Execute("add", "src/")
	require.NoError(t, err)
	_, err = gitExec.Execute("commit", "-m", "Add source files")
	require.NoError(t, err)

	return testDir
}


