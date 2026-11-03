package commands

import (
	"bytes"
	"encoding/json"
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

// testHerdrSpace describes one local space for a herdr-app ls stub.
type testHerdrSpace struct {
	ID       string
	Folder   string
	Terminal string
}

func herdrListJSON(t *testing.T, spaces ...testHerdrSpace) string {
	t.Helper()

	entries := make([]map[string]any, 0, len(spaces))
	for _, space := range spaces {
		entry := map[string]any{
			"id":      space.ID,
			"folder":  space.Folder,
			"machine": nil,
		}
		if space.Terminal != "" {
			entry["tabs"] = []map[string]any{
				{"terminals": []map[string]any{{"terminal": space.Terminal}}},
			}
		}
		entries = append(entries, entry)
	}

	data, err := json.Marshal(map[string]any{"spaces": entries})
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}

// herdrListScript returns a shell command that prints the given spaces as the
// JSON herdr-app ls writes.
func herdrListScript(t *testing.T, spaces ...testHerdrSpace) string {
	t.Helper()

	listPath := filepath.Join(t.TempDir(), "spaces.json")
	testutil.WriteFile(t, listPath, herdrListJSON(t, spaces...))

	return "cat '" + listPath + "'"
}

// writeHerdrAppStub writes a herdr-app shell stub into directory. listScript
// runs for `ls` and doScript for `do`; every invocation's arguments are
// appended to the file named by HERDR_APP_CALLS, one argument per line.
func writeHerdrAppStub(t *testing.T, directory, listScript, doScript string) string {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("shell herdr-app stub is not executable on Windows")
	}

	callsPath := filepath.Join(t.TempDir(), "calls")
	t.Setenv("HERDR_APP_CALLS", callsPath)
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$@\" >> \"$HERDR_APP_CALLS\"\n" +
		"if [ \"$1\" = ls ]; then\n" + listScript + "\nexit 0\nfi\n" +
		doScript
	testutil.WriteFileMode(t, filepath.Join(directory, "herdr-app"), script, fs.FileExec)

	return callsPath
}

// stubHerdrApp creates a temporary directory with a herdr-app stub and a fake
// gh, prepends it to PATH, and returns the calls path.
func stubHerdrApp(t *testing.T, listScript, doScript string) string {
	t.Helper()

	directory := t.TempDir()
	callsPath := writeHerdrAppStub(t, directory, listScript, doScript)
	testutil.WriteFileMode(t, filepath.Join(directory, "gh"), "#!/bin/sh\nexit 1\n", fs.FileExec)
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))

	return callsPath
}

