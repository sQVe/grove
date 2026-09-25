package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/testutil"
	testgit "github.com/sqve/grove/internal/testutil/git"
)

func stubHerdr(t *testing.T, result string) string {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("shell herdr stub is not executable on Windows")
	}

	directory := t.TempDir()
	argumentsPath := filepath.Join(directory, "arguments")
	t.Setenv("HERDR_ARGUMENTS", argumentsPath)
	testutil.WriteFileMode(t, filepath.Join(directory, "herdr"), "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$HERDR_ARGUMENTS\"\n"+result, fs.FileExec)
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))

	return argumentsPath
}

// Log one line per invocation so tests can assert close order.
func stubHerdrCalls(t *testing.T, listScript, closeScript string) string {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("shell herdr stub is not executable on Windows")
	}

	directory := t.TempDir()
	callsPath := filepath.Join(directory, "calls")
	t.Setenv("HERDR_CALLS", callsPath)
	t.Setenv("HERDR_WORKSPACE_ID", "")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$HERDR_CALLS\"\ncase \"$1 $2\" in\n" +
		"'worktree list')\n" + listScript + "\n;;\n" +
		"'workspace close')\n" + closeScript + "\n;;\nesac\n"
	testutil.WriteFileMode(t, filepath.Join(directory, "herdr"), script, fs.FileExec)
	// A fake gh keeps prune from asking GitHub about merged PRs.
	testutil.WriteFileMode(t, filepath.Join(directory, "gh"), "#!/bin/sh\nexit 1\n", fs.FileExec)
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))

	return callsPath
}

func herdrListScript(t *testing.T, workspaces map[string]string) string {
	t.Helper()

	type entry struct {
		Path            string `json:"path"`
		Branch          string `json:"branch"`
		OpenWorkspaceID string `json:"open_workspace_id,omitempty"`
	}
	var worktrees []entry
	for path, id := range workspaces {
		worktrees = append(worktrees, entry{Path: path, Branch: filepath.Base(path), OpenWorkspaceID: id})
	}
	result := map[string]any{"id": "cli:worktree:list", "result": map[string]any{"type": "worktree_list", "worktrees": worktrees}}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	listPath := filepath.Join(t.TempDir(), "list.json")
	testutil.WriteFile(t, listPath, string(data))

	return "cat '" + listPath + "'"
}

func readHerdrCalls(t *testing.T, callsPath string) []string {
	t.Helper()

	data, err := os.ReadFile(callsPath) // nolint:gosec // The test creates this log in t.TempDir().
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}

	return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
}

func executeSwitch(t *testing.T, arguments ...string) (stdoutText, stderrText string, executionError error) {
	t.Helper()

	return executeCommand(t, NewSwitchCmd(), arguments...)
}

func executeCommand(t *testing.T, command *cobra.Command, arguments ...string) (stdoutText, stderrText string, executionError error) {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	originalStdout := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = originalStdout }()

	var stderr bytes.Buffer
	previousOutput := logger.SetOutput(&stderr)
	defer logger.SetOutput(previousOutput)
	command.SilenceErrors = true
	command.SilenceUsage = true
	command.SetArgs(arguments)
	err = command.Execute()

	_ = writer.Close()
	var stdout bytes.Buffer
	_, _ = io.Copy(&stdout, reader)

	return stdout.String(), stderr.String(), err
}

func assertHerdrNotCalled(t *testing.T, argumentsPath string) {
	t.Helper()

	if _, err := os.Stat(argumentsPath); !os.IsNotExist(err) {
		t.Fatalf("herdr must not run: marker stat = %v", err)
	}
}

func TestHerdrLabel(t *testing.T) {
	for _, scenario := range []struct {
		branch string
		want   string
	}{
		{branch: "feat/auth", want: "auth"},
		{branch: "abu-377-set-nice-workspace-name-when-using-herdr", want: "set nice workspace name when using herdr"},
		{branch: "fix_typo", want: "fix typo"},
		{branch: "abu-377", want: "abu-377"},
		{branch: "", want: ""},
		{branch: "team/feat/ABU-377-fix__the---typo", want: "fix the typo"},
		{branch: "feat/ABU-377-___", want: "feat/ABU-377-___"},
		{branch: "feat/", want: "feat/"},
	} {
		t.Run(scenario.branch, func(t *testing.T) {
			label := herdrLabel(scenario.branch)

			if label != scenario.want {
				t.Errorf("herdrLabel(%q) = %q, want %q", scenario.branch, label, scenario.want)
			}
		})
	}
}

