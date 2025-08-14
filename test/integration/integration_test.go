//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rogpeppe/go-internal/testscript"
	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/app"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"grove": func() {
			// Isolate test args from real CLI invocation to prevent test interference
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			// Simulate command line execution for testscript integration
			os.Args = []string{"grove"}
			os.Args = append(os.Args, oldArgs[1:]...)

			// Execute grove command with isolated environment
			rootCmd := createRootCommand()
			if err := rootCmd.Execute(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	})
}

const (
	// defaultTestTimeout is the default timeout for individual test scripts.
	// This can be overridden by setting the GROVE_TEST_TIMEOUT environment variable.
	defaultTestTimeout = 30 * time.Second
)

func TestCLI(t *testing.T) {
	// Get timeout from environment or use default.
	timeout := defaultTestTimeout
	if envTimeout := os.Getenv("GROVE_TEST_TIMEOUT"); envTimeout != "" {
		if parsed, err := time.ParseDuration(envTimeout); err == nil {
			timeout = parsed
		}
	}

	testscript.Run(t, testscript.Params{
		Dir:                 "testdata",
		TestWork:            os.Getenv("GROVE_TEST_WORK") != "", // Preserve work dir if env var set
		ContinueOnError:     os.Getenv("GROVE_TEST_CONTINUE") != "", // Continue on error if env var set
		UpdateScripts:       os.Getenv("GROVE_TEST_UPDATE") != "", // Update golden files if env var set
		Deadline:            time.Now().Add(timeout), // Set timeout for each test script
		Setup: func(env *testscript.Env) error {
			// Isolate test environment to prevent interference with system configuration
			// and ensure reproducible test results across different development environments

			env.Setenv("HOME", env.WorkDir)
			env.Setenv("XDG_CONFIG_HOME", env.WorkDir+"/.config")

			// Prevent Git from using system/global configuration that could affect test behavior
			env.Setenv("GIT_CONFIG_NOSYSTEM", "1")
			
			// Create an empty config file for cross-platform compatibility
			// (Windows doesn't have /dev/null)
			emptyConfigPath := filepath.Join(env.WorkDir, ".empty-git-config")
			if err := os.WriteFile(emptyConfigPath, []byte{}, 0644); err != nil {
				return fmt.Errorf("failed to create empty git config: %w", err)
			}
			env.Setenv("GIT_CONFIG_GLOBAL", emptyConfigPath)

			return nil
		},
	})
}

func createRootCommand() *cobra.Command {
	return app.NewRootCommand()
}
