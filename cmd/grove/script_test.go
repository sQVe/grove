//go:build integration

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestScript(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/script",
		Setup: func(env *testscript.Env) error {
			env.Vars = append(env.Vars, "GROVE_PLAIN=true")
			homeDir := filepath.Join(env.WorkDir, ".home")
			if err := os.MkdirAll(homeDir, 0755); err != nil {
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
			if err := os.WriteFile(gitConfigPath, []byte(gitConfigContent), 0644); err != nil {
				return err
			}
			return nil
		},
	})
}

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"grove": main,
	})
}