func TestRunSwitchHerdr(t *testing.T) {
	for _, directory := range []string{"linked worktree", "worktree subdirectory"} {
		t.Run("opens from "+directory, func(t *testing.T) {
			groveWorkspace := testgit.NewGroveWorkspace(t, "main", "feat-auth")
			workingDirectory := groveWorkspace.Worktrees["main"]
			if directory == "worktree subdirectory" {
				workingDirectory = filepath.Join(workingDirectory, "nested")
				if err := os.MkdirAll(workingDirectory, fs.DirGit); err != nil {
					t.Fatal(err)
				}
			}
			t.Chdir(workingDirectory)
			argumentsPath := stubHerdr(t, "printf '%s\\n' '{\"ok\":true}'\n")

			stdout, stderr, err := executeSwitch(t, "feat-auth", "--herdr")
			if err != nil {
				t.Fatal(err)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if strings.Count(stderr, "\n") != 1 || !strings.Contains(stderr, groveWorkspace.Worktrees["feat-auth"]) || !strings.Contains(stderr, "Herdr") {
				t.Errorf("expected one success line with path and Herdr, got %q", stderr)
			}

			arguments, err := os.ReadFile(argumentsPath) // nolint:gosec // The test creates this marker in t.TempDir().
			if err != nil {
				t.Fatal(err)
			}
			root, err := filepath.EvalSymlinks(groveWorkspace.Dir)
			if err != nil {
				t.Fatal(err)
			}
			worktreePath, err := filepath.EvalSymlinks(groveWorkspace.Worktrees["feat-auth"])
			if err != nil {
				t.Fatal(err)
			}

			want := strings.Join([]string{"worktree", "open", "--cwd", root, "--path", worktreePath, "--focus", "--label", "feat auth", ""}, "\n")
			if string(arguments) != want {
				t.Errorf("herdr arguments = %q, want %q", arguments, want)
			}
		})
	}

	t.Run("opens a detached worktree without a label", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main")
		t.Chdir(groveWorkspace.Dir)
		worktreePath := filepath.Join(groveWorkspace.Dir, "parked")
		if err := git.CreateWorktree(groveWorkspace.BareDir, worktreePath, git.CreateWorktreeOptions{Branch: "main", Detach: true}, true); err != nil {
			t.Fatal(err)
		}
		argumentsPath := stubHerdr(t, "printf '%s\\n' '{\"ok\":true}'\n")

		if _, _, err := executeSwitch(t, "parked", "--herdr"); err != nil {
			t.Fatal(err)
		}

		arguments, err := os.ReadFile(argumentsPath) // nolint:gosec // The test creates this marker in t.TempDir().
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(arguments), "--label") {
			t.Errorf("herdr arguments = %q, want no --label for a detached worktree", arguments)
		}
	})

	t.Run("keeps the label of a PR worktree", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main", "feat-auth")
		t.Chdir(groveWorkspace.Dir)
		if err := git.SetBranchConfig(groveWorkspace.BareDir, "feat-auth", "grovePr", "42"); err != nil {
			t.Fatal(err)
		}
		argumentsPath := stubHerdr(t, "printf '%s\\n' '{\"ok\":true}'\n")

		if _, _, err := executeSwitch(t, "feat-auth", "--herdr"); err != nil {
			t.Fatal(err)
		}

		arguments, err := os.ReadFile(argumentsPath) // nolint:gosec // The test creates this marker in t.TempDir().
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(arguments), "--label") {
			t.Errorf("herdr arguments = %q, want no --label for a PR worktree", arguments)
		}
	})

	t.Run("prints the path without the flag", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main")
		t.Chdir(groveWorkspace.Dir)
		argumentsPath := stubHerdr(t, "exit 0\n")

		stdout, stderr, err := executeSwitch(t, "main")
		if err != nil || stdout != groveWorkspace.Worktrees["main"]+"\n" || stderr != "" {
			t.Fatalf("switch = (%q, %q, %v)", stdout, stderr, err)
		}
		assertHerdrNotCalled(t, argumentsPath)
	})

	t.Run("opens a unique substring match", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main", "feat-auth")
		t.Chdir(groveWorkspace.Dir)
		argumentsPath := stubHerdr(t, "exit 0\n")

		stdout, _, err := executeSwitch(t, "auth", "--herdr")
		if err != nil || stdout != "" {
			t.Fatalf("switch = (%q, %v)", stdout, err)
		}
		worktreePath, err := filepath.EvalSymlinks(groveWorkspace.Worktrees["feat-auth"])
		if err != nil {
			t.Fatal(err)
		}
		arguments, err := os.ReadFile(argumentsPath) // nolint:gosec // The test creates this marker in t.TempDir().
		if err != nil || !strings.Contains(string(arguments), "--path\n"+worktreePath+"\n") {
			t.Fatalf("herdr arguments = %q, error = %v", arguments, err)
		}
	})

	t.Run("rejects ambiguous and missing targets before calling herdr", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main", "feat-auth", "auth-fix")
		t.Chdir(groveWorkspace.Dir)
		argumentsPath := stubHerdr(t, "exit 0\n")

		for target, want := range map[string]string{
			"auth":    `ambiguous target "auth": auth-fix, feat-auth`,
			"missing": "worktree not found: missing",
		} {
			stdout, stderr, err := executeSwitch(t, target, "--herdr")
			if err == nil || err.Error() != want || stdout != "" || stderr != "" {
				t.Errorf("switch %s = (%q, %q, %v), want %q", target, stdout, stderr, err, want)
			}
		}
		assertHerdrNotCalled(t, argumentsPath)
	})

	t.Run("names the missing binary and resolved path", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("PATH isolation via symlink is not portable to Windows")
		}
		groveWorkspace := testgit.NewGroveWorkspace(t, "main")
		t.Chdir(groveWorkspace.Dir)
		gitPath, err := exec.LookPath("git")
		if err != nil {
			t.Fatal(err)
		}
		directory := t.TempDir()
		if err := os.Symlink(gitPath, filepath.Join(directory, filepath.Base(gitPath))); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", directory)

		stdout, stderr, err := executeSwitch(t, "main", "--herdr")
		if err == nil || !strings.Contains(err.Error(), "herdr") || !strings.Contains(err.Error(), groveWorkspace.Worktrees["main"]) || stdout != "" || stderr != "" {
			t.Fatalf("switch = (%q, %q, %v), want missing herdr error with path", stdout, stderr, err)
		}
	})

	t.Run("includes trimmed stderr and the resolved path on failure", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main")
		t.Chdir(groveWorkspace.Dir)
		stubHerdr(t, "printf '%s\\n' '{\"ok\":false}'\nprintf '  {\"error\":\"unavailable\"}\\n' >&2\nexit 1\n")
		worktreePath, err := filepath.EvalSymlinks(groveWorkspace.Worktrees["main"])
		if err != nil {
			t.Fatal(err)
		}

		stdout, stderr, err := executeSwitch(t, "main", "--herdr")
		if err == nil || !strings.Contains(err.Error(), `{"error":"unavailable"}`) || !strings.Contains(err.Error(), worktreePath) || stdout != "" || stderr != "" {
			t.Fatalf("switch = (%q, %q, %v), want herdr failure with path and stderr", stdout, stderr, err)
		}
		if strings.Contains(err.Error(), "\n") || strings.Contains(err.Error(), "  {") {
			t.Errorf("stderr was not trimmed: %q", err)
		}
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			t.Errorf("expected wrapped exit error, got %v", err)
		}
	})
}

