package commands

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/testutil"
	testgit "github.com/sqve/grove/internal/testutil/git"
	"github.com/sqve/grove/internal/workspace"
)

func TestAddHerdr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell herdr stub is not executable on Windows")
	}

	for _, scenario := range []struct {
		name      string
		arguments []string
		hook      string
		exitCode  string
		missing   bool
		wantCall  bool
		wantError string
	}{
		{name: "branch", arguments: []string{"feature", "--herdr"}, wantCall: true},
		{name: "detached", arguments: []string{"main", "--detach", "--herdr"}, wantCall: true},
		{name: "pull request", arguments: []string{"https://github.com/owner/repo/pull/42", "--herdr"}, wantCall: true},
		{name: "without flag", arguments: []string{"feature"}},
		{name: "failed hook", arguments: []string{"feature", "--herdr"}, hook: "exit 9", wantError: "hook failed"},
		{name: "missing binary", arguments: []string{"feature", "--herdr"}, missing: true, wantError: "PATH"},
		{name: "failed handoff", arguments: []string{"feature", "--herdr"}, exitCode: "7", wantCall: true, wantError: "exit status 7"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			repository := testgit.NewTestRepo(t)
			repository.CreateBranch("pr-feature")
			workspaceRoot := filepath.Join(repository.TempDir, "workspace with spaces")
			bareDir := filepath.Join(workspaceRoot, ".bare")
			repository.RunOutput("clone", "--bare", repository.Path, bareDir)
			mainPath := filepath.Join(workspaceRoot, "main")
			repository.RunOutput("-C", bareDir, "worktree", "add", mainPath, "main")
			t.Chdir(mainPath)

			hook := "printf ready > prepared"
			if scenario.hook != "" {
				hook = scenario.hook
			}
			testutil.WriteFile(t, filepath.Join(mainPath, ".grove.toml"), "[hooks]\nadd = [\""+hook+"\"]\n")

			binaryDir := t.TempDir()
			gitPath, err := exec.LookPath("git")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(gitPath, filepath.Join(binaryDir, "git")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("/bin/sh", filepath.Join(binaryDir, "sh")); err != nil {
				t.Fatal(err)
			}

			callPath := filepath.Join(t.TempDir(), "calls")
			t.Setenv("HERDR_CALLS", callPath)
			t.Setenv("HERDR_EXIT", scenario.exitCode)
			if !scenario.missing {
				testutil.WriteFileMode(t, filepath.Join(binaryDir, "herdr"), `#!/bin/sh
[ -f "$6/prepared" ] || exit 8
[ -e "$4/.grove-worktree.lock" ] && exit 7
printf '%s\n' "$@" >> "$HERDR_CALLS"
printf 'herdr diagnostic\n' >&2
exit "${HERDR_EXIT:-0}"
`, fs.FileExec)
			}
			testutil.WriteFileMode(t, filepath.Join(binaryDir, "gh"), `#!/bin/sh
if [ "$1" = auth ]; then exit 0; fi
printf '%s\n' '{"headRefName":"pr-feature","headRepository":{"name":"repo"},"headRepositoryOwner":{"login":"owner"}}'
`, fs.FileExec)
			t.Setenv("PATH", binaryDir)

			stderr, err := os.CreateTemp(t.TempDir(), "stderr")
			if err != nil {
				t.Fatal(err)
			}
			originalStderr := os.Stderr
			os.Stderr = stderr
			t.Cleanup(func() {
				os.Stderr = originalStderr
				_ = stderr.Close()
			})

			command := NewAddCmd()
			command.SetArgs(append(scenario.arguments, "--name", "prepared tree", "--no-fetch"))
			err = command.Execute()
			worktreePath := filepath.Join(workspaceRoot, "prepared tree")
			if scenario.wantError == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), scenario.wantError) {
				t.Fatalf("error = %v, want %q", err, scenario.wantError)
			}
			if scenario.missing || scenario.exitCode != "" {
				if !strings.Contains(err.Error(), worktreePath) || errors.Unwrap(err) == nil {
					t.Fatalf("expected wrapped error naming prepared path, got %v", err)
				}
			}
			if scenario.missing {
				if !errors.Is(err, exec.ErrNotFound) || !strings.Contains(err.Error(), "ensure herdr is installed and on PATH") {
					t.Fatalf("expected missing binary with install hint, got %v", err)
				}
			}
			if scenario.exitCode != "" {
				var exitError *exec.ExitError
				if !errors.As(err, &exitError) || !strings.Contains(err.Error(), "Herdr exited with an error") || !strings.Contains(err.Error(), "output above") {
					t.Errorf("expected failed exit with output hint, got %v", err)
				}
				if strings.Contains(err.Error(), "installed") || strings.Contains(err.Error(), "PATH") {
					t.Errorf("unexpected install hint after Herdr ran: %v", err)
				}
			}

			if _, err := os.Stat(filepath.Join(worktreePath, ".git")); err != nil {
				t.Fatalf("worktree must remain intact: %v", err)
			}

			calls, readError := os.ReadFile(callPath) //nolint:gosec // Test-owned temporary path.
			if scenario.wantCall {
				want := "worktree\nopen\n--cwd\n" + workspaceRoot + "\n--path\n" + worktreePath + "\n--focus\n"
				if readError != nil || string(calls) != want {
					t.Fatalf("calls = %q, error = %v, want %q", calls, readError, want)
				}
				diagnostic, err := os.ReadFile(stderr.Name())
				if err != nil || !strings.Contains(string(diagnostic), "herdr diagnostic") {
					t.Fatalf("missing inherited stderr: %q (%v)", diagnostic, err)
				}
			} else if !os.IsNotExist(readError) {
				t.Fatalf("unexpected herdr call: %q (%v)", calls, readError)
			}

			if scenario.wantCall && scenario.wantError == "" {
				worktreesBefore := repository.RunOutput("-C", bareDir, "worktree", "list", "--porcelain")
				branchesBefore := repository.RunOutput("-C", bareDir, "show-ref", "--heads")
				testutil.WriteFile(t, filepath.Join(mainPath, "preserved"), "new source file")
				testutil.WriteFile(t, filepath.Join(mainPath, "linked", "file"), "new source directory")
				testutil.WriteFile(t, filepath.Join(mainPath, ".grove.toml"), "[preserve]\npatterns = [\"preserved\"]\n[link]\npatterns = [\"linked\"]\n[hooks]\nadd = [\"exit 9\"]\n")

				rerunName := "unused directory"
				if scenario.name == "detached" {
					rerunName = "prepared tree"
				}
				command = NewAddCmd()
				command.SetArgs(append(scenario.arguments, "--name", rerunName, "--no-fetch"))
				if err := command.Execute(); err != nil {
					t.Fatalf("handoff-only rerun: %v", err)
				}

				rerunCalls, err := os.ReadFile(callPath) //nolint:gosec // Test-owned temporary path.
				if err != nil || string(rerunCalls) != string(calls)+string(calls) {
					t.Fatalf("rerun must use identical argv: %q (%v)", rerunCalls, err)
				}
				worktreesAfter := repository.RunOutput("-C", bareDir, "worktree", "list", "--porcelain")
				if worktreesAfter != worktreesBefore || strings.Count(worktreesAfter, "worktree "+worktreePath+"\n") != 1 {
					t.Fatalf("rerun changed worktrees: %s", worktreesAfter)
				}
				if branchesAfter := repository.RunOutput("-C", bareDir, "show-ref", "--heads"); branchesAfter != branchesBefore {
					t.Fatalf("rerun changed branches: %s", branchesAfter)
				}
				for _, absentPath := range []string{filepath.Join(workspaceRoot, "unused directory"), filepath.Join(worktreePath, "preserved"), filepath.Join(worktreePath, "linked"), filepath.Join(workspaceRoot, ".grove-worktree.lock")} {
					if _, err := os.Lstat(absentPath); !os.IsNotExist(err) {
						t.Fatalf("handoff-only rerun created %s (%v)", absentPath, err)
					}
				}
			}
		})
	}
}

