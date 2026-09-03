//go:build integration

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
	"github.com/sqve/grove/internal/fs"
)

func TestScript(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/script",
		Setup: func(env *testscript.Env) error {
			homeDir := filepath.Join(env.WorkDir, ".home")
			if err := os.MkdirAll(homeDir, fs.DirGit); err != nil {
				return err
			}
			env.Vars = append(env.Vars, "HOME="+homeDir)
			gitConfigPath := filepath.Join(homeDir, ".gitconfig")
			gitConfigContent := `[init]
	defaultBranch = main
[advice]
	defaultBranchName = false
[user]
	name = Test
	email = test@example.com
[commit]
	gpgsign = false
`
			if err := os.WriteFile(gitConfigPath, []byte(gitConfigContent), fs.FileGit); err != nil {
				return err
			}
			env.Vars = append(env.Vars, "GIT_CONFIG_GLOBAL="+gitConfigPath)
			env.Vars = append(env.Vars, "GIT_CONFIG_SYSTEM=/dev/null")
			env.Vars = append(env.Vars, "GIT_CONFIG_NOSYSTEM=1")
			env.Vars = append(env.Vars, "XDG_CONFIG_HOME="+homeDir)

			readmePath := filepath.Join(env.WorkDir, "README.md")
			if err := os.WriteFile(readmePath, []byte("# Test\n"), fs.FileGit); err != nil {
				return err
			}

			return nil
		},
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"setup_workspace":      cmdSetupWorkspace,
			"assert_worktree":      cmdAssertWorktree,
			"assert_branch":        cmdAssertBranch,
			"assert_outside_error": cmdAssertOutsideError,
		},
		Condition: func(cond string) (bool, error) {
			switch cond {
			case "slow":
				return os.Getenv("GROVE_SKIP_SLOW") == "", nil
			}
			return false, nil
		},
	})
}

// cmdSetupWorkspace creates a grove workspace with optional branches.
// Usage: setup_workspace [branch...]
// Creates testrepo, clones to workspace, changes to workspace/main.
func cmdSetupWorkspace(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("setup_workspace does not support negation")
	}

	workDir := ts.Getenv("WORK")
	repoDir := filepath.Join(workDir, "testrepo")
	if err := os.MkdirAll(repoDir, fs.DirGit); err != nil {
		ts.Fatalf("mkdir testrepo: %v", err)
	}

	gitRun(ts, repoDir, "init")

	readmeSrc := filepath.Join(workDir, "README.md")
	readmeDst := filepath.Join(repoDir, "README.md")
	content, err := os.ReadFile(readmeSrc)
	if err != nil {
		ts.Fatalf("read README.md: %v", err)
	}
	if err := os.WriteFile(readmeDst, content, fs.FileGit); err != nil {
		ts.Fatalf("write README.md: %v", err)
	}

	gitRun(ts, repoDir, "add", ".")
	gitRun(ts, repoDir, "commit", "-m", "initial commit")

	for _, branch := range args {
		gitRun(ts, repoDir, "checkout", "-b", branch)
	}
	gitRun(ts, repoDir, "checkout", "main")

	wsDir := filepath.Join(workDir, "workspace")
	if err := os.MkdirAll(wsDir, fs.DirGit); err != nil {
		ts.Fatalf("mkdir workspace: %v", err)
	}

	cmd := exec.Command("grove", "clone", "file://"+repoDir, wsDir) // nolint:gosec
	cmd.Dir = workDir
	cmd.Env = testEnv(ts)
	if out, err := cmd.CombinedOutput(); err != nil {
		ts.Fatalf("grove clone: %v\n%s", err, out)
	}

	mainDir := filepath.Join(wsDir, "main")
	if err := ts.Chdir(mainDir); err != nil {
		ts.Fatalf("chdir to workspace/main: %v", err)
	}
}

// cmdAssertWorktree verifies worktree exists and is on expected branch.
// Usage: assert_worktree <dir-name> <branch>
func cmdAssertWorktree(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) != 2 {
		ts.Fatalf("usage: assert_worktree <dir-name> <branch>")
	}

	dirName := args[0]
	expectedBranch := args[1]

	cwd, err := os.Getwd()
	if err != nil {
		ts.Fatalf("getwd: %v", err)
	}
	worktreePath := filepath.Join(filepath.Dir(cwd), dirName)

	exists := true
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		exists = false
	}

	if neg {
		if exists {
			ts.Fatalf("expected worktree %s to not exist", dirName)
		}
		return
	}

	if !exists {
		ts.Fatalf("expected worktree %s to exist", dirName)
	}

	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD") // nolint:gosec
	cmd.Dir = worktreePath
	out, err := cmd.Output()
	if err != nil {
		ts.Fatalf("git rev-parse in %s: %v", dirName, err)
	}

	actualBranch := strings.TrimSpace(string(out))
	if actualBranch != expectedBranch {
		ts.Fatalf("worktree %s: expected branch %q, got %q", dirName, expectedBranch, actualBranch)
	}
}

