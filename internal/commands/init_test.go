//go:build !integration
// +build !integration

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
	mockExecutor := testutils.NewMockGitExecutor()
	mockExecutor.SetSafeRepositoryState()

	output, err := mockExecutor.Execute("status", "--porcelain=v1")
	require.NoError(t, err)
	assert.Empty(t, output)

	output, err = mockExecutor.Execute("stash", "list")
	require.NoError(t, err)
	assert.Empty(t, output)

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

	convertFlag := cmd.Flags().Lookup("convert")
	require.NotNil(t, convertFlag)
	assert.Equal(t, "false", convertFlag.DefValue)
	assert.Equal(t, "Convert existing traditional Git repository to Grove structure", convertFlag.Usage)
}

func TestValidateAndPrepareDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "grove-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test empty directory
	result, err := validateAndPrepareDirectory()
	require.NoError(t, err)
	assert.Equal(t, tempDir, result)

	// Test directory with hidden files (should pass)
	hiddenFile := filepath.Join(tempDir, ".hidden")
	err = os.WriteFile(hiddenFile, []byte("test"), 0o644)
	require.NoError(t, err)

	result, err = validateAndPrepareDirectory()
	require.NoError(t, err)
	assert.Equal(t, tempDir, result)

	// Test directory with non-hidden files (should fail)
	visibleFile := filepath.Join(tempDir, "visible.txt")
	err = os.WriteFile(visibleFile, []byte("test"), 0o644)
	require.NoError(t, err)

	_, err = validateAndPrepareDirectory()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not empty")
}

func TestPrintSuccessMessage(t *testing.T) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		done <- buf.String()
	}()

	printSuccessMessage("/test/dir", "/test/dir/.bare")
	_ = w.Close()
	os.Stdout = oldStdout

	output := <-done
	assert.Contains(t, output, "Successfully cloned and configured repository in /test/dir")
	assert.Contains(t, output, "Git objects stored in: /test/dir/.bare")
	assert.Contains(t, output, "Next steps:")
	assert.Contains(t, output, "grove create <branch-name>")
}

func TestRunInitRouting(t *testing.T) {
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
