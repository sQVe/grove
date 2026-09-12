package commands

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

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

func executeSwitch(t *testing.T, arguments ...string) (stdoutText, stderrText string, executionError error) {
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
	command := NewSwitchCmd()
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