func TestNewSwitchCmdHerdr(t *testing.T) {
	t.Run("registers a local flag with help text", func(t *testing.T) {
		command := NewSwitchCmd()
		flag := command.Flags().Lookup("herdr")
		if flag == nil || flag.Usage == "" {
			t.Fatal("expected local herdr flag with help text")
		}
		if command.PersistentFlags().Lookup("herdr") != nil {
			t.Fatal("herdr must not be persistent")
		}
	})

	t.Run("does not expose the flag on shell init", func(t *testing.T) {
		command := NewSwitchCmd()
		command.SetArgs([]string{"shell-init", "--herdr"})
		command.SilenceErrors = true
		command.SilenceUsage = true

		if err := command.Execute(); err == nil || !strings.Contains(err.Error(), "unknown flag: --herdr") {
			t.Fatalf("expected unknown flag, got %v", err)
		}
	})
}

func TestShellSwitch(t *testing.T) {
	for _, shell := range []string{"sh", "fish"} {
		t.Run(shell+" leaves the directory unchanged with empty output", func(t *testing.T) {
			shellPath, err := exec.LookPath(shell)
			if err != nil {
				t.Skipf("%s is not installed", shell)
			}
			directory := t.TempDir()
			testutil.WriteFileMode(t, filepath.Join(directory, "grove"), "#!/bin/sh\n[ \"$*\" = 'switch main --herdr' ] || exit 1\nexit 0\n", fs.FileExec)
			t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
			wrapper, err := filepath.Abs(filepath.Join("shell", "grove."+shell))
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("GROVE_WRAPPER", wrapper)
			script := `. "$GROVE_WRAPPER"
original_directory="$PWD"
grove switch main --herdr || exit 1
[ "$PWD" = "$original_directory" ] || exit 2
`
			if shell == "fish" {
				script = `source "$GROVE_WRAPPER"
set original_directory "$PWD"
grove switch main --herdr; or exit 1
test "$PWD" = "$original_directory"; or exit 2
`
			}

			command := exec.Command(shellPath, "-c", script) // nolint:gosec // The shell runs only the fixed test script.
			command.Dir = directory
			output, err := command.CombinedOutput()
			if err != nil || len(output) != 0 {
				t.Fatalf("wrapper output = %q, error = %v", output, err)
			}
		})
	}
}

