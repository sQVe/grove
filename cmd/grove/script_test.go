package main

import (
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestScript(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/script",
		Setup: func(env *testscript.Env) error {
			env.Vars = append(env.Vars, "GROVE_PLAIN=true")
			return nil
		},
	})
}

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"grove": main,
	})
}
