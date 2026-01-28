package workspace

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/testutil"
)

const testEnvFile = ".env"

func TestIsInsideGroveWorkspace(t *testing.T) {
	t.Parallel()

	t.Run("returns false for non-grove directory", func(t *testing.T) {
		t.Parallel()
		tempDir := testutil.TempDir(t)
		result := IsInsideGroveWorkspace(tempDir)
		if result {
			t.Error("expected false for non-grove directory")
		}
	})

	t.Run("returns true for grove workspace root", func(t *testing.T) {
		t.Parallel()

		tempDir := testutil.TempDir(t)
		bareDir := filepath.Join(tempDir, ".bare")
		if err := os.MkdirAll(bareDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create .bare directory: %v", err)
		}

		result := IsInsideGroveWorkspace(tempDir)
		if !result {
			t.Error("expected true for grove workspace root")
		}
	})

	t.Run("returns true for subdirectory of grove workspace", func(t *testing.T) {
		t.Parallel()

		tempDir := testutil.TempDir(t)
		bareDir := filepath.Join(tempDir, ".bare")
		if err := os.MkdirAll(bareDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create .bare directory: %v", err)
		}

		subDir := filepath.Join(tempDir, "subdir", "nested")
		if err := os.MkdirAll(subDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		result := IsInsideGroveWorkspace(subDir)
		if !result {
			t.Error("expected true for subdirectory of grove workspace")
		}
	})

	t.Run("returns false for regular git repository", func(t *testing.T) {
		t.Parallel()

		tempDir := testutil.TempDir(t)
		gitDir := filepath.Join(tempDir, ".git")
		if err := os.Mkdir(gitDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create .git directory: %v", err)
		}

		result := IsInsideGroveWorkspace(tempDir)
		if result {
			t.Error("expected false for regular git repository")
		}
	})
}

func TestSanitizeBranchName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		branch   string
		expected string
	}{
		// Standard slash replacement
		{"feature/add-button", "feature-add-button"},
		{"feat/user-auth", "feat-user-auth"},
		{"bug/fix-123", "bug-fix-123"},
		{"release/v1.0.0", "release-v1.0.0"},
		{"hotfix/urgent-patch", "hotfix-urgent-patch"},
		{"no-slash", "no-slash"},
		{"multiple//slashes///here", "multiple--slashes---here"},
		{"trailing/slash/", "trailing-slash-"},
		{"/leading/slash", "-leading-slash"},

		// Special characters (<>|"?*:)
		{"branch<name>with|chars", "branch-name-with-chars"},
		{`branch"with"quotes`, "branch-with-quotes"},
		{"branch?with?questions", "branch-with-questions"},
		{"branch*with*wildcards", "branch-with-wildcards"},
		{"branch:with:colons", "branch-with-colons"},
		{"all<>|\"?*:special", "all-------special"},

		// Backslash (Windows path separator)
		{`feature\auth`, "feature-auth"},
		{`path\to\branch`, "path-to-branch"},

		// Combined edge cases
		{"feature/auth<v2>", "feature-auth-v2-"},
		{`release\v1.0.0:final`, "release-v1.0.0-final"},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			t.Parallel()

			result := SanitizeBranchName(tt.branch)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestPreserveIgnoredFilesFromList_NoIgnoredFiles(t *testing.T) {
	t.Parallel()
	tempDir := testutil.TempDir(t)
	branches := []string{"main", "develop"}

	count, patterns, err := preserveIgnoredFilesFromList(tempDir, branches, []string{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected preserved file count 0, got %d", count)
	}
	if patterns != nil {
		t.Errorf("expected nil matched patterns, got %v", patterns)
	}
}

func TestPreserveIgnoredFilesFromList_ValidPreserve(t *testing.T) {
	t.Parallel()

	tempDir := testutil.TempDir(t)
	branches := []string{"main", "develop"}

	// Create worktree directories
	for _, branch := range branches {
		branchDir := filepath.Join(tempDir, SanitizeBranchName(branch))
		if err := os.MkdirAll(branchDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create branch directory %s: %v", branchDir, err)
		}
	}

	// Create a file that matches default ".env" preserve pattern
	envPath := filepath.Join(tempDir, testEnvFile)
	content := []byte("preserve test content")
	if err := os.WriteFile(envPath, content, fs.FileStrict); err != nil {
		t.Fatalf("failed to create file %s: %v", envPath, err)
	}

	// Create a non-matching file
	nonMatchPath := filepath.Join(tempDir, "ignored.txt")
	if err := os.WriteFile(nonMatchPath, []byte("should not be preserved"), fs.FileStrict); err != nil {
		t.Fatalf("failed to create file %s: %v", nonMatchPath, err)
	}

	ignoredFiles := []string{".env", "ignored.txt"}
	count, matched, err := preserveIgnoredFilesFromList(tempDir, branches, ignoredFiles, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the ".env" file should be preserved
	if count != 1 {
		t.Errorf("expected preserved file count 1, got %d", count)
	}
	found := false
	for _, pat := range matched {
		if pat == testEnvFile {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected matched patterns to include '.env', got %v", matched)
	}

	// Verify the preserved file exists in each worktree directory
	for _, branch := range branches {
		branchDir := filepath.Join(tempDir, SanitizeBranchName(branch))
		preservedFile := filepath.Join(branchDir, testEnvFile)
		if _, err := os.Stat(preservedFile); err != nil {
			t.Errorf("expected preserved file %s in branch %q, error: %v", preservedFile, branch, err)
		} else {
			c, err := os.ReadFile(preservedFile) // nolint:gosec // Test file - controlled path
			if err != nil {
				t.Errorf("failed to read file %s: %v", preservedFile, err)
			}
			if !bytes.Equal(c, content) {
				t.Errorf("content mismatch in preserved file %s: got %q, want %q", preservedFile, string(c), string(content))
			}
		}
	}
}

func TestPreserveIgnoredFilesFromList_CustomPattern(t *testing.T) {
	t.Parallel()

	tempDir := testutil.TempDir(t)
	branches := []string{"feature"}

	// Create worktree directory
	branchDir := filepath.Join(tempDir, SanitizeBranchName("feature"))
	if err := os.MkdirAll(branchDir, fs.DirGit); err != nil {
		t.Fatalf("failed to create branch directory %s: %v", branchDir, err)
	}

	// Create .grove.toml with custom pattern
	tomlContent := `[preserve]
patterns = ["*.custom"]
`
	if err := os.WriteFile(filepath.Join(tempDir, ".grove.toml"), []byte(tomlContent), fs.FileStrict); err != nil {
		t.Fatalf("failed to create .grove.toml: %v", err)
	}

	// Create a file that matches the custom pattern
	customFileName := "data.custom"
	customFilePath := filepath.Join(tempDir, customFileName)
	content := []byte("custom content")
	if err := os.WriteFile(customFilePath, content, fs.FileStrict); err != nil {
		t.Fatalf("failed to create file %s: %v", customFilePath, err)
	}

	ignoredFiles := []string{customFileName}
	count, matched, err := preserveIgnoredFilesFromList(tempDir, branches, ignoredFiles, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 1 {
		t.Errorf("expected preserved file count 1, got %d", count)
	}
	found := false
	for _, pat := range matched {
		if pat == "*.custom" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected matched patterns to include '*.custom', got %v", matched)
	}

	// Verify the file is copied to the worktree directory
	preservedFile := filepath.Join(branchDir, customFileName)
	if _, err := os.Stat(preservedFile); err != nil {
		t.Errorf("expected preserved file %s in branch 'feature', error: %v", preservedFile, err)
	}
}

func TestPreserveIgnoredFilesFromList_ReadsTomlConfig(t *testing.T) {
	t.Parallel()

	tempDir := testutil.TempDir(t)
	branches := []string{"feature"}

	// Create worktree directory
	branchDir := filepath.Join(tempDir, SanitizeBranchName("feature"))
	if err := os.MkdirAll(branchDir, fs.DirGit); err != nil {
		t.Fatalf("failed to create branch directory %s: %v", branchDir, err)
	}

	// Create .grove.toml with a custom pattern
	tomlContent := `[preserve]
patterns = ["*.tomltest"]
`
	if err := os.WriteFile(filepath.Join(tempDir, ".grove.toml"), []byte(tomlContent), fs.FileStrict); err != nil {
		t.Fatalf("failed to create .grove.toml: %v", err)
	}

	// Create a file that matches the TOML pattern (NOT the global default)
	testFileName := "data.tomltest"
	testFilePath := filepath.Join(tempDir, testFileName)
	content := []byte("toml config content")
	if err := os.WriteFile(testFilePath, content, fs.FileStrict); err != nil {
		t.Fatalf("failed to create file %s: %v", testFilePath, err)
	}

	ignoredFiles := []string{testFileName}
	count, matched, err := preserveIgnoredFilesFromList(tempDir, branches, ignoredFiles, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 1 {
		t.Errorf("expected preserved file count 1, got %d (TOML config pattern not read)", count)
	}
	found := false
	for _, pat := range matched {
		if pat == "*.tomltest" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected matched patterns to include '*.tomltest' from .grove.toml, got %v", matched)
	}

	// Verify the file is copied to the worktree directory
	preservedFile := filepath.Join(branchDir, testFileName)
	if _, err := os.Stat(preservedFile); err != nil {
		t.Errorf("expected preserved file %s in branch 'feature', error: %v", preservedFile, err)
	}
}

func TestPreserveIgnoredFilesFromList_MissingSource(t *testing.T) {
	t.Parallel()

	tempDir := testutil.TempDir(t)
	branches := []string{"main"}

	// Create worktree directory
	branchDir := filepath.Join(tempDir, SanitizeBranchName("main"))
	if err := os.MkdirAll(branchDir, fs.DirGit); err != nil {
		t.Fatalf("failed to create branch directory %s: %v", branchDir, err)
	}

	// Do not create the file ".env" even though it matches default preserve pattern
	ignoredFiles := []string{".env"}
	_, _, err := preserveIgnoredFilesFromList(tempDir, branches, ignoredFiles, nil)
	if err == nil {
		t.Error("expected error due to missing source file, got nil")
	}
	if !strings.Contains(err.Error(), "failed to preserve file") {
		t.Errorf("expected error message about failing to preserve file, got: %v", err)
	}
}

func TestParseBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		branches   string
		skipBranch string
		want       []string
	}{
		{"single branch", "main", "", []string{"main"}},
		{"multiple branches", "main,develop,feature", "", []string{"main", "develop", "feature"}},
		{"with spaces", " main , develop , feature ", "", []string{"main", "develop", "feature"}},
		{"with skip branch", "main,develop,feature", "develop", []string{"main", "feature"}},
		{"empty string", "", "", nil},
		{"only whitespace", "   ", "", nil},
		{"skip only branch", "main", "main", nil},
		{"trailing comma", "main,develop,", "", []string{"main", "develop"}},
		{"leading comma", ",main,develop", "", []string{"main", "develop"}},
		{"multiple commas", "main,,develop", "", []string{"main", "develop"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := parseBranches(tt.branches, tt.skipBranch)
			if len(got) != len(tt.want) {
				t.Errorf("parseBranches(%q, %q) = %v, want %v", tt.branches, tt.skipBranch, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseBranches(%q, %q)[%d] = %q, want %q", tt.branches, tt.skipBranch, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestMatchesPattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filePath string
		pattern  string
		want     bool
	}{
		{"exact match filename", ".env", ".env", true},
		{"exact match path", "config/.env", ".env", true},
		{"wildcard extension", "data.json", "*.json", true},
		{"wildcard path", "config/data.json", "*.json", true},
		{"no match", "data.txt", "*.json", false},
		{"no match different name", "readme.md", ".env", false},
		{"nested path exact", "deep/nested/.env", ".env", true},
		{"wildcard prefix", "test.env.local", "*.env.local", true},
		{"partial no match", ".env.local", ".env", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := matchesPattern(tt.filePath, tt.pattern)
			if got != tt.want {
				t.Errorf("matchesPattern(%q, %q) = %v, want %v", tt.filePath, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestValidateAndPrepareDirectory(t *testing.T) {
	t.Parallel()

	t.Run("rejects non-empty directory", func(t *testing.T) {
		t.Parallel()
		dir := testutil.TempDir(t)

		// Create a file to make directory non-empty
		if err := os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("data"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		err := ValidateAndPrepareDirectory(dir)
		if err == nil {
			t.Error("expected error for non-empty directory")
		}
		if !strings.Contains(err.Error(), "not empty") {
			t.Errorf("expected 'not empty' error, got: %v", err)
		}
	})

	t.Run("rejects directory inside git repository", func(t *testing.T) {
		t.Parallel()

		dir := testutil.TempDir(t)

		// Initialize a real git repo
		cmd := exec.Command("git", "init")
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to init git repo: %v", err)
		}

		// Create existing subdir inside the git repo
		subDir := filepath.Join(dir, "subdir")
		if err := os.Mkdir(subDir, fs.DirGit); err != nil {
			t.Fatal(err)
		}

		err := ValidateAndPrepareDirectory(subDir)
		if err == nil {
			t.Error("expected error for directory inside git repo")
		}
		if !strings.Contains(err.Error(), "existing git repository") {
			t.Errorf("expected 'existing git repository' error, got: %v", err)
		}
	})

	t.Run("rejects directory inside grove workspace", func(t *testing.T) {
		t.Parallel()

		dir := testutil.TempDir(t)

		// Create .bare directory to simulate grove workspace
		if err := os.Mkdir(filepath.Join(dir, ".bare"), fs.DirGit); err != nil {
			t.Fatal(err)
		}

		// Create existing subdir inside the grove workspace
		subDir := filepath.Join(dir, "subdir")
		if err := os.Mkdir(subDir, fs.DirGit); err != nil {
			t.Fatal(err)
		}

		err := ValidateAndPrepareDirectory(subDir)
		if err == nil {
			t.Error("expected error for directory inside grove workspace")
		}
		if !strings.Contains(err.Error(), "existing grove workspace") {
			t.Errorf("expected 'existing grove workspace' error, got: %v", err)
		}
	})

	t.Run("accepts empty existing directory", func(t *testing.T) {
		t.Parallel()

		dir := testutil.TempDir(t)

		err := ValidateAndPrepareDirectory(dir)
		if err != nil {
			t.Errorf("unexpected error for empty directory: %v", err)
		}
	})

	t.Run("creates new directory if it does not exist", func(t *testing.T) {
		t.Parallel()

		parentDir := testutil.TempDir(t)
		newDir := filepath.Join(parentDir, "newdir")

		err := ValidateAndPrepareDirectory(newDir)
		if err != nil {
			t.Errorf("unexpected error creating new directory: %v", err)
		}

		// Verify directory was created
		info, err := os.Stat(newDir)
		if err != nil {
			t.Errorf("expected directory to be created, got error: %v", err)
		}
		if !info.IsDir() {
			t.Error("expected created path to be a directory")
		}
	})
}

func TestFindBareDir(t *testing.T) {
	t.Parallel()

	t.Run("returns bare dir path from workspace root", func(t *testing.T) {
		t.Parallel()
		workspaceDir := testutil.TempDir(t)
		bareDir := filepath.Join(workspaceDir, ".bare")
		if err := os.MkdirAll(bareDir, fs.DirGit); err != nil {
			t.Fatal(err)
		}

		result, err := FindBareDir(workspaceDir)
		if err != nil {
			t.Fatalf("FindBareDir failed: %v", err)
		}
		if result != bareDir {
			t.Errorf("expected %s, got %s", bareDir, result)
		}
	})

	t.Run("returns bare dir from subdirectory", func(t *testing.T) {
		t.Parallel()

		workspaceDir := testutil.TempDir(t)
		bareDir := filepath.Join(workspaceDir, ".bare")
		subDir := filepath.Join(workspaceDir, "main", "src")
		if err := os.MkdirAll(bareDir, fs.DirGit); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(subDir, fs.DirGit); err != nil {
			t.Fatal(err)
		}

		result, err := FindBareDir(subDir)
		if err != nil {
			t.Fatalf("FindBareDir failed: %v", err)
		}
		if result != bareDir {
			t.Errorf("expected %s, got %s", bareDir, result)
		}
	})

	t.Run("returns error outside workspace", func(t *testing.T) {
		t.Parallel()

		dir := testutil.TempDir(t)

		_, err := FindBareDir(dir)
		if err == nil {
			t.Error("expected error for non-workspace dir")
		}
		if !errors.Is(err, ErrNotInWorkspace) {
			t.Errorf("expected ErrNotInWorkspace, got %v", err)
		}
	})

	t.Run("returns bare dir from deeply nested subdirectory (50 levels)", func(t *testing.T) {
		t.Parallel()
		workspaceDir := testutil.TempDir(t)
		bareDir := filepath.Join(workspaceDir, ".bare")
		if err := os.MkdirAll(bareDir, fs.DirGit); err != nil {
			t.Fatal(err)
		}

		// Create a 50-level deep directory structure
		deepDir := workspaceDir
		for i := 0; i < 50; i++ {
			deepDir = filepath.Join(deepDir, "level")
		}
		if err := os.MkdirAll(deepDir, fs.DirGit); err != nil {
			t.Fatal(err)
		}

		result, err := FindBareDir(deepDir)
		if err != nil {
			t.Fatalf("FindBareDir failed for deep path: %v", err)
		}
		if result != bareDir {
			t.Errorf("expected %s, got %s", bareDir, result)
		}
	})
}

func TestResolveConfigDir(t *testing.T) {
	t.Parallel()

	t.Run("returns worktree root when inside worktree", func(t *testing.T) {
		t.Parallel()
		workspaceDir := testutil.TempDir(t)
		bareDir := filepath.Join(workspaceDir, ".bare")

		// Create bare repo
		if err := os.MkdirAll(bareDir, fs.DirGit); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("git", "init", "--bare")
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		// Create a worktree directory with .git file
		worktreeDir := filepath.Join(workspaceDir, "main")
		if err := os.MkdirAll(worktreeDir, fs.DirGit); err != nil {
			t.Fatal(err)
		}
		gitFile := filepath.Join(worktreeDir, ".git")
		if err := os.WriteFile(gitFile, []byte("gitdir: ../.bare/worktrees/main"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		// Test from worktree root
		result, err := ResolveConfigDir(worktreeDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != worktreeDir {
			t.Errorf("expected %s, got %s", worktreeDir, result)
		}

		// Test from subdirectory within worktree
		subDir := filepath.Join(worktreeDir, "src", "pkg")
		if err := os.MkdirAll(subDir, fs.DirGit); err != nil {
			t.Fatal(err)
		}
		result, err = ResolveConfigDir(subDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != worktreeDir {
			t.Errorf("expected %s, got %s", worktreeDir, result)
		}
	})

	t.Run("returns default branch worktree from workspace root", func(t *testing.T) {
		t.Parallel()

		workspaceDir := testutil.TempDir(t)
		bareDir := filepath.Join(workspaceDir, ".bare")

		// Create bare repo with HEAD pointing to main
		if err := os.MkdirAll(bareDir, fs.DirGit); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("git", "init", "--bare")
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		// HEAD file should already point to refs/heads/main by default

		// Create main worktree directory
		mainWorktree := filepath.Join(workspaceDir, "main")
		if err := os.MkdirAll(mainWorktree, fs.DirGit); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(mainWorktree, ".git"), []byte("gitdir: ../.bare/worktrees/main"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		// Create another worktree (alphabetically first)
		alphaWorktree := filepath.Join(workspaceDir, "alpha")
		if err := os.MkdirAll(alphaWorktree, fs.DirGit); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(alphaWorktree, ".git"), []byte("gitdir: ../.bare/worktrees/alpha"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		// Register worktrees with git (create worktree metadata)
		worktreesDir := filepath.Join(bareDir, "worktrees")
		for _, name := range []string{"main", "alpha"} {
			wtDir := filepath.Join(worktreesDir, name)
			if err := os.MkdirAll(wtDir, fs.DirGit); err != nil {
				t.Fatal(err)
			}
			gitdirPath := filepath.Join(workspaceDir, name)
			if err := os.WriteFile(filepath.Join(wtDir, "gitdir"), []byte(gitdirPath), fs.FileStrict); err != nil {
				t.Fatal(err)
			}
		}

		result, err := ResolveConfigDir(workspaceDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != mainWorktree {
			t.Errorf("expected default branch worktree %s, got %s", mainWorktree, result)
		}
	})

	t.Run("returns first worktree when default branch missing", func(t *testing.T) {
		t.Parallel()

		workspaceDir := testutil.TempDir(t)
		bareDir := filepath.Join(workspaceDir, ".bare")

		// Create bare repo
		if err := os.MkdirAll(bareDir, fs.DirGit); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("git", "init", "--bare")
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		// Create only a feature worktree (no main)
		featureWorktree := filepath.Join(workspaceDir, "feature")
		if err := os.MkdirAll(featureWorktree, fs.DirGit); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(featureWorktree, ".git"), []byte("gitdir: ../.bare/worktrees/feature"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		// Register worktree with git
		worktreesDir := filepath.Join(bareDir, "worktrees", "feature")
		if err := os.MkdirAll(worktreesDir, fs.DirGit); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(worktreesDir, "gitdir"), []byte(featureWorktree), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		result, err := ResolveConfigDir(workspaceDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != featureWorktree {
			t.Errorf("expected first worktree %s, got %s", featureWorktree, result)
		}
	})

	t.Run("returns error outside workspace", func(t *testing.T) {
		t.Parallel()

		dir := testutil.TempDir(t)

		_, err := ResolveConfigDir(dir)
		if err == nil {
			t.Error("expected error for non-workspace dir")
		}
		if !errors.Is(err, ErrNotInWorkspace) {
			t.Errorf("expected ErrNotInWorkspace, got %v", err)
		}
	})
}