func TestAddHerdrDetachedRerun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell herdr stub is not executable on Windows")
	}

	for _, scenario := range []struct {
		name      string
		ref       string
		parkAtTag bool
		wantCall  bool
		wantError string
	}{
		{name: "annotated tag resolves to its commit", ref: "v1.0.0", wantCall: true},
		{name: "refuses a worktree parked at another commit", ref: "main", parkAtTag: true, wantError: "not \"main\""},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			repository := testgit.NewTestRepo(t)
			repository.RunOutput("-C", repository.Path, "tag", "-a", "v1.0.0", "-m", "release")
			// Move main past the tag, so "main" and "v1.0.0" name different commits.
			repository.WriteFile("moved", "after the tag")
			repository.Add("moved")
			repository.Commit("chore: move main past the tag")
			workspaceRoot := filepath.Join(repository.TempDir, "workspace")
			bareDir := filepath.Join(workspaceRoot, ".bare")
			repository.RunOutput("clone", "--bare", repository.Path, bareDir)
			repository.RunOutput("-C", bareDir, "fetch", "origin", "refs/tags/*:refs/tags/*")
			mainPath := filepath.Join(workspaceRoot, "main")
			repository.RunOutput("-C", bareDir, "worktree", "add", mainPath, "main")
			t.Chdir(mainPath)

			binaryDir := t.TempDir()
			gitPath, err := exec.LookPath("git")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(gitPath, filepath.Join(binaryDir, "git")); err != nil {
				t.Fatal(err)
			}
			callPath := filepath.Join(t.TempDir(), "calls")
			t.Setenv("HERDR_CALLS", callPath)
			testutil.WriteFileMode(t, filepath.Join(binaryDir, "herdr"), `#!/bin/sh
printf '%s\n' "$@" >> "$HERDR_CALLS"
`, fs.FileExec)
			t.Setenv("PATH", binaryDir)

			worktreePath := filepath.Join(workspaceRoot, "probe")
			command := NewAddCmd()
			command.SetArgs([]string{scenario.ref, "--detach", "--herdr", "--name", "probe", "--no-fetch"})
			if err := command.Execute(); err != nil {
				t.Fatalf("first run: %v", err)
			}
			if err := os.Remove(callPath); err != nil {
				t.Fatal(err)
			}

			// Park the existing worktree on a commit the ref does not name.
			if scenario.parkAtTag {
				repository.RunOutput("-C", worktreePath, "checkout", "--detach", "v1.0.0")
			}

			command = NewAddCmd()
			command.SetArgs([]string{scenario.ref, "--detach", "--herdr", "--name", "probe", "--no-fetch"})
			err = command.Execute()

			calls, readError := os.ReadFile(callPath) //nolint:gosec // Test-owned temporary path.
			if scenario.wantCall {
				if err != nil {
					t.Fatalf("rerun must hand off: %v", err)
				}
				want := "worktree\nopen\n--cwd\n" + workspaceRoot + "\n--path\n" + worktreePath + "\n--focus\n"
				if readError != nil || string(calls) != want {
					t.Fatalf("calls = %q, error = %v, want %q", calls, readError, want)
				}
				return
			}

			if err == nil || !strings.Contains(err.Error(), scenario.wantError) {
				t.Fatalf("error = %v, want %q", err, scenario.wantError)
			}
			if !os.IsNotExist(readError) {
				t.Fatalf("mismatched HEAD must not reach Herdr: %q", calls)
			}
		})
	}
}

// testgit builds on testutil.TempDir, which resolves symlinks, so a symlinked
// workspace root has to be built by hand to be exercised at all.
func TestAddHerdrSymlinkedRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell herdr stub is not executable on Windows")
	}

	repository := testgit.NewTestRepo(t)
	realRoot := filepath.Join(repository.TempDir, "real")
	bareDir := filepath.Join(realRoot, ".bare")
	repository.RunOutput("clone", "--bare", repository.Path, bareDir)
	mainPath := filepath.Join(realRoot, "main")
	repository.RunOutput("-C", bareDir, "worktree", "add", mainPath, "main")

	linkRoot := filepath.Join(repository.TempDir, "link")
	if err := os.Symlink(realRoot, linkRoot); err != nil {
		t.Fatal(err)
	}

	// os.Getwd prefers $PWD when it names the same directory, which is how the
	// unresolved spelling reaches Grove in a real shell.
	linkMain := filepath.Join(linkRoot, "main")
	t.Chdir(linkMain)
	t.Setenv("PWD", linkMain)

	binaryDir := t.TempDir()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(gitPath, filepath.Join(binaryDir, "git")); err != nil {
		t.Fatal(err)
	}
	callPath := filepath.Join(t.TempDir(), "calls")
	t.Setenv("HERDR_CALLS", callPath)
	testutil.WriteFileMode(t, filepath.Join(binaryDir, "herdr"), `#!/bin/sh
printf '%s\n' "$@" >> "$HERDR_CALLS"
`, fs.FileExec)
	t.Setenv("PATH", binaryDir)

	command := NewAddCmd()
	command.SetArgs([]string{"main", "--detach", "--herdr", "--name", "probe", "--no-fetch"})
	if err := command.Execute(); err != nil {
		t.Fatalf("first run: %v", err)
	}

	command = NewAddCmd()
	command.SetArgs([]string{"main", "--detach", "--herdr", "--name", "probe", "--no-fetch"})
	if err := command.Execute(); err != nil {
		t.Fatalf("rerun through the symlinked root must hand off, got: %v", err)
	}

	calls, err := os.ReadFile(callPath) //nolint:gosec // Test-owned temporary path.
	if err != nil {
		t.Fatal(err)
	}

	// Both runs must send one identity, in the resolved namespace, or Herdr
	// treats them as two workspaces.
	want := "worktree\nopen\n--cwd\n" + realRoot + "\n--path\n" + filepath.Join(realRoot, "probe") + "\n--focus\n"
	if string(calls) != want+want {
		t.Fatalf("calls = %q, want two identical resolved calls %q", calls, want)
	}
}

func TestAddHerdrOpensDirtyPRWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell herdr stub is not executable on Windows")
	}

	repository := testgit.NewTestRepo(t)
	repository.CreateBranch("pr-feature")
	workspaceRoot := filepath.Join(repository.TempDir, "workspace")
	bareDir := filepath.Join(workspaceRoot, ".bare")
	repository.RunOutput("clone", "--bare", repository.Path, bareDir)
	mainPath := filepath.Join(workspaceRoot, "main")
	repository.RunOutput("-C", bareDir, "worktree", "add", mainPath, "main")
	t.Chdir(mainPath)

	binaryDir := t.TempDir()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(gitPath, filepath.Join(binaryDir, "git")); err != nil {
		t.Fatal(err)
	}
	callPath := filepath.Join(t.TempDir(), "calls")
	t.Setenv("HERDR_CALLS", callPath)
	testutil.WriteFileMode(t, filepath.Join(binaryDir, "herdr"), `#!/bin/sh
printf '%s\n' "$@" >> "$HERDR_CALLS"
`, fs.FileExec)
	testutil.WriteFileMode(t, filepath.Join(binaryDir, "gh"), `#!/bin/sh
if [ "$1" = auth ]; then exit 0; fi
printf '%s\n' '{"headRefName":"pr-feature","headRepository":{"name":"repo"},"headRepositoryOwner":{"login":"owner"}}'
`, fs.FileExec)
	t.Setenv("PATH", binaryDir)

	arguments := []string{"https://github.com/owner/repo/pull/42", "--herdr"}
	command := NewAddCmd()
	command.SetArgs(arguments)
	if err := command.Execute(); err != nil {
		t.Fatalf("first run: %v", err)
	}
	worktreePath := filepath.Join(workspaceRoot, "pr-42")
	if err := os.Remove(callPath); err != nil {
		t.Fatal(err)
	}

	// A worktree you are working in is dirty by definition; the refresh refuses
	// it, and that must not also refuse to open it. test.txt is the tracked file
	// NewTestRepo commits, so editing it is what trips the refusal.
	testutil.WriteFile(t, filepath.Join(worktreePath, "test.txt"), "edited in the worktree")

	stderr, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	originalStderr := os.Stderr
	os.Stderr = stderr
	t.Cleanup(func() {
		os.Stderr = originalStderr
		_ = stderr.Close()
	})

	command = NewAddCmd()
	command.SetArgs(arguments)
	if err := command.Execute(); err != nil {
		t.Fatalf("dirty worktree must still open in Herdr, got: %v", err)
	}

	calls, err := os.ReadFile(callPath) //nolint:gosec // Test-owned temporary path.
	if err != nil {
		t.Fatalf("dirty worktree never reached Herdr: %v", err)
	}
	want := "worktree\nopen\n--cwd\n" + workspaceRoot + "\n--path\n" + worktreePath + "\n--focus\n"
	if string(calls) != want {
		t.Fatalf("calls = %q, want %q", calls, want)
	}

	// The handoff must open the work in progress, never discard it.
	edited, err := os.ReadFile(filepath.Join(worktreePath, "test.txt")) //nolint:gosec // Test-owned temporary path.
	if err != nil || string(edited) != "edited in the worktree" {
		t.Fatalf("handoff destroyed the uncommitted edit: %q (%v)", edited, err)
	}

	// Pin the reason, so a fetch failure taking the same path is not mistaken
	// for the dirty-worktree branch this test is named for.
	warning, err := os.ReadFile(stderr.Name()) //nolint:gosec // Test-owned temporary path.
	if err != nil || !strings.Contains(string(warning), "uncommitted changes") {
		t.Fatalf("expected the uncommitted-changes warning, got %q (%v)", warning, err)
	}
}

