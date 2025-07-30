//go:build integration
// +build integration

package init

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunInitLocal(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()
	runner.Run(func() {
		tempDir := helper.CreateTempDir("grove-init-local-test")

		err := os.Chdir(tempDir)
		require.NoError(t, err)

		err = runInitLocal("")
		require.NoError(t, err)

		bareDir := filepath.Join(tempDir, ".bare")
		_, err = os.Stat(bareDir)
		require.NoError(t, err)

		gitFile := filepath.Join(tempDir, ".git")
		_, err = os.Stat(gitFile)
		require.NoError(t, err)

		err = runInitLocal("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repository already exists at")
	})
}

func TestRunInitLocalWithTargetDir(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	tempDir := helper.CreateTempDir("grove-init-local-target")
	targetDir := filepath.Join(tempDir, "new-repo")

	err := runInitLocal(targetDir)
	require.NoError(t, err)

	_, err = os.Stat(targetDir)
	require.NoError(t, err)

	bareDir := filepath.Join(targetDir, ".bare")
	_, err = os.Stat(bareDir)
	require.NoError(t, err)

	gitFile := filepath.Join(targetDir, ".git")
	_, err = os.Stat(gitFile)
	require.NoError(t, err)
}

func TestRunInitConvertWithExecutor(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()
	runner.Run(func() {
		tempDir := helper.CreateTempDir("grove-init-convert-test")

		err := os.Chdir(tempDir)
		require.NoError(t, err)

		mockExecutor := testutils.NewMockGitExecutor()

		err = runInitConvertWithExecutor(mockExecutor)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repository not found at")
	})
}

func TestRunInitConvertAlreadyGrove(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()
	runner.Run(func() {
		tempDir := helper.CreateTempDir("grove-init-convert-grove")

		err := os.Chdir(tempDir)
		require.NoError(t, err)

		gitFile := filepath.Join(tempDir, ".git")
		err = os.WriteFile(gitFile, []byte("gitdir: .bare"), 0o644)
		require.NoError(t, err)

		bareDir := filepath.Join(tempDir, ".bare")
		err = os.MkdirAll(bareDir, 0o755)
		require.NoError(t, err)

		mockExecutor := testutils.NewMockGitExecutor()

		err = runInitConvertWithExecutor(mockExecutor)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repository already exists at")
	})
}

func TestRunInitRouting(t *testing.T) {
	cmd := NewInitCmd()

	cmd.SetArgs([]string{"--convert", "arg"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot specify arguments when using --convert flag")

	cmd2 := NewInitCmd()
	cmd2.SetArgs([]string{"arg1", "arg2", "arg3"})
	err = cmd2.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many arguments")
}