// readHerdrAppCalls groups the recorded argument lines back into one string per
// invocation, for example "do space.select --id w1".
func readHerdrAppCalls(t *testing.T, callsPath string) []string {
	t.Helper()

	lines := readHerdrCalls(t, callsPath)
	if lines == nil {
		return nil
	}
	calls := make([]string, 0, len(lines))
	for i := 0; i < len(lines); {
		if lines[i] == "ls" {
			calls = append(calls, "ls")
			i++
			continue
		}

		end := i + 1
		for end < len(lines) && lines[end] != "ls" && lines[end] != "do" {
			end++
		}
		calls = append(calls, strings.Join(lines[i:end], " "))
		i = end
	}

	return calls
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
		t.Fatalf("herdr-app must not run: marker stat = %v", err)
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

func TestHerdrSpacesLookup(t *testing.T) {
	t.Run("matches paths that differ only in case on Windows", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("paths are case-sensitive outside Windows")
		}

		worktree := filepath.Join(t.TempDir(), "feat-auth")
		spaces := &herdrSpaces{list: []herdrSpace{{ID: "w1", Folder: strings.ToUpper(worktree)}}}

		if id := spaces.lookup(strings.ToLower(worktree)); id != "w1" {
			t.Errorf("lookup = %q, want w1", id)
		}
	})

	t.Run("returns the first of several spaces on one folder", func(t *testing.T) {
		worktree := filepath.Join(t.TempDir(), "feat-auth")
		spaces := &herdrSpaces{list: []herdrSpace{
			{ID: "w1", Folder: worktree},
			{ID: "w2", Folder: worktree},
		}}

		if id := spaces.lookup(worktree); id != "w1" {
			t.Errorf("lookup = %q, want w1", id)
		}
	})
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
			binaryDir := t.TempDir()
			callsPath := writeHerdrAppStub(t, binaryDir, herdrListScript(t), "")
			t.Setenv("PATH", binaryDir+string(os.PathListSeparator)+os.Getenv("PATH"))

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

			worktreePath, err := filepath.EvalSymlinks(groveWorkspace.Worktrees["feat-auth"])
			if err != nil {
				t.Fatal(err)
			}
			want := []string{"ls", "do space.new --name feat auth --folder " + worktreePath}
			if calls := readHerdrAppCalls(t, callsPath); strings.Join(calls, "|") != strings.Join(want, "|") {
				t.Errorf("herdr-app calls = %q, want %q", calls, want)
			}
		})
	}

	t.Run("selects the space already open on the worktree", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main", "feat-auth")
		t.Chdir(groveWorkspace.Dir)
		worktreePath, err := filepath.EvalSymlinks(groveWorkspace.Worktrees["feat-auth"])
		if err != nil {
			t.Fatal(err)
		}
		binaryDir := t.TempDir()
		callsPath := writeHerdrAppStub(t, binaryDir, herdrListScript(t, testHerdrSpace{ID: "sp9", Folder: worktreePath}), "")
		t.Setenv("PATH", binaryDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		if _, _, err := executeSwitch(t, "feat-auth", "--herdr"); err != nil {
			t.Fatal(err)
		}

		want := []string{"ls", "do space.select --id sp9"}
		if calls := readHerdrAppCalls(t, callsPath); strings.Join(calls, "|") != strings.Join(want, "|") {
			t.Errorf("herdr-app calls = %q, want %q", calls, want)
		}
	})

	t.Run("names a detached worktree after its directory", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main")
		t.Chdir(groveWorkspace.Dir)
		worktreePath := filepath.Join(groveWorkspace.Dir, "parked")
		if err := git.CreateWorktree(groveWorkspace.BareDir, worktreePath, git.CreateWorktreeOptions{Branch: "main", Detach: true}, true); err != nil {
			t.Fatal(err)
		}
		binaryDir := t.TempDir()
		callsPath := writeHerdrAppStub(t, binaryDir, herdrListScript(t), "")
		t.Setenv("PATH", binaryDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		if _, _, err := executeSwitch(t, "parked", "--herdr"); err != nil {
			t.Fatal(err)
		}

		canonical, err := filepath.EvalSymlinks(worktreePath)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"ls", "do space.new --name parked --folder " + canonical}
		if calls := readHerdrAppCalls(t, callsPath); strings.Join(calls, "|") != strings.Join(want, "|") {
			t.Errorf("herdr-app calls = %q, want a directory-named new space %q", calls, want)
		}
	})

	t.Run("prints the path without the flag", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main")
		t.Chdir(groveWorkspace.Dir)
		binaryDir := t.TempDir()
		callsPath := writeHerdrAppStub(t, binaryDir, herdrListScript(t), "")
		t.Setenv("PATH", binaryDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		stdout, stderr, err := executeSwitch(t, "main")
		if err != nil || stdout != groveWorkspace.Worktrees["main"]+"\n" || stderr != "" {
			t.Fatalf("switch = (%q, %q, %v)", stdout, stderr, err)
		}
		assertHerdrNotCalled(t, callsPath)
	})

	t.Run("opens a unique substring match", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main", "feat-auth")
		t.Chdir(groveWorkspace.Dir)
		binaryDir := t.TempDir()
		callsPath := writeHerdrAppStub(t, binaryDir, herdrListScript(t), "")
		t.Setenv("PATH", binaryDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		stdout, _, err := executeSwitch(t, "auth", "--herdr")
		if err != nil || stdout != "" {
			t.Fatalf("switch = (%q, %v)", stdout, err)
		}
		worktreePath, err := filepath.EvalSymlinks(groveWorkspace.Worktrees["feat-auth"])
		if err != nil {
			t.Fatal(err)
		}
		want := "do space.new --name feat auth --folder " + worktreePath
		if calls := readHerdrAppCalls(t, callsPath); len(calls) != 2 || calls[1] != want {
			t.Fatalf("herdr-app calls = %q, want %q", calls, want)
		}
	})

	t.Run("rejects ambiguous and missing targets before calling herdr-app", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main", "feat-auth", "auth-fix")
		t.Chdir(groveWorkspace.Dir)
		binaryDir := t.TempDir()
		callsPath := writeHerdrAppStub(t, binaryDir, herdrListScript(t), "")
		t.Setenv("PATH", binaryDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		for target, want := range map[string]string{
			"auth":    `ambiguous target "auth": auth-fix, feat-auth`,
			"missing": "worktree not found: missing",
		} {
			stdout, stderr, err := executeSwitch(t, target, "--herdr")
			if err == nil || err.Error() != want || stdout != "" || stderr != "" {
				t.Errorf("switch %s = (%q, %q, %v), want %q", target, stdout, stderr, err, want)
			}
		}
		assertHerdrNotCalled(t, callsPath)
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
		if err == nil || !strings.Contains(err.Error(), "herdr-app") || !strings.Contains(err.Error(), groveWorkspace.Worktrees["main"]) || stdout != "" || stderr != "" {
			t.Fatalf("switch = (%q, %q, %v), want missing herdr-app error with path", stdout, stderr, err)
		}
	})

	t.Run("uses the error object a failed do prints on stdout", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main")
		t.Chdir(groveWorkspace.Dir)
		binaryDir := t.TempDir()
		callsPath := writeHerdrAppStub(t, binaryDir, herdrListScript(t),
			"printf '%s\\n' '{\"error\":\"space name taken\"}'\nexit 1\n")
		t.Setenv("PATH", binaryDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		worktreePath, err := filepath.EvalSymlinks(groveWorkspace.Worktrees["main"])
		if err != nil {
			t.Fatal(err)
		}

		stdout, stderr, err := executeSwitch(t, "main", "--herdr")
		if err == nil || !strings.Contains(err.Error(), "space name taken") || !strings.Contains(err.Error(), worktreePath) || stdout != "" || stderr != "" {
			t.Fatalf("switch = (%q, %q, %v), want the do error object with path", stdout, stderr, err)
		}
		if calls := readHerdrAppCalls(t, callsPath); len(calls) != 2 {
			t.Errorf("herdr-app calls = %q, want ls then do", calls)
		}
	})

	t.Run("uses the stderr of a no-window exit 2", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main")
		t.Chdir(groveWorkspace.Dir)
		binaryDir := t.TempDir()
		writeHerdrAppStub(t, binaryDir, "printf 'herdr-app: no Herdr window at /run/herdr.sock: connection refused\\n' >&2\nexit 2\n", "")
		t.Setenv("PATH", binaryDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		worktreePath, err := filepath.EvalSymlinks(groveWorkspace.Worktrees["main"])
		if err != nil {
			t.Fatal(err)
		}

		stdout, stderr, err := executeSwitch(t, "main", "--herdr")
		if err == nil || !strings.Contains(err.Error(), "no Herdr window at /run/herdr.sock") || !strings.Contains(err.Error(), worktreePath) || stdout != "" || stderr != "" {
			t.Fatalf("switch = (%q, %q, %v), want the no-window stderr with path", stdout, stderr, err)
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
	t.Run("closes only the spaces of removed worktrees", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main", "feat-a", "feat-b", "feat-c")
		t.Chdir(groveWorkspace.Worktrees["main"])
		testutil.WriteFile(t, filepath.Join(groveWorkspace.Worktrees["feat-b"], "dirty.txt"), "dirty")
		if runtime.GOOS == "windows" {
			t.Skip("symlinks need privileges on Windows")
		}
		// herdr-app reports feat-a through a symlinked root; grove must still match it.
		link := filepath.Join(t.TempDir(), "link")
		if err := os.Symlink(groveWorkspace.Dir, link); err != nil {
			t.Fatal(err)
		}
		callsPath := stubHerdrApp(t, herdrListScript(t,
			testHerdrSpace{ID: "w0", Folder: groveWorkspace.Worktrees["main"]},
			testHerdrSpace{ID: "w1", Folder: filepath.Join(link, "feat-a")},
			testHerdrSpace{ID: "w2", Folder: groveWorkspace.Worktrees["feat-b"]},
		), "")

		stdout, _, err := executeCommand(t, NewRemoveCmd(), "feat-a", "feat-b", "feat-c", "--herdr")
		if err == nil || !strings.Contains(err.Error(), "feat-b") {
			t.Fatalf("remove error = %v, want refused feat-b", err)
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
		want := []string{"ls", "do space.close --id w1 --confirmed true"}
		if calls := readHerdrAppCalls(t, callsPath); strings.Join(calls, "|") != strings.Join(want, "|") {
			t.Errorf("herdr-app calls = %q, want %q", calls, want)
		}
		testutil.AssertPathExists(t, groveWorkspace.Worktrees["feat-b"])
	})

	t.Run("closes the current space last", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main", "feat-a", "feat-b")
		t.Chdir(groveWorkspace.Dir)
		callsPath := stubHerdrApp(t, herdrListScript(t,
			testHerdrSpace{ID: "w1", Folder: groveWorkspace.Worktrees["feat-a"], Terminal: "t74"},
			testHerdrSpace{ID: "w2", Folder: groveWorkspace.Worktrees["feat-b"]},
		), "")
		t.Setenv("HC_TERMINAL_ID", "t74")

		if _, _, err := executeCommand(t, NewRemoveCmd(), "feat-a", "feat-b", "--herdr"); err != nil {
			t.Fatal(err)
		}

		calls := readHerdrAppCalls(t, callsPath)
		want := []string{"ls", "do space.close --id w2 --confirmed true", "do space.close --id w1 --confirmed true"}
		if strings.Join(calls, "|") != strings.Join(want, "|") {
			t.Errorf("herdr-app calls = %q, want the current w1 closed last", calls)
		}
	})

	t.Run("warns when a close fails after removal", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main", "feat-a")
		t.Chdir(groveWorkspace.Dir)
		stubHerdrApp(t, herdrListScript(t, testHerdrSpace{ID: "w1", Folder: groveWorkspace.Worktrees["feat-a"]}),
			"printf 'no such space\\n' >&2\nexit 1\n")

		_, stderr, err := executeCommand(t, NewRemoveCmd(), "feat-a", "--herdr")
		if err != nil {
			t.Fatalf("remove = %v, want success", err)
		}
		if !strings.Contains(stderr, "w1") || !strings.Contains(stderr, "no such space") {
			t.Errorf("stderr = %q, want close warning", stderr)
		}
		if _, err := os.Stat(groveWorkspace.Worktrees["feat-a"]); !os.IsNotExist(err) {
			t.Errorf("feat-a should be removed: %v", err)
		}
	})

	t.Run("removes nothing when the space list fails", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main", "feat-a")
		t.Chdir(groveWorkspace.Dir)
		callsPath := stubHerdrApp(t, "printf 'server down\\n' >&2\nexit 1", "")

		_, _, err := executeCommand(t, NewRemoveCmd(), "feat-a", "--herdr")
		if err == nil || !strings.Contains(err.Error(), "server down") {
			t.Fatalf("remove = %v, want list failure", err)
		}
		testutil.AssertPathExists(t, groveWorkspace.Worktrees["feat-a"])
		if calls := readHerdrAppCalls(t, callsPath); len(calls) != 1 {
			t.Errorf("herdr-app calls = %q, want only the list", calls)
		}
	})

	t.Run("removes nothing when herdr-app is missing", func(t *testing.T) {
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
		if err == nil || !strings.Contains(err.Error(), "herdr-app") {
			t.Fatalf("remove = %v, want missing herdr-app error", err)
		}
		testutil.AssertPathExists(t, groveWorkspace.Worktrees["feat-a"])
	})

	t.Run("never calls herdr-app without the flag", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main", "feat-a")
		t.Chdir(groveWorkspace.Dir)
		callsPath := stubHerdrApp(t, herdrListScript(t, testHerdrSpace{ID: "w1", Folder: groveWorkspace.Worktrees["feat-a"]}), "")

		if _, _, err := executeCommand(t, NewRemoveCmd(), "feat-a"); err != nil {
			t.Fatal(err)
		}
		if calls := readHerdrAppCalls(t, callsPath); calls != nil {
			t.Errorf("herdr-app calls = %q, want none", calls)
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
		callsPath = stubHerdrApp(t, herdrListScript(t,
			testHerdrSpace{ID: "w0", Folder: groveWorkspace.Worktrees["main"]},
			testHerdrSpace{ID: "w1", Folder: filepath.Join(groveWorkspace.Dir, "parked")},
		), "")

		return groveWorkspace, callsPath
	}

	t.Run("dry run marks spaces without closing them", func(t *testing.T) {
		_, callsPath := setup(t)

		_, stderr, err := executeCommand(t, NewPruneCmd(), "--detached", "--herdr")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(stderr, "(closes Herdr space") != 1 {
			t.Errorf("stderr = %q, want one marked candidate", stderr)
		}
		for _, line := range strings.Split(stderr, "\n") {
			marked := strings.HasSuffix(line, "(closes Herdr space w1)")
			if strings.Contains(line, "parked [") != marked || strings.Contains(line, "idle [") && marked {
				t.Errorf("stderr line %q, want only parked marked", line)
			}
		}
		if calls := readHerdrAppCalls(t, callsPath); len(calls) != 1 {
			t.Errorf("herdr-app calls = %q, want only the list", calls)
		}
	})

	t.Run("dry run JSON marks spaces without closing them", func(t *testing.T) {
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
			id, marked := candidate["herdr_space_id"]
			if candidate["name"] == "parked" && id != "w1" || candidate["name"] == "idle" && marked {
				t.Errorf("candidate %v, want parked w1 and idle without herdr_space_id", candidate)
			}
			if _, aliased := candidate["herdr_workspace_id"]; aliased {
				t.Errorf("candidate %v must not carry the herdr_workspace_id alias", candidate)
			}
		}
		if calls := readHerdrAppCalls(t, callsPath); len(calls) != 1 {
			t.Errorf("herdr-app calls = %q, want only the list", calls)
		}
	})

	t.Run("commit closes spaces of pruned worktrees", func(t *testing.T) {
		groveWorkspace, callsPath := setup(t)

		stdout, _, err := executeCommand(t, NewPruneCmd(), "--detached", "--herdr", "--commit")
		if err != nil {
			t.Fatal(err)
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
		want := []string{"ls", "do space.close --id w1 --confirmed true"}
		if calls := readHerdrAppCalls(t, callsPath); strings.Join(calls, "|") != strings.Join(want, "|") {
			t.Errorf("herdr-app calls = %q, want %q", calls, want)
		}
		if _, err := os.Stat(filepath.Join(groveWorkspace.Dir, "parked")); !os.IsNotExist(err) {
			t.Errorf("parked should be removed: %v", err)
		}
	})

	t.Run("commit closes the current space last", func(t *testing.T) {
		groveWorkspace := testgit.NewGroveWorkspace(t, "main")
		t.Chdir(groveWorkspace.Worktrees["main"])
		for _, name := range []string{"parked", "idle"} {
			path := filepath.Join(groveWorkspace.Dir, name)
			if err := git.CreateWorktree(groveWorkspace.BareDir, path, git.CreateWorktreeOptions{Branch: "main", Detach: true}, true); err != nil {
				t.Fatal(err)
			}
		}
		callsPath := stubHerdrApp(t, herdrListScript(t,
			testHerdrSpace{ID: "w0", Folder: groveWorkspace.Worktrees["main"]},
			testHerdrSpace{ID: "w1", Folder: filepath.Join(groveWorkspace.Dir, "parked")},
			testHerdrSpace{ID: "w2", Folder: filepath.Join(groveWorkspace.Dir, "idle"), Terminal: "t74"},
		), "")
		t.Setenv("HC_TERMINAL_ID", "t74")

		if _, _, err := executeCommand(t, NewPruneCmd(), "--detached", "--herdr", "--commit"); err != nil {
			t.Fatal(err)
		}

		calls := readHerdrAppCalls(t, callsPath)
		want := []string{"ls", "do space.close --id w1 --confirmed true", "do space.close --id w2 --confirmed true"}
		if strings.Join(calls, "|") != strings.Join(want, "|") {
			t.Errorf("herdr-app calls = %q, want the current w2 closed last", calls)
		}
	})

	t.Run("never calls herdr-app without the flag", func(t *testing.T) {
		_, callsPath := setup(t)

		stdout, _, err := executeCommand(t, NewPruneCmd(), "--detached", "--json")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(stdout, "herdr_space_id") || strings.Contains(stdout, "herdr_workspace_id") {
			t.Errorf("stdout = %q, want no herdr field", stdout)
		}
		if _, _, err := executeCommand(t, NewPruneCmd(), "--detached", "--commit"); err != nil {
			t.Fatal(err)
		}
		if calls := readHerdrAppCalls(t, callsPath); calls != nil {
			t.Errorf("herdr-app calls = %q, want none", calls)
		}
	})

	t.Run("commit removes nothing when the space list fails", func(t *testing.T) {
		groveWorkspace, _ := setup(t)
		callsPath := stubHerdrApp(t, "printf 'server down\\n' >&2\nexit 1", "")

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
		if calls := readHerdrAppCalls(t, callsPath); len(calls) != 1 {
			t.Errorf("herdr-app calls = %q, want only the list", calls)
		}
	})

	t.Run("skipped candidates keep their space", func(t *testing.T) {
		groveWorkspace, callsPath := setup(t)
		testutil.WriteFile(t, filepath.Join(groveWorkspace.Dir, "parked", "dirty.txt"), "dirty")

		_, stderr, err := executeCommand(t, NewPruneCmd(), "--detached", "--herdr")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(stderr, "closes Herdr space") {
			t.Errorf("stderr = %q, want no marked candidate", stderr)
		}
		if _, _, err := executeCommand(t, NewPruneCmd(), "--detached", "--herdr", "--commit"); err != nil {
			t.Fatal(err)
		}
		if calls := readHerdrAppCalls(t, callsPath); strings.Contains(strings.Join(calls, "|"), "space.close") {
			t.Errorf("herdr-app calls = %q, want no close", calls)
		}
		testutil.AssertPathExists(t, filepath.Join(groveWorkspace.Dir, "parked"))

		if _, _, err := executeCommand(t, NewPruneCmd(), "--detached", "--herdr", "--commit", "--force"); err != nil {
			t.Fatal(err)
		}
		calls := readHerdrAppCalls(t, callsPath)
		if calls[len(calls)-1] != "do space.close --id w1 --confirmed true" {
			t.Errorf("herdr-app calls = %q, want w1 closed with --force", calls)
		}
	})

	t.Run("commit closes the space of a path-gone worktree under a symlinked root", func(t *testing.T) {
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
		// herdr-app spells the missing worktree through an alias of the root git recorded.
		link := filepath.Join(t.TempDir(), "link")
		if err := os.Symlink(groveWorkspace.Dir, link); err != nil {
			t.Fatal(err)
		}
		callsPath := stubHerdrApp(t, herdrListScript(t, testHerdrSpace{ID: "w1", Folder: filepath.Join(link, "gone")}), "")

		if _, _, err := executeCommand(t, NewPruneCmd(), "--herdr", "--commit"); err != nil {
			t.Fatal(err)
		}
		if calls := readHerdrAppCalls(t, callsPath); len(calls) != 2 || calls[1] != "do space.close --id w1 --confirmed true" {
			t.Errorf("herdr-app calls = %q, want w1 closed", calls)
		}
	})
}