// The PR number is recorded even when the sync that follows refuses, or a
// worktree adopted by a PR re-run never shows its PR in grove list.
func TestAddHerdrRecordsPROnDirtyWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell herdr stub is not executable on Windows")
	}

	repository := testgit.NewTestRepo(t)
	repository.CreateBranch("pr-feature")
	workspaceRoot := filepath.Join(repository.TempDir, "workspace")
	bareDir := filepath.Join(workspaceRoot, ".bare")
	repository.RunOutput("clone", "--bare", repository.Path, bareDir)
	mainPath := filepath.Join(workspaceRoot, "main")
	repository.RunOutput("-C", bareDir, "worktree", "add", mainPath, "main")
	t.Chdir(mainPath)

	binaryDir := t.TempDir()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(gitPath, filepath.Join(binaryDir, "git")); err != nil {
		t.Fatal(err)
	}
	callPath := filepath.Join(t.TempDir(), "calls")
	t.Setenv("HERDR_CALLS", callPath)
	testutil.WriteFileMode(t, filepath.Join(binaryDir, "herdr"), `#!/bin/sh
printf '%s\n' "$@" >> "$HERDR_CALLS"
`, fs.FileExec)
	testutil.WriteFileMode(t, filepath.Join(binaryDir, "gh"), `#!/bin/sh
if [ "$1" = auth ]; then exit 0; fi
printf '%s\n' '{"headRefName":"pr-feature","headRepository":{"name":"repo"},"headRepositoryOwner":{"login":"owner"}}'
`, fs.FileExec)
	t.Setenv("PATH", binaryDir)

	// Created as a plain branch worktree, so nothing has recorded a PR yet.
	command := NewAddCmd()
	command.SetArgs([]string{"pr-feature", "--no-fetch"})
	if err := command.Execute(); err != nil {
		t.Fatalf("branch add: %v", err)
	}
	worktreePath := filepath.Join(workspaceRoot, "pr-feature")
	testutil.WriteFile(t, filepath.Join(worktreePath, "test.txt"), "edited in the worktree")

	command = NewAddCmd()
	command.SetArgs([]string{"https://github.com/owner/repo/pull/42", "--herdr"})
	if err := command.Execute(); err != nil {
		t.Fatalf("dirty PR re-run must open, got: %v", err)
	}

	configs, err := git.GetBranchConfigs(bareDir, "grovePr")
	if err != nil {
		t.Fatal(err)
	}
	if configs["pr-feature"] != "42" {
		t.Fatalf("grovePr = %q, want \"42\"", configs["pr-feature"])
	}
}

// A fork PR worktree is checked out from the fork's remote-tracking ref, so it
// is detached and has no branch to match on. Its path is the identity.
func TestAddHerdrRerunsForkPRWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell herdr stub is not executable on Windows")
	}

	repository := testgit.NewTestRepo(t)
	repository.CreateBranch("fork-feature")
	workspaceRoot := filepath.Join(repository.TempDir, "workspace")
	bareDir := filepath.Join(workspaceRoot, ".bare")
	repository.RunOutput("clone", "--bare", repository.Path, bareDir)
	mainPath := filepath.Join(workspaceRoot, "main")
	repository.RunOutput("-C", bareDir, "worktree", "add", mainPath, "main")
	t.Chdir(mainPath)

	binaryDir := t.TempDir()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(gitPath, filepath.Join(binaryDir, "git")); err != nil {
		t.Fatal(err)
	}
	callPath := filepath.Join(t.TempDir(), "calls")
	t.Setenv("HERDR_CALLS", callPath)
	testutil.WriteFileMode(t, filepath.Join(binaryDir, "herdr"), `#!/bin/sh
printf '%s\n' "$@" >> "$HERDR_CALLS"
`, fs.FileExec)
	// headRepositoryOwner differs from the URL's owner, which is what makes it a
	// fork; repo view hands back the local repo standing in for the fork.
	testutil.WriteFileMode(t, filepath.Join(binaryDir, "gh"), `#!/bin/sh
if [ "$1" = auth ]; then exit 0; fi
if [ "$1" = repo ]; then printf '%s\n' "$GH_FORK_URL"; exit 0; fi
printf '%s\n' '{"headRefName":"fork-feature","headRepository":{"name":"repo"},"headRepositoryOwner":{"login":"contributor"}}'
`, fs.FileExec)
	t.Setenv("GH_FORK_URL", repository.Path)
	t.Setenv("PATH", binaryDir)

	arguments := []string{"https://github.com/owner/repo/pull/42", "--herdr"}
	command := NewAddCmd()
	command.SetArgs(arguments)
	if err := command.Execute(); err != nil {
		t.Fatalf("first run: %v", err)
	}
	worktreePath := filepath.Join(workspaceRoot, "pr-42")

	// The fork worktree must be detached, or this test is not exercising the bug.
	listing := repository.RunOutput("-C", bareDir, "worktree", "list", "--porcelain")
	if !strings.Contains(listing, "worktree "+worktreePath+"\nHEAD") || !strings.Contains(listing, "detached") {
		t.Fatalf("expected a detached fork worktree, got %s", listing)
	}
	if err := os.Remove(callPath); err != nil {
		t.Fatal(err)
	}

	// A local worktree that merely shares the fork's branch name must not be
	// mistaken for the PR's worktree.
	decoyPath := filepath.Join(workspaceRoot, "decoy")
	repository.RunOutput("-C", bareDir, "worktree", "add", decoyPath, "fork-feature")

	command = NewAddCmd()
	command.SetArgs(arguments)
	if err := command.Execute(); err != nil {
		t.Fatalf("fork PR rerun must hand off, got: %v", err)
	}

	calls, err := os.ReadFile(callPath) //nolint:gosec // Test-owned temporary path.
	if err != nil {
		t.Fatalf("fork PR rerun never reached Herdr: %v", err)
	}
	want := "worktree\nopen\n--cwd\n" + workspaceRoot + "\n--path\n" + worktreePath + "\n--focus\n"
	if string(calls) != want {
		t.Fatalf("calls = %q, want the fork PR worktree %q", calls, want)
	}
}