func TestShellPowerShell(t *testing.T) {
	t.Run("does not print the switch exit code", func(t *testing.T) {
		switchBlock, _, _ := strings.Cut(shellPowerShell, "} elseif")

		if strings.Contains(switchBlock, "return $LASTEXITCODE") || strings.Contains(switchBlock, "\n            $LASTEXITCODE") {
			t.Fatal("switch must leave LASTEXITCODE set without printing it")
		}
	})
}

func TestRunRemoveHerdr(t *testing.T) {
	t.Run("closes only the workspaces of removed worktrees", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main", "feat-a", "feat-b", "feat-c")
		t.Chdir(groveWorkspace.Worktrees["main"])
		testutil.WriteFile(t, filepath.Join(groveWorkspace.Worktrees["feat-b"], "dirty.txt"), "dirty")
		if runtime.GOOS == "windows" {
			t.Skip("symlinks need privileges on Windows")
		}
		// Herdr reports feat-a through a symlinked root; grove must still match it.
		link := filepath.Join(t.TempDir(), "link")
		if err := os.Symlink(groveWorkspace.Dir, link); err != nil {
			t.Fatal(err)
		}
		callsPath := stubHerdrCalls(t, herdrListScript(t, map[string]string{
			groveWorkspace.Worktrees["main"]:   "w0",
			filepath.Join(link, "feat-a"):      "w1",
			groveWorkspace.Worktrees["feat-b"]: "w2",
			groveWorkspace.Worktrees["feat-c"]: "",
		}), "printf '%s\\n' '{\"ok\":true}'")

		stdout, _, err := executeCommand(t, NewRemoveCmd(), "feat-a", "feat-b", "feat-c", "--herdr")
		if err == nil || !strings.Contains(err.Error(), "feat-b") {
			t.Fatalf("remove error = %v, want refused feat-b", err)
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
		root, err := filepath.EvalSymlinks(groveWorkspace.Dir)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"worktree list --cwd " + root, "workspace close w1"}
		if calls := readHerdrCalls(t, callsPath); strings.Join(calls, "|") != strings.Join(want, "|") {
			t.Errorf("herdr calls = %q, want %q", calls, want)
		}
		testutil.AssertPathExists(t, groveWorkspace.Worktrees["feat-b"])
	})

	t.Run("closes the current workspace last", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main", "feat-a", "feat-b")
		t.Chdir(groveWorkspace.Dir)
		callsPath := stubHerdrCalls(t, herdrListScript(t, map[string]string{
			groveWorkspace.Worktrees["feat-a"]: "w1",
			groveWorkspace.Worktrees["feat-b"]: "w2",
		}), "exit 0")
		t.Setenv("HERDR_WORKSPACE_ID", "w1")

		if _, _, err := executeCommand(t, NewRemoveCmd(), "feat-a", "feat-b", "--herdr"); err != nil {
			t.Fatal(err)
		}

		calls := readHerdrCalls(t, callsPath)
		if len(calls) != 3 || calls[1] != "workspace close w2" || calls[2] != "workspace close w1" {
			t.Errorf("herdr calls = %q, want w2 closed before the current w1", calls)
		}
	})

	t.Run("warns when a close fails after removal", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main", "feat-a")
		t.Chdir(groveWorkspace.Dir)
		stubHerdrCalls(t, herdrListScript(t, map[string]string{groveWorkspace.Worktrees["feat-a"]: "w1"}), "printf 'no such workspace\\n' >&2\nexit 1")

		_, stderr, err := executeCommand(t, NewRemoveCmd(), "feat-a", "--herdr")
		if err != nil {
			t.Fatalf("remove = %v, want success", err)
		}
		if !strings.Contains(stderr, "w1") || !strings.Contains(stderr, "no such workspace") {
			t.Errorf("stderr = %q, want close warning", stderr)
		}
		if _, err := os.Stat(groveWorkspace.Worktrees["feat-a"]); !os.IsNotExist(err) {
			t.Errorf("feat-a should be removed: %v", err)
		}
	})

	t.Run("removes nothing when the workspace list fails", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main", "feat-a")
		t.Chdir(groveWorkspace.Dir)
		callsPath := stubHerdrCalls(t, "printf 'server down\\n' >&2\nexit 1", "exit 0")

		_, _, err := executeCommand(t, NewRemoveCmd(), "feat-a", "--herdr")
		if err == nil || !strings.Contains(err.Error(), "server down") {
			t.Fatalf("remove = %v, want list failure", err)
		}
		testutil.AssertPathExists(t, groveWorkspace.Worktrees["feat-a"])
		if calls := readHerdrCalls(t, callsPath); len(calls) != 1 {
			t.Errorf("herdr calls = %q, want only the list", calls)
		}
	})

	t.Run("removes nothing when herdr is missing", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("PATH isolation via symlink is not portable to Windows")
		}
		groveWorkspace := testgit.NewGroveWorkspace(t, "main", "feat-a")
		t.Chdir(groveWorkspace.Dir)
		gitPath, err := exec.LookPath("git")
		if err != nil {
			t.Fatal(err)
		}
		directory := t.TempDir()
		if err := os.Symlink(gitPath, filepath.Join(directory, filepath.Base(gitPath))); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", directory)

		_, _, err = executeCommand(t, NewRemoveCmd(), "feat-a", "--herdr")
		if err == nil || !strings.Contains(err.Error(), "herdr") {
			t.Fatalf("remove = %v, want missing herdr error", err)
		}
		testutil.AssertPathExists(t, groveWorkspace.Worktrees["feat-a"])
	})

	t.Run("never calls herdr without the flag", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main", "feat-a")
		t.Chdir(groveWorkspace.Dir)
		callsPath := stubHerdrCalls(t, herdrListScript(t, map[string]string{groveWorkspace.Worktrees["feat-a"]: "w1"}), "exit 0")

		if _, _, err := executeCommand(t, NewRemoveCmd(), "feat-a"); err != nil {
			t.Fatal(err)
		}
		if calls := readHerdrCalls(t, callsPath); calls != nil {
			t.Errorf("herdr calls = %q, want none", calls)
		}
	})
}

