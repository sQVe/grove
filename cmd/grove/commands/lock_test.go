package commands

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/testutil"
	testgit "github.com/sqve/grove/internal/testutil/git"
	"github.com/sqve/grove/internal/workspace"
)

func TestNewLockCmd(t *testing.T) {
	cmd := NewLockCmd()

	if cmd.Use != "lock <worktree>..." {
		t.Errorf("expected Use 'lock <worktree>...', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description")
	}
	reasonFlag := cmd.Flags().Lookup("reason")
	if reasonFlag == nil {
		t.Fatal("expected --reason flag")
	}
	if reasonFlag.Shorthand != "" {
		t.Errorf("expected reason to have no shorthand (git convention), got %q", reasonFlag.Shorthand)
	}
}

func TestRunLock_NotInWorkspace(t *testing.T) {
	defer testutil.SaveCwd(t)()

	tmpDir := testutil.TempDir(t)
	testutil.Chdir(t, tmpDir)

	err := runLock([]string{"some-branch"}, "")
	if !errors.Is(err, workspace.ErrNotInWorkspace) {
		t.Errorf("expected ErrNotInWorkspace, got %v", err)
	}
}

func TestRunLock_WorktreeNotFound(t *testing.T) {
	defer testutil.SaveCwd(t)()

	ws := testgit.NewGroveWorkspace(t, "main")
	testutil.Chdir(t, ws.WorktreePath("main"))

	err := runLock([]string{"nonexistent"}, "")
	if err == nil {
		t.Error("expected error for non-existent branch")
	}
	if !strings.Contains(err.Error(), "worktree not found") {
		t.Errorf("expected 'worktree not found' error, got: %v", err)
	}
}

func TestRunLock_AlreadyLocked(t *testing.T) {
	defer testutil.SaveCwd(t)()

	ws := testgit.NewGroveWorkspace(t, "main", "feature")

	// Lock the feature worktree
	cmd := exec.Command("git", "worktree", "lock", "--reason", "existing lock", ws.WorktreePath("feature")) //nolint:gosec
	cmd.Dir = ws.BareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to lock worktree: %v", err)
	}

	testutil.Chdir(t, ws.WorktreePath("main"))

	err := runLock([]string{"feature"}, "new reason")
	if err == nil {
		t.Error("expected error for already locked worktree")
	}
	if !strings.Contains(err.Error(), "feature") {
		t.Errorf("expected error to mention 'feature', got: %v", err)
	}
}

func TestRunLock_AlreadyLockedHint(t *testing.T) {
	defer testutil.SaveCwd(t)()

	ws := testgit.NewGroveWorkspace(t, "main", "feature")

	cmd := exec.Command("git", "worktree", "lock", ws.WorktreePath("feature")) //nolint:gosec
	cmd.Dir = ws.BareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to lock worktree: %v", err)
	}

	testutil.Chdir(t, ws.WorktreePath("main"))

	var buf bytes.Buffer
	logger.SetOutput(&buf)
	defer logger.SetOutput(nil)
	logger.Init(true, false)

	err := runLock([]string{"feature"}, "")
	if err == nil {
		t.Error("expected error for already locked worktree")
	}

	output := buf.String()
	if !strings.Contains(output, "grove unlock") {
		t.Errorf("expected output to contain 'grove unlock' hint, got: %s", output)
	}
}

func TestRunLock_Success(t *testing.T) {
	defer testutil.SaveCwd(t)()

	ws := testgit.NewGroveWorkspace(t, "main", "feature")
	testutil.Chdir(t, ws.WorktreePath("main"))

	err := runLock([]string{"feature"}, "")
	if err != nil {
		t.Fatalf("runLock failed: %v", err)
	}

	if !git.IsWorktreeLocked(ws.WorktreePath("feature")) {
		t.Error("worktree should be locked after runLock")
	}
}

func TestRunLock_SuccessWithReason(t *testing.T) {
	defer testutil.SaveCwd(t)()

	ws := testgit.NewGroveWorkspace(t, "main", "feature")
	testutil.Chdir(t, ws.WorktreePath("main"))

	reason := "WIP - do not remove"
	err := runLock([]string{"feature"}, reason)
	if err != nil {
		t.Fatalf("runLock with reason failed: %v", err)
	}

	featurePath := ws.WorktreePath("feature")
	if !git.IsWorktreeLocked(featurePath) {
		t.Error("worktree should be locked after runLock")
	}
	gotReason := git.GetWorktreeLockReason(featurePath)
	if gotReason != reason {
		t.Errorf("expected lock reason %q, got %q", reason, gotReason)
	}
}

func TestRunLock_MultipleWorktrees(t *testing.T) {
	defer testutil.SaveCwd(t)()

	ws := testgit.NewGroveWorkspace(t, "main", "feature", "bugfix")
	testutil.Chdir(t, ws.WorktreePath("main"))

	err := runLock([]string{"feature", "bugfix"}, "")
	if err != nil {
		t.Fatalf("runLock failed: %v", err)
	}

	if !git.IsWorktreeLocked(ws.WorktreePath("feature")) {
		t.Error("feature worktree should be locked")
	}
	if !git.IsWorktreeLocked(ws.WorktreePath("bugfix")) {
		t.Error("bugfix worktree should be locked")
	}
}