// Local commits are work in progress too, so --herdr opens past that refusal
// exactly as it does past a dirty worktree.
func TestAddHerdrOpensUnsyncedPRWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell herdr stub is not executable on Windows")
	}

	repository := testgit.NewTestRepo(t)
	repository.CreateBranch("pr-feature")
	workspaceRoot := filepath.Join(repository.TempDir, "workspace")
	bareDir := filepath.Join(workspaceRoot, ".bare")
	repository.RunOutput("clone", "--bare", repository.Path, bareDir)
	mainPath := filepath.Join(workspaceRoot, "main")
	repository.RunOutput("-C", bareDir, "worktree", "add", mainPath, "main")
	t.Chdir(mainPath)

	binaryDir := t.TempDir()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(gitPath, filepath.Join(binaryDir, "git")); err != nil {
		t.Fatal(err)
	}
	callPath := filepath.Join(t.TempDir(), "calls")
	t.Setenv("HERDR_CALLS", callPath)
	testutil.WriteFileMode(t, filepath.Join(binaryDir, "herdr"), `#!/bin/sh
printf '%s\n' "$@" >> "$HERDR_CALLS"
`, fs.FileExec)
	testutil.WriteFileMode(t, filepath.Join(binaryDir, "gh"), `#!/bin/sh
if [ "$1" = auth ]; then exit 0; fi
printf '%s\n' '{"headRefName":"pr-feature","headRepository":{"name":"repo"},"headRepositoryOwner":{"login":"owner"}}'
`, fs.FileExec)
	t.Setenv("PATH", binaryDir)

	arguments := []string{"https://github.com/owner/repo/pull/42", "--herdr"}
	command := NewAddCmd()
	command.SetArgs(arguments)
	if err := command.Execute(); err != nil {
		t.Fatalf("first run: %v", err)
	}
	worktreePath := filepath.Join(workspaceRoot, "pr-42")
	if err := os.Remove(callPath); err != nil {
		t.Fatal(err)
	}

	// Commit locally so the branch is ahead of the PR head, leaving the worktree
	// clean: this is the other refusal, not the dirty one.
	repository.RunOutput("-C", worktreePath, "-c", "commit.gpgsign=false",
		"-c", "user.email=test@example.com", "-c", "user.name=Test",
		"commit", "--allow-empty", "-m", "local work not yet pushed")

	stderr, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	originalStderr := os.Stderr
	os.Stderr = stderr
	t.Cleanup(func() {
		os.Stderr = originalStderr
		_ = stderr.Close()
	})

	command = NewAddCmd()
	command.SetArgs(arguments)
	if err := command.Execute(); err != nil {
		t.Fatalf("unsynced worktree must still open in Herdr, got: %v", err)
	}

	calls, err := os.ReadFile(callPath) //nolint:gosec // Test-owned temporary path.
	if err != nil {
		t.Fatalf("unsynced worktree never reached Herdr: %v", err)
	}
	want := "worktree\nopen\n--cwd\n" + workspaceRoot + "\n--path\n" + worktreePath + "\n--focus\n"
	if string(calls) != want {
		t.Fatalf("calls = %q, want %q", calls, want)
	}

	// Without --reset the local commits must survive being opened.
	if subject := repository.RunOutput("-C", worktreePath, "log", "-1", "--format=%s"); !strings.Contains(subject, "local work not yet pushed") {
		t.Fatalf("handoff discarded the local commit, HEAD is now %q", subject)
	}

	warning, err := os.ReadFile(stderr.Name()) //nolint:gosec // Test-owned temporary path.
	if err != nil || !strings.Contains(string(warning), "not on remote") {
		t.Fatalf("expected the local-commits warning, got %q (%v)", warning, err)
	}
}

// A refresh that fails for any reason other than work in progress must abort:
// the worktree may be half-synced, and opening it would hide that.
func TestAddHerdrAbortsOnBrokenPRRefresh(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell herdr stub is not executable on Windows")
	}

	repository := testgit.NewTestRepo(t)
	repository.CreateBranch("pr-feature")
	workspaceRoot := filepath.Join(repository.TempDir, "workspace")
	bareDir := filepath.Join(workspaceRoot, ".bare")
	repository.RunOutput("clone", "--bare", repository.Path, bareDir)
	mainPath := filepath.Join(workspaceRoot, "main")
	repository.RunOutput("-C", bareDir, "worktree", "add", mainPath, "main")
	t.Chdir(mainPath)

	binaryDir := t.TempDir()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(gitPath, filepath.Join(binaryDir, "git")); err != nil {
		t.Fatal(err)
	}
	callPath := filepath.Join(t.TempDir(), "calls")
	t.Setenv("HERDR_CALLS", callPath)
	testutil.WriteFileMode(t, filepath.Join(binaryDir, "herdr"), `#!/bin/sh
printf '%s\n' "$@" >> "$HERDR_CALLS"
`, fs.FileExec)
	testutil.WriteFileMode(t, filepath.Join(binaryDir, "gh"), `#!/bin/sh
if [ "$1" = auth ]; then exit 0; fi
printf '%s\n' '{"headRefName":"pr-feature","headRepository":{"name":"repo"},"headRepositoryOwner":{"login":"owner"}}'
`, fs.FileExec)
	t.Setenv("PATH", binaryDir)

	arguments := []string{"https://github.com/owner/repo/pull/42", "--herdr"}
	command := NewAddCmd()
	command.SetArgs(arguments)
	if err := command.Execute(); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if err := os.Remove(callPath); err != nil {
		t.Fatal(err)
	}

	// Break the fetch the refresh depends on, leaving the worktree clean.
	repository.RunOutput("-C", bareDir, "remote", "set-url", "origin", filepath.Join(repository.TempDir, "gone"))

	command = NewAddCmd()
	command.SetArgs(arguments)
	err = command.Execute()

	if err == nil || !strings.Contains(err.Error(), "fetch") {
		t.Fatalf("a broken fetch must abort, got %v", err)
	}
	if calls, readError := os.ReadFile(callPath); !os.IsNotExist(readError) { //nolint:gosec // Test-owned temporary path.
		t.Fatalf("a broken refresh must not reach Herdr: %q", calls)
	}
}

func TestAddHerdrSwitchConflict(t *testing.T) {
	command := NewAddCmd()
	command.SetArgs([]string{"feature", "--herdr", "--switch"})

	err := command.Execute()
	if err == nil || !strings.Contains(err.Error(), "herdr") || !strings.Contains(err.Error(), "switch") {
		t.Fatalf("expected flag conflict, got %v", err)
	}
}

