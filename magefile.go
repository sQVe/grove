//go:build mage

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type (
	Test  mg.Namespace
	Build mg.Namespace
)

var Aliases = map[string]interface{}{
	"build": Build.Dev,
	"test":  Test.Unit,
}

func (Test) Unit() error {
	fmt.Println("Running unit tests...")
	return sh.RunV("go", "run", "gotest.tools/gotestsum@latest", "--format", "testname", "--", "-tags=!integration", "-short", "./...")
}

func (Test) Integration() error {
	fmt.Println("Running integration tests...")
	return sh.RunV("go", "run", "gotest.tools/gotestsum@latest", "--format", "testname", "--", "-tags=integration", "-timeout=300s", "./test/integration/...")
}

// Coverage runs unit tests with coverage reporting and optional CI validation.
func (Test) Coverage() error {
	fmt.Println("Running unit tests with coverage...")

	if err := os.MkdirAll("coverage", 0o755); err != nil {
		return err
	}

	// Determine if we're in CI environment
	isCI := os.Getenv("CI") != ""

	// Build coverage command
	args := []string{"test", "-v", "-tags=!integration", "-short", "-coverprofile=coverage/coverage.out", "-coverpkg=./internal/...", "-covermode=atomic"}
	if isCI {
		args = append(args, "-race") // Add race detection for CI
	}
	args = append(args, "./...")

	if err := sh.RunV("go", args...); err != nil {
		return err
	}

	// Generate HTML coverage report (skip in CI)
	if !isCI {
		if err := sh.RunV("go", "tool", "cover", "-html=coverage/coverage.out", "-o=coverage/coverage.html"); err != nil {
			return err
		}
		fmt.Println("Coverage report generated at coverage/coverage.html")
	}

	// Get and display coverage percentage
	if output, err := sh.Output("go", "tool", "cover", "-func=coverage/coverage.out"); err == nil {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) > 0 {
			totalLine := lines[len(lines)-1]
			if strings.Contains(totalLine, "total:") {
				coverageStr := strings.TrimPrefix(totalLine, "total:")
				fmt.Printf("Total %s\n", coverageStr)

				// In CI, validate coverage meets 90% threshold
				if isCI {
					parts := strings.Fields(coverageStr)
					if len(parts) >= 1 {
						// Extract the percentage from the last field
						lastField := parts[len(parts)-1]
						percentage := strings.TrimSuffix(lastField, "%")
						if percentageFloat, err := strconv.ParseFloat(percentage, 64); err == nil {
							if percentageFloat < 90.0 {
								return fmt.Errorf("coverage %.1f%% is below required 90%% threshold", percentageFloat)
							}
						}
					}
				}
			}
		}
	}

	if isCI {
		fmt.Println("Coverage requirement (90%+) met successfully!")
	}
	return nil
}

// Dev builds the main Grove binary for development.
func (Build) Dev() error {
	fmt.Println("Building Grove...")
	return sh.RunV("go", "build", "-o", "bin/grove", "./cmd/grove")
}

// Release builds release binaries for common platforms.
func (Build) Release() error {
	fmt.Println("Building release binaries...")

	platforms := []struct {
		os   string
		arch string
	}{
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"windows", "amd64"},
	}

	for _, platform := range platforms {
		output := fmt.Sprintf("bin/grove-%s-%s", platform.os, platform.arch)
		if platform.os == "windows" {
			output += ".exe"
		}

		fmt.Printf("Building %s...\n", output)

		env := map[string]string{
			"GOOS":        platform.os,
			"GOARCH":      platform.arch,
			"CGO_ENABLED": "0",
		}

		if err := sh.RunWithV(env, "go", "build", "-ldflags", "-s -w", "-o", output, "./cmd/grove"); err != nil {
			return err
		}
	}

	return nil
}

// Lint runs golangci-lint (with --fix unless in CI).
func Lint() error {
	fmt.Println("Running golangci-lint...")

	if os.Getenv("CI") != "" {
		return sh.RunV("golangci-lint", "run")
	}

	return sh.RunV("golangci-lint", "run", "--fix")
}

func CI() error {
	fmt.Println("Running CI pipeline...")

	if err := Clean(); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	if err := Lint(); err != nil {
		return fmt.Errorf("linting failed: %w", err)
	}

	test := Test{}
	if err := test.Coverage(); err != nil {
		return fmt.Errorf("unit tests with coverage failed: %w", err)
	}

	if err := test.Integration(); err != nil {
		return fmt.Errorf("integration tests failed: %w", err)
	}

	build := Build{}
	if err := build.Dev(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	fmt.Println("CI pipeline completed successfully!")
	return nil
}

// Clean removes all generated artifacts.
func Clean() error {
	fmt.Println("Cleaning all artifacts...")

	if err := os.RemoveAll("coverage"); err != nil {
		return err
	}

	if err := os.RemoveAll("bin"); err != nil {
		return err
	}

	return sh.RunV("go", "clean", "-testcache")
}

// Default target runs unit tests.
func Default() error {
	test := Test{}
	return test.Unit()
}

func init() {
	if err := os.MkdirAll("bin", 0o755); err != nil {
		fmt.Printf("Warning: failed to create bin directory: %v\n", err)
	}
}