func TestRunLock_MultipleWithReason(t *testing.T) {
	defer testutil.SaveCwd(t)()

	ws := testgit.NewGroveWorkspace(t, "main", "feature", "bugfix")
	testutil.Chdir(t, ws.WorktreePath("main"))

	reason := "WIP - shared reason"
	err := runLock([]string{"feature", "bugfix"}, reason)
	if err != nil {
		t.Fatalf("runLock failed: %v", err)
	}

	if git.GetWorktreeLockReason(ws.WorktreePath("feature")) != reason {
		t.Errorf("feature should have reason %q", reason)
	}
	if git.GetWorktreeLockReason(ws.WorktreePath("bugfix")) != reason {
		t.Errorf("bugfix should have reason %q", reason)
	}
}

func TestRunLock_MultipleDuplicateArgs(t *testing.T) {
	defer testutil.SaveCwd(t)()

	ws := testgit.NewGroveWorkspace(t, "main", "feature")
	testutil.Chdir(t, ws.WorktreePath("main"))

	// Pass same worktree twice - should deduplicate and succeed
	err := runLock([]string{"feature", "feature"}, "")
	if err != nil {
		t.Fatalf("expected success with duplicate args, got: %v", err)
	}

	if !git.IsWorktreeLocked(ws.WorktreePath("feature")) {
		t.Error("feature should be locked")
	}
}

func TestRunLock_DuplicateViaBranchAndDirName(t *testing.T) {
	defer testutil.SaveCwd(t)()

	ws := testgit.NewGroveWorkspace(t, "main")

	// Create worktree where directory name differs from branch name
	// Branch: feature/auth, Directory: feature-auth
	featurePath := ws.Dir + "/feature-auth"
	cmd := exec.Command("git", "worktree", "add", "-b", "feature/auth", featurePath) //nolint:gosec
	cmd.Dir = ws.BareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}
	testgit.CleanupWorktree(t, ws.BareDir, featurePath)

	testutil.Chdir(t, ws.WorktreePath("main"))

	// Pass both directory name AND branch name - should deduplicate by path
	err := runLock([]string{"feature-auth", "feature/auth"}, "")
	if err != nil {
		t.Fatalf("expected success with branch/dir duplicate, got: %v", err)
	}

	if !git.IsWorktreeLocked(featurePath) {
		t.Error("feature-auth should be locked")
	}
}

func TestRunLock_OutputShowsWorktreeLabel(t *testing.T) {
	defer testutil.SaveCwd(t)()

	ws := testgit.NewGroveWorkspace(t, "main")

	// Create worktree where directory name differs from branch name
	featurePath := ws.Dir + "/feat-auth"
	cmd := exec.Command("git", "worktree", "add", "-b", "feature/auth", featurePath) //nolint:gosec
	cmd.Dir = ws.BareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}
	testgit.CleanupWorktree(t, ws.BareDir, featurePath)

	testutil.Chdir(t, ws.WorktreePath("main"))

	var buf bytes.Buffer
	logger.SetOutput(&buf)
	defer logger.SetOutput(nil)
	logger.Init(true, false)

	err := runLock([]string{"feat-auth"}, "")
	if err != nil {
		t.Fatalf("runLock failed: %v", err)
	}

	output := buf.String()
	// Success message should show directory name as primary with branch in brackets
	if !strings.Contains(output, "feat-auth [feature/auth]") {
		t.Errorf("expected success output to show worktree label 'feat-auth [feature/auth]', got: %s", output)
	}
}

func TestCompleteLockArgs_MultipleArgs(t *testing.T) {
	defer testutil.SaveCwd(t)()

	ws := testgit.NewGroveWorkspace(t, "main", "feature", "bugfix")
	testutil.Chdir(t, ws.WorktreePath("main"))

	lockCmd := NewLockCmd()

	// First completion - should show all unlocked worktrees
	completions, _ := completeLockArgs(lockCmd, nil, "")
	if len(completions) != 3 {
		t.Errorf("expected 3 completions, got %d: %v", len(completions), completions)
	}

	// Second completion (feature already typed) - should exclude feature
	completions, _ = completeLockArgs(lockCmd, []string{"feature"}, "")
	if len(completions) != 2 {
		t.Errorf("expected 2 completions after 'feature', got %d: %v", len(completions), completions)
	}
	for _, c := range completions {
		if c == "feature" {
			t.Error("completions should not include already-typed 'feature'")
		}
	}
}

func TestCompleteLockArgs(t *testing.T) {
	defer testutil.SaveCwd(t)()

	ws := testgit.NewGroveWorkspace(t, "main", "feature", "bugfix")

	// Lock bugfix worktree
	cmd := exec.Command("git", "worktree", "lock", ws.WorktreePath("bugfix")) //nolint:gosec
	cmd.Dir = ws.BareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to lock bugfix worktree: %v", err)
	}

	testutil.Chdir(t, ws.WorktreePath("main"))

	lockCmd := NewLockCmd()
	completions, directive := completeLockArgs(lockCmd, nil, "")

	// Should include unlocked worktrees (main, feature)
	hasMain := false
	hasFeature := false
	for _, c := range completions {
		if c == "main" {
			hasMain = true
		}
		if c == "feature" {
			hasFeature = true
		}
	}
	if !hasMain {
		t.Error("completions should include unlocked 'main'")
	}
	if !hasFeature {
		t.Error("completions should include unlocked 'feature'")
	}

	// Should NOT include locked worktrees
	for _, c := range completions {
		if c == "bugfix" {
			t.Error("completions should not include locked 'bugfix'")
		}
	}

	// Should disable file completion
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}
}