func TestNewAddCmd(t *testing.T) {
	cmd := NewAddCmd()

	// Verify command structure
	if cmd.Use != "add [branch|PR-URL|ref]" {
		t.Errorf("unexpected Use: %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	// Verify required flags exist with correct configuration
	flags := []struct {
		name      string
		shorthand string
		defValue  string
		valueType string
	}{
		{"switch", "s", "false", "bool"},
		{"herdr", "", "false", "bool"},
		{"base", "", "", "string"},
		{"detach", "d", "false", "bool"},
		{"name", "", "", "string"},
		{"pr", "", "0", "int"},
		{"from", "", "", "string"},
	}

	for _, f := range flags {
		flag := cmd.Flags().Lookup(f.name)
		if flag == nil {
			t.Errorf("expected --%s flag to exist", f.name)
			continue
		}
		if f.shorthand != "" && flag.Shorthand != f.shorthand {
			t.Errorf("--%s: expected shorthand %q, got %q", f.name, f.shorthand, flag.Shorthand)
		}
		if flag.DefValue != f.defValue {
			t.Errorf("--%s: expected default value %q, got %q", f.name, f.defValue, flag.DefValue)
		}
		if flag.Value.Type() != f.valueType {
			t.Errorf("--%s: expected type %q, got %q", f.name, f.valueType, flag.Value.Type())
		}
	}

	// Verify ValidArgsFunction is set
	if cmd.ValidArgsFunction == nil {
		t.Error("expected ValidArgsFunction to be set")
	}
}

func TestRunAdd_NotInWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := testutil.TempDir(t)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	err = runAdd([]string{"feature-test"}, false, false, "", "", false, 0, false, "", false)
	if !errors.Is(err, workspace.ErrNotInWorkspace) {
		t.Errorf("expected ErrNotInWorkspace, got %v", err)
	}
}

func TestRunAdd_PRValidation(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := testutil.TempDir(t)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	t.Run("base flag cannot be used with --pr", func(t *testing.T) {
		err := runAdd(nil, false, false, "main", "", false, 123, false, "", false)
		if err == nil || !strings.Contains(err.Error(), "--base cannot be used with PR") {
			t.Errorf("expected base/PR error, got %v", err)
		}
	})

	t.Run("detach flag cannot be used with --pr", func(t *testing.T) {
		err := runAdd(nil, false, false, "", "", true, 123, false, "", false)
		if err == nil || !strings.Contains(err.Error(), "--detach cannot be used with PR") {
			t.Errorf("expected detach/PR error, got %v", err)
		}
	})

	t.Run("negative --pr gives clear error", func(t *testing.T) {
		err := runAdd(nil, false, false, "", "", false, -5, false, "", false)
		if err == nil || !strings.Contains(err.Error(), "--pr must be a positive number") {
			t.Errorf("expected positive number error, got %v", err)
		}
	})

	t.Run("--pr cannot be combined with positional argument", func(t *testing.T) {
		err := runAdd([]string{"feature"}, false, false, "", "", false, 123, false, "", false)
		if err == nil || !strings.Contains(err.Error(), "--pr flag cannot be combined with positional argument") {
			t.Errorf("expected --pr/positional conflict error, got %v", err)
		}
	})

	t.Run("old #N syntax gives helpful error", func(t *testing.T) {
		err := runAdd([]string{"#123"}, false, false, "", "", false, 0, false, "", false)
		if err == nil || !strings.Contains(err.Error(), "syntax no longer supported") {
			t.Errorf("expected helpful migration error, got %v", err)
		}
	})

	t.Run("base flag cannot be used with PR URL", func(t *testing.T) {
		err := runAdd([]string{"https://github.com/owner/repo/pull/456"}, false, false, "main", "", false, 0, false, "", false)
		if err == nil || !strings.Contains(err.Error(), "--base cannot be used with PR") {
			t.Errorf("expected base/PR error, got %v", err)
		}
	})

	t.Run("detach flag cannot be used with PR URL", func(t *testing.T) {
		err := runAdd([]string{"https://github.com/owner/repo/pull/456"}, false, false, "", "", true, 0, false, "", false)
		if err == nil || !strings.Contains(err.Error(), "--detach cannot be used with PR") {
			t.Errorf("expected detach/PR error, got %v", err)
		}
	})

	t.Run("reset flag can only be used with PR references", func(t *testing.T) {
		err := runAdd([]string{"feature-branch"}, false, false, "", "", false, 0, true, "", false)
		if err == nil || !strings.Contains(err.Error(), "--reset can only be used with PR references") {
			t.Errorf("expected --reset/PR error, got %v", err)
		}
	})
}

func TestRunAdd_DetachBaseValidation(t *testing.T) {
	t.Run("detach and base cannot be used together", func(t *testing.T) {
		err := runAdd([]string{"v1.0.0"}, false, false, "main", "", true, 0, false, "", false)
		if err == nil || err.Error() != "--detach and --base cannot be used together" {
			t.Errorf("expected detach/base error, got %v", err)
		}
	})
}

func TestRunAdd_InputValidation(t *testing.T) {
	// Save and restore cwd - we need to be outside a workspace
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := testutil.TempDir(t)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	t.Run("whitespace-only branch name", func(t *testing.T) {
		// Whitespace is trimmed, resulting in empty string
		// This should fail with "requires branch" error
		err := runAdd([]string{"   "}, false, false, "", "", false, 0, false, "", false)
		if err == nil || !strings.Contains(err.Error(), "requires branch") {
			t.Errorf("expected 'requires branch' error for whitespace-only branch name, got %v", err)
		}
	})

	t.Run("no args and no --pr flag", func(t *testing.T) {
		err := runAdd(nil, false, false, "", "", false, 0, false, "", false)
		if err == nil || !strings.Contains(err.Error(), "requires branch") {
			t.Errorf("expected 'requires branch' error, got %v", err)
		}
	})

	t.Run("leading and trailing whitespace is trimmed", func(t *testing.T) {
		// The trimming happens, then workspace detection runs
		// We're not in a workspace, so we'll get that error
		// But this verifies the trim doesn't crash
		err := runAdd([]string{"  feature-test  "}, false, false, "", "", false, 0, false, "", false)
		if !errors.Is(err, workspace.ErrNotInWorkspace) {
			t.Errorf("expected ErrNotInWorkspace after trimming, got %v", err)
		}
	})

	t.Run("PR URL with /files suffix works", func(t *testing.T) {
		// PR URLs with /files suffix should be detected as PR references
		// Flag validation happens before workspace detection
		err := runAdd([]string{"https://github.com/owner/repo/pull/123/files"}, false, false, "main", "", false, 0, false, "", false)
		if err == nil || !strings.Contains(err.Error(), "--base cannot be used with PR") {
			t.Errorf("expected base/PR error for URL with /files suffix, got %v", err)
		}
	})

	t.Run("PR URL with query params works", func(t *testing.T) {
		err := runAdd([]string{"https://github.com/owner/repo/pull/123?diff=split"}, false, false, "", "", true, 0, false, "", false)
		if err == nil || !strings.Contains(err.Error(), "--detach cannot be used with PR") {
			t.Errorf("expected detach/PR error for URL with query params, got %v", err)
		}
	})
}

func TestFindSourceWorktree(t *testing.T) {
	t.Run("returns empty at workspace root", func(t *testing.T) {
		workspaceRoot := testutil.TempDir(t)

		result := findSourceWorktree(workspaceRoot, workspaceRoot)
		if result != "" {
			t.Errorf("expected empty string at workspace root, got %q", result)
		}
	})

	t.Run("returns worktree path when in worktree", func(t *testing.T) {
		workspaceRoot := testutil.TempDir(t)

		// Create a fake worktree directory with .git file
		worktreeDir := filepath.Join(workspaceRoot, "main")
		if err := os.MkdirAll(worktreeDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		gitFile := filepath.Join(worktreeDir, ".git")
		if err := os.WriteFile(gitFile, []byte("gitdir: ../.bare"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		result := findSourceWorktree(worktreeDir, workspaceRoot)
		if result != worktreeDir {
			t.Errorf("expected %q, got %q", worktreeDir, result)
		}
	})

	t.Run("returns worktree from subdirectory", func(t *testing.T) {
		workspaceRoot := testutil.TempDir(t)

		// Create worktree with subdirectory
		worktreeDir := filepath.Join(workspaceRoot, "main")
		subDir := filepath.Join(worktreeDir, "src", "pkg")
		if err := os.MkdirAll(subDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		gitFile := filepath.Join(worktreeDir, ".git")
		if err := os.WriteFile(gitFile, []byte("gitdir: ../.bare"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		result := findSourceWorktree(subDir, workspaceRoot)
		if result != worktreeDir {
			t.Errorf("expected %q, got %q", worktreeDir, result)
		}
	})

	t.Run("returns empty when not in worktree", func(t *testing.T) {
		workspaceRoot := testutil.TempDir(t)

		// Create a directory that's not a worktree
		otherDir := filepath.Join(workspaceRoot, "other")
		if err := os.MkdirAll(otherDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}

		result := findSourceWorktree(otherDir, workspaceRoot)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})
}

func TestFindFallbackSourceWorktree(t *testing.T) {
	// Helper to create a bare repo with initial content
	setupBareRepo := func(t *testing.T, defaultBranch string) (tempDir, bareDir string) {
		t.Helper()
		tempDir = testutil.TempDir(t)
		bareDir = filepath.Join(tempDir, ".bare")

		// Create a regular repo first, then clone it as bare
		srcDir := filepath.Join(tempDir, "src")
		if err := os.MkdirAll(srcDir, fs.DirStrict); err != nil {
			t.Fatalf("failed to create src dir: %v", err)
		}

		// Init with specified default branch
		cmd := exec.Command("git", "init", "-b", defaultBranch) //nolint:gosec
		cmd.Dir = srcDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to init: %v", err)
		}

		// Configure git
		for _, cfg := range [][]string{
			{"user.email", "test@test.com"},
			{"user.name", "Test"},
			{"commit.gpgsign", "false"},
		} {
			cmd = exec.Command("git", "config", cfg[0], cfg[1]) //nolint:gosec
			cmd.Dir = srcDir
			if err := cmd.Run(); err != nil {
				t.Fatalf("failed to set config: %v", err)
			}
		}

		// Create initial commit
		if err := os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("test"), fs.FileStrict); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
		cmd = exec.Command("git", "add", ".")
		cmd.Dir = srcDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add: %v", err)
		}
		cmd = exec.Command("git", "commit", "-m", "init")
		cmd.Dir = srcDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// Clone as bare repo
		cmd = exec.Command("git", "clone", "--bare", srcDir, bareDir) //nolint:gosec
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to clone bare: %v", err)
		}

		return tempDir, bareDir
	}

	t.Run("returns worktree for default branch", func(t *testing.T) {
		tempDir, bareDir := setupBareRepo(t, "develop")

		// Create worktree for develop branch
		developDir := filepath.Join(tempDir, "develop")
		cmd := exec.Command("git", "worktree", "add", developDir, "develop") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add worktree: %v", err)
		}

		t.Cleanup(func() {
			cmd := exec.Command("git", "worktree", "remove", "--force", developDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
		})

		result := findFallbackSourceWorktree(bareDir)
		if result != developDir {
			t.Errorf("expected %q, got %q", developDir, result)
		}
	})

	t.Run("falls back to main when no default worktree", func(t *testing.T) {
		tempDir, bareDir := setupBareRepo(t, "develop")

		// Create main branch from develop
		cmd := exec.Command("git", "branch", "main", "develop") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create main branch: %v", err)
		}

		// Only create worktree for main (not develop which is the default)
		mainDir := filepath.Join(tempDir, "main")
		cmd = exec.Command("git", "worktree", "add", mainDir, "main") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add main worktree: %v", err)
		}

		t.Cleanup(func() {
			cmd := exec.Command("git", "worktree", "remove", "--force", mainDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
		})

		result := findFallbackSourceWorktree(bareDir)
		if result != mainDir {
			t.Errorf("expected %q, got %q", mainDir, result)
		}
	})

	t.Run("falls back to master when no main", func(t *testing.T) {
		tempDir, bareDir := setupBareRepo(t, "develop")

		// Create master branch
		cmd := exec.Command("git", "branch", "master", "develop") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create master branch: %v", err)
		}

		// Only create worktree for master
		masterDir := filepath.Join(tempDir, "master")
		cmd = exec.Command("git", "worktree", "add", masterDir, "master") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add master worktree: %v", err)
		}

		t.Cleanup(func() {
			cmd := exec.Command("git", "worktree", "remove", "--force", masterDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
		})

		result := findFallbackSourceWorktree(bareDir)
		if result != masterDir {
			t.Errorf("expected %q, got %q", masterDir, result)
		}
	})

	t.Run("returns empty when no worktrees exist", func(t *testing.T) {
		_, bareDir := setupBareRepo(t, "main")

		result := findFallbackSourceWorktree(bareDir)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("returns empty when no matching branch worktree", func(t *testing.T) {
		tempDir, bareDir := setupBareRepo(t, "develop")

		// Create feature branch
		cmd := exec.Command("git", "branch", "feature-x", "develop") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create feature branch: %v", err)
		}

		// Only create worktree for feature-x (not develop/main/master)
		featureDir := filepath.Join(tempDir, "feature-x")
		cmd = exec.Command("git", "worktree", "add", featureDir, "feature-x") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add feature worktree: %v", err)
		}

		t.Cleanup(func() {
			cmd := exec.Command("git", "worktree", "remove", "--force", featureDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
		})

		result := findFallbackSourceWorktree(bareDir)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("falls back to directory named main when branch differs", func(t *testing.T) {
		tempDir, bareDir := setupBareRepo(t, "develop")

		// Create feature branch
		cmd := exec.Command("git", "branch", "feature-working", "develop") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create feature branch: %v", err)
		}

		// Create worktree in directory named "main" but checked out on feature-working
		mainDir := filepath.Join(tempDir, "main")
		cmd = exec.Command("git", "worktree", "add", mainDir, "feature-working") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add worktree: %v", err)
		}

		t.Cleanup(func() {
			cmd := exec.Command("git", "worktree", "remove", "--force", mainDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
		})

		result := findFallbackSourceWorktree(bareDir)
		if result != mainDir {
			t.Errorf("expected %q (directory-name fallback), got %q", mainDir, result)
		}
	})

	t.Run("avoids duplicate when default branch is main", func(t *testing.T) {
		tempDir, bareDir := setupBareRepo(t, "main")

		// Create worktree for main branch
		mainDir := filepath.Join(tempDir, "main")
		cmd := exec.Command("git", "worktree", "add", mainDir, "main") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add main worktree: %v", err)
		}

		t.Cleanup(func() {
			cmd := exec.Command("git", "worktree", "remove", "--force", mainDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
		})

		// Should find main worktree without issue (no duplicate candidate bug)
		result := findFallbackSourceWorktree(bareDir)
		if result != mainDir {
			t.Errorf("expected %q, got %q", mainDir, result)
		}
	})

	t.Run("branch match takes priority over directory match", func(t *testing.T) {
		tempDir, bareDir := setupBareRepo(t, "develop")

		// Create main branch
		cmd := exec.Command("git", "branch", "main", "develop") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create main branch: %v", err)
		}

		// Create feature branch
		cmd = exec.Command("git", "branch", "feature-x", "develop") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create feature branch: %v", err)
		}

		// Create worktree checked out on main branch, but in directory named "other"
		mainBranchDir := filepath.Join(tempDir, "other")
		cmd = exec.Command("git", "worktree", "add", mainBranchDir, "main") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add main branch worktree: %v", err)
		}

		// Create worktree in directory named "main" but checked out on feature-x
		mainNameDir := filepath.Join(tempDir, "main")
		cmd = exec.Command("git", "worktree", "add", mainNameDir, "feature-x") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add main-named worktree: %v", err)
		}

		t.Cleanup(func() {
			cmd := exec.Command("git", "worktree", "remove", "--force", mainNameDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
			cmd = exec.Command("git", "worktree", "remove", "--force", mainBranchDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
		})

		// Should return worktree on main BRANCH, not directory named "main"
		result := findFallbackSourceWorktree(bareDir)
		if result != mainBranchDir {
			t.Errorf("expected %q (branch match), got %q", mainBranchDir, result)
		}
	})

	t.Run("prefers main directory over master directory in fallback", func(t *testing.T) {
		tempDir, bareDir := setupBareRepo(t, "develop")

		// Create two feature branches
		cmd := exec.Command("git", "branch", "feature-a", "develop") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create feature-a branch: %v", err)
		}

		cmd = exec.Command("git", "branch", "feature-b", "develop") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create feature-b branch: %v", err)
		}

		// Create worktree in directory named "master" on feature-a
		masterDir := filepath.Join(tempDir, "master")
		cmd = exec.Command("git", "worktree", "add", masterDir, "feature-a") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add master-named worktree: %v", err)
		}

		// Create worktree in directory named "main" on feature-b
		mainDir := filepath.Join(tempDir, "main")
		cmd = exec.Command("git", "worktree", "add", mainDir, "feature-b") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add main-named worktree: %v", err)
		}

		t.Cleanup(func() {
			cmd := exec.Command("git", "worktree", "remove", "--force", mainDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
			cmd = exec.Command("git", "worktree", "remove", "--force", masterDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
		})

		// Should prefer "main" directory over "master" directory
		result := findFallbackSourceWorktree(bareDir)
		if result != mainDir {
			t.Errorf("expected %q (main directory), got %q", mainDir, result)
		}
	})
}