// cmdAssertBranch verifies branch existence.
// Usage: assert_branch exists|deleted <branch>
func cmdAssertBranch(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) != 2 {
		ts.Fatalf("usage: assert_branch exists|deleted <branch>")
	}

	action := args[0]
	branch := args[1]

	cwd, _ := os.Getwd()
	cmd := exec.Command("git", "branch", "--list", branch) // nolint:gosec
	cmd.Dir = cwd
	out, _ := cmd.Output()
	exists := strings.TrimSpace(string(out)) != ""

	switch action {
	case "exists":
		if neg {
			if exists {
				ts.Fatalf("expected branch %q to not exist", branch)
			}
		} else {
			if !exists {
				ts.Fatalf("expected branch %q to exist", branch)
			}
		}
	case "deleted":
		if neg {
			if !exists {
				ts.Fatalf("expected branch %q to exist (not deleted)", branch)
			}
		} else {
			if exists {
				ts.Fatalf("expected branch %q to be deleted", branch)
			}
		}
	default:
		ts.Fatalf("assert_branch: unknown action %q (use 'exists' or 'deleted')", action)
	}
}

// cmdAssertOutsideError runs a command and expects "not in a grove workspace" error.
// Usage: assert_outside_error <command> [args...]
func cmdAssertOutsideError(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) < 1 {
		ts.Fatalf("usage: assert_outside_error <command> [args...]")
	}

	workDir := ts.Getenv("WORK")
	outsideDir := filepath.Join(workDir, "outside")
	if err := os.MkdirAll(outsideDir, fs.DirGit); err != nil {
		ts.Fatalf("mkdir outside: %v", err)
	}

	cmd := exec.Command(args[0], args[1:]...) // nolint:gosec
	cmd.Dir = outsideDir
	out, err := cmd.CombinedOutput()

	if neg {
		if err != nil && strings.Contains(string(out), "not in a grove workspace") {
			ts.Fatalf("expected command to succeed outside workspace")
		}
		return
	}

	if err == nil {
		ts.Fatalf("expected command to fail outside workspace")
	}
	if !strings.Contains(string(out), "not in a grove workspace") {
		ts.Fatalf("expected 'not in a grove workspace' error, got: %s", out)
	}
}

func gitRun(ts *testscript.TestScript, dir string, args ...string) {
	cmd := exec.Command("git", args...) // nolint:gosec
	cmd.Dir = dir
	cmd.Env = testEnv(ts)
	if out, err := cmd.CombinedOutput(); err != nil {
		ts.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// testEnv returns the environment for running commands in testscript.
// This ensures git config and other settings are properly inherited.
func testEnv(ts *testscript.TestScript) []string {
	return []string{
		"HOME=" + ts.Getenv("HOME"),
		"PATH=" + ts.Getenv("PATH"),
		"GIT_CONFIG_GLOBAL=" + ts.Getenv("GIT_CONFIG_GLOBAL"),
		"GIT_CONFIG_SYSTEM=" + ts.Getenv("GIT_CONFIG_SYSTEM"),
		"GIT_CONFIG_NOSYSTEM=" + ts.Getenv("GIT_CONFIG_NOSYSTEM"),
		"XDG_CONFIG_HOME=" + ts.Getenv("XDG_CONFIG_HOME"),
		"TMPDIR=" + ts.Getenv("TMPDIR"),
	}
}

func fakeGh() {
	args := os.Args[1:]
	logPath := os.Getenv("FAKE_GH_LOG")
	if logPath == "" {
		if len(args) == 1 && args[0] == "--version" {
			fmt.Println("gh version 2.40.0")
			return
		}
		os.Exit(1)
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, fs.FileGit)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := json.NewEncoder(logFile).Encode(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := logFile.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "fake gh: expected a command")
		os.Exit(2)
	}
	command := args[0]
	if len(args) > 1 {
		command = strings.Join(args[:2], " ")
	}
	if command == os.Getenv("FAKE_GH_FAIL") {
		fmt.Fprintln(os.Stderr, os.Getenv("FAKE_GH_STDERR"))
		exitCode, err := strconv.Atoi(os.Getenv("FAKE_GH_EXIT"))
		if err != nil || exitCode == 0 {
			exitCode = 1
		}
		os.Exit(exitCode)
	}

	switch command {
	case "--version":
		fmt.Println("gh version 2.40.0")
		return
	case "auth status":
		return
	case "repo clone":
		if len(args) < 4 || os.Getenv("FAKE_GH_REPO") == "" {
			fmt.Fprintln(os.Stderr, "fake gh: FAKE_GH_REPO is required")
			os.Exit(2)
		}
		cmd := exec.Command("git", "clone", "--bare", os.Getenv("FAKE_GH_REPO"), args[3]) //nolint:gosec
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			os.Exit(1)
		}
		return
	case "pr view", "pr list", "repo view":
		fixtureName := "FAKE_GH_" + strings.ToUpper(strings.ReplaceAll(command, " ", "_"))
		fixture, err := os.ReadFile(os.Getenv(fixtureName))
		if err != nil {
			fmt.Fprintf(os.Stderr, "fake gh: %s: %v\n", fixtureName, err)
			os.Exit(2)
		}
		_, _ = fmt.Fprint(os.Stdout, os.ExpandEnv(string(fixture)))
		return
	default:
		fmt.Fprintf(os.Stderr, "fake gh: unsupported arguments: %s\n", strings.Join(args, " "))
		os.Exit(2)
	}
}

func TestMain(m *testing.M) {
	// Ensure grove binary is available in PATH. While testscript.Main registers
	// "grove" as an in-process command, custom commands like cmdSetupWorkspace
	// use exec.Command("grove", ...) which requires the binary in PATH.
	if _, err := exec.LookPath("grove"); err != nil {
		fmt.Fprintf(os.Stderr, "grove binary not found in PATH; run 'go install' first\n")
		os.Exit(1)
	}

	testscript.Main(m, map[string]func(){
		"grove": main,
		"gh":    fakeGh,
	})
}
