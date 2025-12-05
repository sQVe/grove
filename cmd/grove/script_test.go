//go:build integration

package main

import (
	"os"
	"path/filepath"
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

			// Pass through GH_TOKEN for PR tests (gh CLI uses this for auth)
			if token := os.Getenv("GH_TOKEN"); token != "" {
				env.Vars = append(env.Vars, "GH_TOKEN="+token)
			}

			return nil
		},
		Condition: func(cond string) (bool, error) {
			switch cond {
			case "ghauth":
				// Check if GH_TOKEN is set for gh CLI authentication
				return os.Getenv("GH_TOKEN") != "", nil
			}
			return false, nil
		},
	})
}

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"grove": main,
	})
}