func TestLinkDirectoriesFromSource_LoadsConfigFromConfigWorktree(t *testing.T) {
	tempDir := testutil.TempDir(t)

	configDir := filepath.Join(tempDir, "main")
	if err := os.MkdirAll(configDir, fs.DirStrict); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	tomlBody := "[link]\npatterns = [\".claude\"]\n"
	if err := os.WriteFile(filepath.Join(configDir, ".grove.toml"), []byte(tomlBody), fs.FileStrict); err != nil {
		t.Fatalf("write toml: %v", err)
	}

	sourceDir := filepath.Join(tempDir, "feature")
	if err := os.MkdirAll(filepath.Join(sourceDir, ".claude"), fs.DirStrict); err != nil {
		t.Fatalf("mkdir source/.claude: %v", err)
	}

	destDir := filepath.Join(tempDir, "dest")
	if err := os.MkdirAll(destDir, fs.DirStrict); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}

	result := linkDirectoriesFromSource(sourceDir, destDir, configDir)
	if result == nil || len(result.Linked) != 1 || result.Linked[0] != ".claude" {
		t.Fatalf("expected .claude linked, got %+v", result)
	}
}

func TestFindConfigWorktree(t *testing.T) {
	setupBare := func(t *testing.T) (tempDir, bareDir string) {
		t.Helper()
		tempDir = testutil.TempDir(t)
		bareDir = filepath.Join(tempDir, ".bare")
		srcDir := filepath.Join(tempDir, "src")
		if err := os.MkdirAll(srcDir, fs.DirStrict); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		run := func(dir, name string, args ...string) {
			t.Helper()
			cmd := exec.Command(name, args...) //nolint:gosec
			cmd.Dir = dir
			if err := cmd.Run(); err != nil {
				t.Fatalf("%s %v: %v", name, args, err)
			}
		}
		run(srcDir, "git", "init", "-b", "main")
		run(srcDir, "git", "config", "user.email", "a@a")
		run(srcDir, "git", "config", "user.name", "a")
		run(srcDir, "git", "config", "commit.gpgsign", "false")
		if err := os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("x"), fs.FileStrict); err != nil {
			t.Fatalf("write: %v", err)
		}
		run(srcDir, "git", "add", ".")
		run(srcDir, "git", "commit", "-m", "init")
		run("", "git", "clone", "--bare", srcDir, bareDir)
		return tempDir, bareDir
	}

	removeWorktree := func(t *testing.T, bareDir, wtDir string) {
		t.Helper()
		t.Cleanup(func() {
			cmd := exec.Command("git", "worktree", "remove", "--force", wtDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
		})
	}

	t.Run("returns main worktree with .grove.toml", func(t *testing.T) {
		tempDir, bareDir := setupBare(t)
		mainDir := filepath.Join(tempDir, "main")
		cmd := exec.Command("git", "worktree", "add", mainDir, "main") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("worktree add: %v", err)
		}
		removeWorktree(t, bareDir, mainDir)
		if err := os.WriteFile(filepath.Join(mainDir, ".grove.toml"), []byte("[link]\n"), fs.FileStrict); err != nil {
			t.Fatalf("write toml: %v", err)
		}
		got := findConfigWorktree(bareDir)
		if got != mainDir {
			t.Errorf("expected %q, got %q", mainDir, got)
		}
	})

	t.Run("falls back to any worktree with .grove.toml", func(t *testing.T) {
		tempDir, bareDir := setupBare(t)
		featDir := filepath.Join(tempDir, "feat")
		cmd := exec.Command("git", "worktree", "add", "-b", "feat", featDir) //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("worktree add: %v", err)
		}
		removeWorktree(t, bareDir, featDir)
		if err := os.WriteFile(filepath.Join(featDir, ".grove.toml"), []byte("[link]\n"), fs.FileStrict); err != nil {
			t.Fatalf("write toml: %v", err)
		}
		got := findConfigWorktree(bareDir)
		if got != featDir {
			t.Errorf("expected %q, got %q", featDir, got)
		}
	})

	t.Run("returns empty when no worktree has .grove.toml", func(t *testing.T) {
		tempDir, bareDir := setupBare(t)
		mainDir := filepath.Join(tempDir, "main")
		cmd := exec.Command("git", "worktree", "add", mainDir, "main") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("worktree add: %v", err)
		}
		removeWorktree(t, bareDir, mainDir)
		if got := findConfigWorktree(bareDir); got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})
}

