package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCommandConvertWithMockExecutor(t *testing.T) {
	// Test that the mock executor is properly called for safety checks
	mockExecutor := testutils.NewMockGitExecutor()
	mockExecutor.SetSafeRepositoryState()

	// Test that the mock executor responds to safety check commands
	output, err := mockExecutor.Execute("status", "--porcelain=v1")
	require.NoError(t, err)
	assert.Empty(t, output)

	output, err = mockExecutor.Execute("stash", "list")
	require.NoError(t, err)
	assert.Empty(t, output)

	// Test unhandled command returns error
	_, err = mockExecutor.Execute("unhandled", "command")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mock: unhandled git command")
}

func TestInitCommandConvertCannotSpecifyArgs(t *testing.T) {
	cmd := NewInitCmd()
	cmd.SetArgs([]string{"--convert", "some-arg"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot specify arguments when using --convert flag")
}

func TestInitCommandTooManyArgs(t *testing.T) {
	cmd := NewInitCmd()
	cmd.SetArgs([]string{"arg1", "arg2", "arg3"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many arguments")
}

func TestInitCommandUsage(t *testing.T) {
	cmd := NewInitCmd()
	assert.Equal(t, "init [directory|remote-url]", cmd.Use)
	assert.Equal(t, "Initialize or clone a Git repository optimized for worktrees", cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Test that the command has the convert flag
	convertFlag := cmd.Flags().Lookup("convert")
	require.NotNil(t, convertFlag)
	assert.Equal(t, "false", convertFlag.DefValue)
	assert.Equal(t, "Convert existing traditional Git repository to Grove structure", convertFlag.Usage)
}

func TestValidateAndPrepareDirectory(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "grove-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to the temporary directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test empty directory
	result, err := validateAndPrepareDirectory()
	require.NoError(t, err)
	assert.Equal(t, tempDir, result)

	// Test directory with hidden files (should pass)
	hiddenFile := filepath.Join(tempDir, ".hidden")
	err = os.WriteFile(hiddenFile, []byte("test"), 0644)
	require.NoError(t, err)

	result, err = validateAndPrepareDirectory()
	require.NoError(t, err)
	assert.Equal(t, tempDir, result)

	// Test directory with non-hidden files (should fail)
	visibleFile := filepath.Join(tempDir, "visible.txt")
	err = os.WriteFile(visibleFile, []byte("test"), 0644)
	require.NoError(t, err)

	_, err = validateAndPrepareDirectory()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not empty")
}

func TestPrintSuccessMessage(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		buf.ReadFrom(r)
		done <- buf.String()
	}()

	printSuccessMessage("/test/dir", "/test/dir/.bare")
	w.Close()
	os.Stdout = oldStdout

	output := <-done
	assert.Contains(t, output, "Successfully cloned and configured repository in /test/dir")
	assert.Contains(t, output, "Git objects stored in: /test/dir/.bare")
	assert.Contains(t, output, "Next steps:")
	assert.Contains(t, output, "grove create <branch-name>")
}

// Test runInitLocal function
func TestRunInitLocal(t *testing.T) {
	// Test with empty target directory (should use current directory)
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "grove-init-local-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	// Test successful local initialization
	err = runInitLocal("")
	require.NoError(t, err)

	// Verify .bare directory was created
	bareDir := filepath.Join(tempDir, ".bare")
	_, err = os.Stat(bareDir)
	require.NoError(t, err)

	// Verify .git file was created
	gitFile := filepath.Join(tempDir, ".git")
	_, err = os.Stat(gitFile)
	require.NoError(t, err)

	// Test with existing .git (should fail)
	err = runInitLocal("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already contains a .git file or directory")
}

func TestRunInitLocalWithTargetDir(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "grove-init-local-target-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	targetDir := filepath.Join(tempDir, "new-repo")

	// Test successful local initialization with target directory
	err = runInitLocal(targetDir)
	require.NoError(t, err)

	// Verify target directory was created
	_, err = os.Stat(targetDir)
	require.NoError(t, err)

	// Verify .bare directory was created
	bareDir := filepath.Join(targetDir, ".bare")
	_, err = os.Stat(bareDir)
	require.NoError(t, err)

	// Verify .git file was created
	gitFile := filepath.Join(targetDir, ".git")
	_, err = os.Stat(gitFile)
	require.NoError(t, err)
}

// Test runInitConvertWithExecutor function - error cases
func TestRunInitConvertWithExecutor(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "grove-init-convert-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	mockExecutor := testutils.NewMockGitExecutor()

	// Test conversion of non-traditional repo (should fail)
	err = runInitConvertWithExecutor(mockExecutor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not contain a traditional Git repository")
}

func TestRunInitConvertAlreadyGrove(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "grove-init-convert-grove-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create a mock .git file and .bare directory to simulate a grove repo
	gitFile := filepath.Join(tempDir, ".git")
	err = os.WriteFile(gitFile, []byte("gitdir: .bare"), 0644)
	require.NoError(t, err)

	bareDir := filepath.Join(tempDir, ".bare")
	err = os.MkdirAll(bareDir, 0755)
	require.NoError(t, err)

	mockExecutor := testutils.NewMockGitExecutor()

	// Test conversion of already grove repo (should fail)
	err = runInitConvertWithExecutor(mockExecutor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is already a Grove repository")
}

// Test that runInit properly routes to the right sub-function
func TestRunInitRouting(t *testing.T) {
	// Create a mock command to test argument validation
	cmd := NewInitCmd()

	// Test convert flag validation
	cmd.SetArgs([]string{"--convert", "arg"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot specify arguments when using --convert flag")

	// Test too many args validation - need to create a fresh command instance
	cmd2 := NewInitCmd()
	cmd2.SetArgs([]string{"arg1", "arg2", "arg3"})
	err = cmd2.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many arguments")
}