func TestRunPruneHerdr(t *testing.T) {
	setup := func(t *testing.T) (groveWorkspace *testgit.GroveWorkspace, callsPath string) {
		t.Helper()

		groveWorkspace = testgit.NewGroveWorkspace(t, "main")
		t.Chdir(groveWorkspace.Worktrees["main"])
		for _, name := range []string{"parked", "idle"} {
			path := filepath.Join(groveWorkspace.Dir, name)
			if err := git.CreateWorktree(groveWorkspace.BareDir, path, git.CreateWorktreeOptions{Branch: "main", Detach: true}, true); err != nil {
				t.Fatal(err)
			}
		}
		callsPath = stubHerdrCalls(t, herdrListScript(t, map[string]string{
			groveWorkspace.Worktrees["main"]:            "w0",
			filepath.Join(groveWorkspace.Dir, "parked"): "w1",
			filepath.Join(groveWorkspace.Dir, "idle"):   "",
		}), "printf '%s\\n' '{\"ok\":true}'")

		return groveWorkspace, callsPath
	}

	t.Run("dry run marks workspaces without closing them", func(t *testing.T) {
		_, callsPath := setup(t)

		_, stderr, err := executeCommand(t, NewPruneCmd(), "--detached", "--herdr")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(stderr, "(closes Herdr workspace") != 1 {
			t.Errorf("stderr = %q, want one marked candidate", stderr)
		}
		for _, line := range strings.Split(stderr, "\n") {
			marked := strings.HasSuffix(line, "(closes Herdr workspace w1)")
			if strings.Contains(line, "parked [") != marked || strings.Contains(line, "idle [") && marked {
				t.Errorf("stderr line %q, want only parked marked", line)
			}
		}
		if calls := readHerdrCalls(t, callsPath); len(calls) != 1 {
			t.Errorf("herdr calls = %q, want only the list", calls)
		}
	})

	t.Run("dry run JSON marks workspaces without closing them", func(t *testing.T) {
		_, callsPath := setup(t)

		stdout, _, err := executeCommand(t, NewPruneCmd(), "--detached", "--herdr", "--json")
		if err != nil {
			t.Fatal(err)
		}
		var candidates []map[string]any
		if err := json.Unmarshal([]byte(stdout), &candidates); err != nil {
			t.Fatalf("stdout = %q: %v", stdout, err)
		}
		if len(candidates) != 2 {
			t.Fatalf("candidates = %v, want parked and idle", candidates)
		}
		for _, candidate := range candidates {
			id, marked := candidate["herdr_workspace_id"]
			if candidate["name"] == "parked" && id != "w1" || candidate["name"] == "idle" && marked {
				t.Errorf("candidate %v, want parked w1 and idle without herdr_workspace_id", candidate)
			}
		}
		if calls := readHerdrCalls(t, callsPath); len(calls) != 1 {
			t.Errorf("herdr calls = %q, want only the list", calls)
		}
	})

	t.Run("commit closes workspaces of pruned worktrees", func(t *testing.T) {
		groveWorkspace, callsPath := setup(t)

		stdout, _, err := executeCommand(t, NewPruneCmd(), "--detached", "--herdr", "--commit")
		if err != nil {
			t.Fatal(err)
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
		root, err := filepath.EvalSymlinks(groveWorkspace.Dir)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"worktree list --cwd " + root, "workspace close w1"}
		if calls := readHerdrCalls(t, callsPath); strings.Join(calls, "|") != strings.Join(want, "|") {
			t.Errorf("herdr calls = %q, want %q", calls, want)
		}
		if _, err := os.Stat(filepath.Join(groveWorkspace.Dir, "parked")); !os.IsNotExist(err) {
			t.Errorf("parked should be removed: %v", err)
		}
	})

	t.Run("never calls herdr without the flag", func(t *testing.T) {
		_, callsPath := setup(t)

		stdout, _, err := executeCommand(t, NewPruneCmd(), "--detached", "--json")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(stdout, "herdr_workspace_id") {
			t.Errorf("stdout = %q, want no herdr field", stdout)
		}
		if _, _, err := executeCommand(t, NewPruneCmd(), "--detached", "--commit"); err != nil {
			t.Fatal(err)
		}
		if calls := readHerdrCalls(t, callsPath); calls != nil {
			t.Errorf("herdr calls = %q, want none", calls)
		}
	})

	t.Run("commit removes nothing when the workspace list fails", func(t *testing.T) {
		groveWorkspace, _ := setup(t)
		callsPath := stubHerdrCalls(t, "printf 'server down\\n' >&2\nexit 1", "exit 0")

		_, _, err := executeCommand(t, NewPruneCmd(), "--detached", "--herdr", "--commit")
		if err == nil || !strings.Contains(err.Error(), "server down") {
			t.Fatalf("prune = %v, want list failure", err)
		}
		registered, err := git.ListWorktrees(groveWorkspace.BareDir)
		if err != nil {
			t.Fatal(err)
		}
		if len(registered) != 3 {
			t.Errorf("registered worktrees = %q, want main, parked, and idle", registered)
		}
		for _, name := range []string{"parked", "idle"} {
			testutil.AssertPathExists(t, filepath.Join(groveWorkspace.Dir, name))
		}
		if calls := readHerdrCalls(t, callsPath); len(calls) != 1 {
			t.Errorf("herdr calls = %q, want only the list", calls)
		}
	})

	t.Run("skipped candidates keep their workspace", func(t *testing.T) {
		groveWorkspace, callsPath := setup(t)
		testutil.WriteFile(t, filepath.Join(groveWorkspace.Dir, "parked", "dirty.txt"), "dirty")

		_, stderr, err := executeCommand(t, NewPruneCmd(), "--detached", "--herdr")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(stderr, "closes Herdr workspace") {
			t.Errorf("stderr = %q, want no marked candidate", stderr)
		}
		if _, _, err := executeCommand(t, NewPruneCmd(), "--detached", "--herdr", "--commit"); err != nil {
			t.Fatal(err)
		}
		if calls := readHerdrCalls(t, callsPath); strings.Contains(strings.Join(calls, "|"), "workspace close") {
			t.Errorf("herdr calls = %q, want no close", calls)
		}
		testutil.AssertPathExists(t, filepath.Join(groveWorkspace.Dir, "parked"))

		if _, _, err := executeCommand(t, NewPruneCmd(), "--detached", "--herdr", "--commit", "--force"); err != nil {
			t.Fatal(err)
		}
		if calls := readHerdrCalls(t, callsPath); calls[len(calls)-1] != "workspace close w1" {
			t.Errorf("herdr calls = %q, want w1 closed with --force", calls)
		}
	})

	t.Run("commit closes the workspace of a path-gone worktree under a symlinked root", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlinks need privileges on Windows")
		}
		groveWorkspace := testgit.NewGroveWorkspace(t, "main")
		t.Chdir(groveWorkspace.Worktrees["main"])
		gone := filepath.Join(groveWorkspace.Dir, "gone")
		if err := git.CreateWorktree(groveWorkspace.BareDir, gone, git.CreateWorktreeOptions{Branch: "main", Detach: true}, true); err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(gone); err != nil {
			t.Fatal(err)
		}
		// Herdr spells the missing worktree through an alias of the root git recorded.
		link := filepath.Join(t.TempDir(), "link")
		if err := os.Symlink(groveWorkspace.Dir, link); err != nil {
			t.Fatal(err)
		}
		callsPath := stubHerdrCalls(t, herdrListScript(t, map[string]string{filepath.Join(link, "gone"): "w1"}), "exit 0")

		if _, _, err := executeCommand(t, NewPruneCmd(), "--herdr", "--commit"); err != nil {
			t.Fatal(err)
		}
		if calls := readHerdrCalls(t, callsPath); len(calls) != 2 || calls[1] != "workspace close w1" {
			t.Errorf("herdr calls = %q, want w1 closed", calls)
		}
	})
}