func TestRunAdd_FromValidation(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Helper to create a grove workspace with worktrees
	setupWorkspace := func(t *testing.T) (tempDir, bareDir string) {
		t.Helper()
		tempDir = testutil.TempDir(t)
		bareDir = filepath.Join(tempDir, ".bare")

		// Create a regular repo first, then clone it as bare
		srcDir := filepath.Join(tempDir, "src")
		if err := os.MkdirAll(srcDir, fs.DirStrict); err != nil {
			t.Fatalf("failed to create src dir: %v", err)
		}

		// Init with main branch
		cmd := exec.Command("git", "init", "-b", "main")
		cmd.Dir = srcDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to init: %v", err)
		}

		// Configure git
		for _, cfg := range [][]string{
			{"user.email", "test@test.com"},
			{"user.name", "Test"},
			{"commit.gpgsign", "false"},
		} {
			cmd = exec.Command("git", "config", cfg[0], cfg[1]) //nolint:gosec
			cmd.Dir = srcDir
			if err := cmd.Run(); err != nil {
				t.Fatalf("failed to set config: %v", err)
			}
		}

		// Create initial commit
		if err := os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("test"), fs.FileStrict); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
		cmd = exec.Command("git", "add", ".")
		cmd.Dir = srcDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add: %v", err)
		}
		cmd = exec.Command("git", "commit", "-m", "init")
		cmd.Dir = srcDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// Clone as bare repo
		cmd = exec.Command("git", "clone", "--bare", srcDir, bareDir) //nolint:gosec
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to clone bare: %v", err)
		}

		// Clean up src directory
		if err := os.RemoveAll(srcDir); err != nil {
			t.Fatalf("failed to remove src: %v", err)
		}

		// Create main worktree
		mainDir := filepath.Join(tempDir, "main")
		cmd = exec.Command("git", "worktree", "add", mainDir, "main") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add main worktree: %v", err)
		}

		// Register cleanup to remove worktrees before temp dir cleanup (Windows file locks)
		t.Cleanup(func() {
			_ = os.Chdir(origDir)                                                // Exit temp dir entirely (Windows requirement)
			cmd := exec.Command("git", "worktree", "remove", "--force", mainDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
		})

		return tempDir, bareDir
	}

	t.Run("--from with nonexistent worktree returns error", func(t *testing.T) {
		tempDir, _ := setupWorkspace(t)
		mainDir := filepath.Join(tempDir, "main")
		if err := os.Chdir(mainDir); err != nil {
			t.Fatal(err)
		}

		err := runAdd([]string{"feature-test"}, false, false, "", "", false, 0, false, "nonexistent", false)
		if err == nil {
			t.Fatal("expected error for nonexistent --from worktree")
		}
		if !strings.Contains(err.Error(), "worktree") || !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'worktree not found' error, got %v", err)
		}
	})

	t.Run("--from with valid worktree by name succeeds", func(t *testing.T) {
		tempDir, bareDir := setupWorkspace(t)
		mainDir := filepath.Join(tempDir, "main")
		if err := os.Chdir(mainDir); err != nil {
			t.Fatal(err)
		}

		// Create another worktree to use as --from source
		sourceDir := filepath.Join(tempDir, "source")
		cmd := exec.Command("git", "worktree", "add", "-b", "source", sourceDir, "main") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create source worktree: %v", err)
		}

		// Register cleanup for worktrees created in this subtest (Windows file locks)
		featureDir := filepath.Join(tempDir, "feature-from-test")
		t.Cleanup(func() {
			_ = os.Chdir(origDir)                                                   // Exit temp dir entirely (Windows requirement)
			cmd := exec.Command("git", "worktree", "remove", "--force", featureDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
			cmd = exec.Command("git", "worktree", "remove", "--force", sourceDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
		})

		// Create a new worktree with --from pointing to source
		err := runAdd([]string{"feature-from-test"}, false, false, "", "", false, 0, false, "source", false)
		if err != nil {
			t.Errorf("expected success with valid --from, got %v", err)
		}
	})
}

