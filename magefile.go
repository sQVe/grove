//go:build mage

package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Test contains test-related targets.
type Test mg.Namespace

// Unit runs fast unit tests (excluding integration tests).
func (Test) Unit() error {
	fmt.Println("Running unit tests...")
	return sh.RunV("go", "test", "-tags=!integration", "-short", "./...")
}

// Integration runs slow integration tests only.
func (Test) Integration() error {
	fmt.Println("Running integration tests...")
	return sh.RunV("go", "test", "-tags=integration", "./...")
}

// All runs both unit and integration tests.
func (Test) All() error {
	fmt.Println("Running all tests...")
	return sh.RunV("go", "test", "./...")
}

// Coverage runs unit tests with coverage reporting.
func (Test) Coverage() error {
	fmt.Println("Running unit tests with coverage...")

	if err := os.MkdirAll("coverage", 0755); err != nil {
		return err
	}

	if err := sh.RunV("go", "test", "-tags=!integration", "-short", "-coverprofile=coverage/coverage.out", "./..."); err != nil {
		return err
	}

	if err := sh.RunV("go", "tool", "cover", "-html=coverage/coverage.out", "-o=coverage/coverage.html"); err != nil {
		return err
	}

	fmt.Println("Coverage report generated at coverage/coverage.html")
	return nil
}

// Watch runs unit tests in watch mode (requires entr).
func (Test) Watch() error {
	fmt.Println("Running unit tests in watch mode...")
	fmt.Println("Press Ctrl+C to stop watching...")

	// Check if entr is available
	if err := sh.Run("which", "entr"); err != nil {
		return fmt.Errorf("entr is required for watch mode. Install with: brew install entr (macOS) or apt-get install entr (Linux)")
	}

	// Use find command to locate Go files and pipe to entr
	return sh.RunV("sh", "-c", "find . -name '*.go' | entr -c -n mage test:unit")
}

// Clean removes test artifacts.
func (Test) Clean() error {
	fmt.Println("Cleaning test artifacts...")

	// Remove coverage directory
	if err := os.RemoveAll("coverage"); err != nil {
		return err
	}

	// Remove any test cache
	return sh.RunV("go", "clean", "-testcache")
}

// Default test target runs unit tests.
func (Test) Default() error {
	return Test{}.Unit()
}

// Build contains build-related targets.
type Build mg.Namespace

// All builds the application for the current platform.
func (Build) All() error {
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

// Clean removes build artifacts from bin directory.
func (Build) Clean() error {
	fmt.Println("Cleaning build artifacts...")
	return os.RemoveAll("bin")
}

// Lint runs golangci-lint (with --fix unless in CI).
func Lint() error {
	fmt.Println("Running golangci-lint...")

	// Check if we're in CI environment
	if os.Getenv("CI") != "" {
		return sh.RunV("golangci-lint", "run")
	}

	// Run with --fix in local development
	return sh.RunV("golangci-lint", "run", "--fix")
}

// CI runs the full CI pipeline.
func CI() error {
	fmt.Println("Running CI pipeline...")

	// Clean first
	test := Test{}
	build := Build{}

	mg.Deps(test.Clean, build.Clean)

	// Run linting
	if err := Lint(); err != nil {
		return fmt.Errorf("linting failed: %w", err)
	}

	// Run unit tests
	if err := test.Unit(); err != nil {
		return fmt.Errorf("unit tests failed: %w", err)
	}

	// Run integration tests
	if err := test.Integration(); err != nil {
		return fmt.Errorf("integration tests failed: %w", err)
	}

	// Build the application
	if err := build.All(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	fmt.Println("CI pipeline completed successfully!")
	return nil
}

// Dev runs a development environment setup.
func Dev() error {
	fmt.Println("Setting up development environment...")

	// Install development tools
	tools := []string{
		"github.com/golangci/golangci-lint/cmd/golangci-lint@latest",
	}

	for _, tool := range tools {
		fmt.Printf("Installing %s...\n", tool)
		if err := sh.RunV("go", "install", tool); err != nil {
			fmt.Printf("Warning: failed to install %s: %v\n", tool, err)
		}
	}

	fmt.Println("Development environment setup complete!")
	return nil
}

// Clean removes all generated artifacts.
func Clean() error {
	fmt.Println("Cleaning all artifacts...")

	test := Test{}
	build := Build{}

	mg.Deps(test.Clean, build.Clean)

	return nil
}

// Info displays environment information.
func Info() error {
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("Go OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	// Show git information
	if output, err := sh.Output("git", "rev-parse", "--short", "HEAD"); err == nil {
		fmt.Printf("Git commit: %s\n", strings.TrimSpace(output))
	}

	if output, err := sh.Output("git", "branch", "--show-current"); err == nil {
		fmt.Printf("Git branch: %s\n", strings.TrimSpace(output))
	}

	// Show module information
	if output, err := sh.Output("go", "list", "-m"); err == nil {
		fmt.Printf("Module: %s\n", strings.TrimSpace(output))
	}

	return nil
}

// Help displays available targets.
func Help() error {
	fmt.Println("Available targets:")
	fmt.Println("")
	fmt.Println("Test targets:")
	fmt.Println("  mage test:unit        - Run fast unit tests (default)")
	fmt.Println("  mage test:integration - Run slow integration tests")
	fmt.Println("  mage test:all         - Run all tests")
	fmt.Println("  mage test:coverage    - Run unit tests with coverage")
	fmt.Println("  mage test:watch       - Watch for changes and run unit tests")
	fmt.Println("  mage test:clean       - Clean test artifacts")
	fmt.Println("")
	fmt.Println("Build targets:")
	fmt.Println("  mage build:all        - Build the application")
	fmt.Println("  mage build:release    - Build release binaries")
	fmt.Println("  mage build:clean      - Clean build artifacts")
	fmt.Println("")
	fmt.Println("Lint targets:")
	fmt.Println("  mage lint             - Run golangci-lint (with --fix unless in CI)")
	fmt.Println("")
	fmt.Println("Other targets:")
	fmt.Println("  mage ci               - Run full CI pipeline")
	fmt.Println("  mage dev              - Setup development environment")
	fmt.Println("  mage clean            - Clean all artifacts")
	fmt.Println("  mage info             - Display environment information")
	fmt.Println("")
	fmt.Println("For faster development, use 'mage test:unit' or just 'mage'")

	return nil
}

// Default target runs unit tests.
func Default() error {
	test := Test{}
	return test.Unit()
}

// Ensure bin directory exists.
func init() {
	if err := os.MkdirAll("bin", 0755); err != nil {
		fmt.Printf("Warning: failed to create bin directory: %v\n", err)
	}
}