func TestRunAddFromBranch_WorktreeExistsHint(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	tempDir := testutil.TempDir(t)
	bareDir := filepath.Join(tempDir, ".bare")

	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, fs.DirStrict); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = srcDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	for _, cfg := range [][]string{
		{"user.email", "test@test.com"},
		{"user.name", "Test"},
		{"commit.gpgsign", "false"},
	} {
		cmd = exec.Command("git", "config", cfg[0], cfg[1]) //nolint:gosec
		cmd.Dir = srcDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to set config: %v", err)
		}
	}

	if err := os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("test"), fs.FileStrict); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = srcDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add: %v", err)
	}
	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = srcDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	cmd = exec.Command("git", "clone", "--bare", srcDir, bareDir) //nolint:gosec
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to clone bare: %v", err)
	}

	if err := os.RemoveAll(srcDir); err != nil {
		t.Fatalf("failed to remove src: %v", err)
	}

	mainDir := filepath.Join(tempDir, "main")
	cmd = exec.Command("git", "worktree", "add", mainDir, "main") //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add main worktree: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		cmd := exec.Command("git", "worktree", "remove", "--force", mainDir) //nolint:gosec
		cmd.Dir = bareDir
		_ = cmd.Run()
	})

	if err := os.Chdir(mainDir); err != nil {
		t.Fatal(err)
	}

	err = runAdd([]string{"main"}, false, false, "", "", false, 0, false, "", false)
	if err == nil {
		t.Fatal("expected error for existing worktree")
	}

	if !strings.Contains(err.Error(), "grove list") {
		t.Errorf("expected error to contain 'grove list' hint, got: %v", err)
	}
	if !strings.Contains(err.Error(), "--name") {
		t.Errorf("expected error to contain '--name' hint, got: %v", err)
	}
}

func TestRunAdd_LinkPatternsAppliedFromOutsideWorktree(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	tempDir := testutil.TempDir(t)
	bareDir := filepath.Join(tempDir, ".bare")
	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, fs.DirStrict); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}

	run := func(dir, name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...) //nolint:gosec
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("%s %v: %v", name, args, err)
		}
	}
	run(srcDir, "git", "init", "-b", "main")
	run(srcDir, "git", "config", "user.email", "a@a")
	run(srcDir, "git", "config", "user.name", "a")
	run(srcDir, "git", "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("x"), fs.FileStrict); err != nil {
		t.Fatalf("write: %v", err)
	}
	run(srcDir, "git", "add", ".")
	run(srcDir, "git", "commit", "-m", "init")
	run("", "git", "clone", "--bare", srcDir, bareDir)
	if err := os.RemoveAll(srcDir); err != nil {
		t.Fatalf("rm src: %v", err)
	}

	mainDir := filepath.Join(tempDir, "main")
	cmd := exec.Command("git", "worktree", "add", mainDir, "main") //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("worktree add main: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		featDir := filepath.Join(tempDir, "feat")
		for _, wt := range []string{featDir, mainDir} {
			cmd := exec.Command("git", "worktree", "remove", "--force", wt) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
		}
	})

	if err := os.WriteFile(filepath.Join(mainDir, ".grove.toml"), []byte("[link]\npatterns = [\".claude\"]\n"), fs.FileStrict); err != nil {
		t.Fatalf("write toml: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(mainDir, ".claude"), fs.DirStrict); err != nil {
		t.Fatalf("mkdir .claude: %v", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	if err := runAdd([]string{"feat"}, false, false, "", "", false, 0, false, "", false); err != nil {
		t.Fatalf("runAdd: %v", err)
	}

	linkPath := filepath.Join(tempDir, "feat", ".claude")
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("expected symlink at %s: %v", linkPath, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected %s to be a symlink, got mode %v", linkPath, info.Mode())
	}
}

func TestRunAdd_LinkAppliedWhenOnlyNonMainWorktreeHasConfig(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	tempDir := testutil.TempDir(t)
	bareDir := filepath.Join(tempDir, ".bare")
	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, fs.DirStrict); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}

	run := func(dir, name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...) //nolint:gosec
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("%s %v: %v", name, args, err)
		}
	}
	run(srcDir, "git", "init", "-b", "dev")
	run(srcDir, "git", "config", "user.email", "a@a")
	run(srcDir, "git", "config", "user.name", "a")
	run(srcDir, "git", "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("x"), fs.FileStrict); err != nil {
		t.Fatalf("write: %v", err)
	}
	run(srcDir, "git", "add", ".")
	run(srcDir, "git", "commit", "-m", "init")
	run("", "git", "clone", "--bare", srcDir, bareDir)
	if err := os.RemoveAll(srcDir); err != nil {
		t.Fatalf("rm src: %v", err)
	}

	// Create feature branch, then a worktree on it with a name that does NOT
	// match main/master and is not the default branch. This is the bug
	// scenario: findFallbackSourceWorktree returns "" so sourceWorktree stays
	// empty unless the code falls back to the config worktree.
	run(bareDir, "git", "branch", "feat-x", "dev")
	featDir := filepath.Join(tempDir, "feat-x")
	cmd := exec.Command("git", "worktree", "add", featDir, "feat-x") //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("worktree add feat-x: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		newworkDir := filepath.Join(tempDir, "newwork")
		for _, wt := range []string{newworkDir, featDir} {
			cmd := exec.Command("git", "worktree", "remove", "--force", wt) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
		}
	})

	if err := os.WriteFile(filepath.Join(featDir, ".grove.toml"), []byte("[link]\npatterns = [\".claude\"]\n"), fs.FileStrict); err != nil {
		t.Fatalf("write toml: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(featDir, ".claude"), fs.DirStrict); err != nil {
		t.Fatalf("mkdir .claude: %v", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	if err := runAdd([]string{"newwork"}, false, false, "", "", false, 0, false, "", false); err != nil {
		t.Fatalf("runAdd: %v", err)
	}

	linkPath := filepath.Join(tempDir, "newwork", ".claude")
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("expected symlink at %s: %v", linkPath, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected %s to be a symlink, got mode %v", linkPath, info.Mode())
	}
}

func TestCompleteFromWorktree(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Helper to create a grove workspace with worktrees
	setupWorkspace := func(t *testing.T) (tempDir, bareDir string) {
		t.Helper()
		tempDir = testutil.TempDir(t)
		bareDir = filepath.Join(tempDir, ".bare")

		srcDir := filepath.Join(tempDir, "src")
		if err := os.MkdirAll(srcDir, fs.DirStrict); err != nil {
			t.Fatalf("failed to create src dir: %v", err)
		}

		cmd := exec.Command("git", "init", "-b", "main")
		cmd.Dir = srcDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to init: %v", err)
		}

		for _, cfg := range [][]string{
			{"user.email", "test@test.com"},
			{"user.name", "Test"},
			{"commit.gpgsign", "false"},
		} {
			cmd = exec.Command("git", "config", cfg[0], cfg[1]) //nolint:gosec
			cmd.Dir = srcDir
			if err := cmd.Run(); err != nil {
				t.Fatalf("failed to set config: %v", err)
			}
		}

		if err := os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("test"), fs.FileStrict); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
		cmd = exec.Command("git", "add", ".")
		cmd.Dir = srcDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add: %v", err)
		}
		cmd = exec.Command("git", "commit", "-m", "init")
		cmd.Dir = srcDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		cmd = exec.Command("git", "clone", "--bare", srcDir, bareDir) //nolint:gosec
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to clone bare: %v", err)
		}

		if err := os.RemoveAll(srcDir); err != nil {
			t.Fatalf("failed to remove src: %v", err)
		}

		return tempDir, bareDir
	}

	t.Run("returns available worktree names", func(t *testing.T) {
		tempDir, bareDir := setupWorkspace(t)

		// Create worktrees
		mainDir := filepath.Join(tempDir, "main")
		cmd := exec.Command("git", "worktree", "add", mainDir, "main") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add main worktree: %v", err)
		}

		featureDir := filepath.Join(tempDir, "feature")
		cmd = exec.Command("git", "worktree", "add", "-b", "feature", featureDir, "main") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add feature worktree: %v", err)
		}

		// Register cleanup to remove worktrees before temp dir cleanup (Windows file locks)
		t.Cleanup(func() {
			_ = os.Chdir(origDir)                                                   // Exit temp dir entirely (Windows requirement)
			cmd := exec.Command("git", "worktree", "remove", "--force", featureDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
			cmd = exec.Command("git", "worktree", "remove", "--force", mainDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
		})

		if err := os.Chdir(mainDir); err != nil {
			t.Fatal(err)
		}

		completions, directive := completeFromWorktree(nil, nil, "")

		if directive != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("expected ShellCompDirectiveNoFileComp, got %d", directive)
		}

		// Should include both worktrees
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
			t.Error("completions should include 'main'")
		}
		if !hasFeature {
			t.Error("completions should include 'feature'")
		}
	})
}
