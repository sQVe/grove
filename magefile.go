//go:build mage

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/sqve/grove/internal/fs"
)

type (
	Build mg.Namespace
	Deps  mg.Namespace
	Test  mg.Namespace
)

var Aliases = map[string]interface{}{
	"build": Build.Dev,
	"deps":  Deps.Check,
	"test":  Test.Unit,
}

func (Test) Unit() error {
	fmt.Println("Running unit tests...")
	return sh.RunV("gotestsum", "--", "-tags=!integration", "-short", "./...")
}

func (Test) Integration() error {
	fmt.Println("Running integration tests...")

	// Auto-detect GH_TOKEN from gh CLI if not already set
	env := map[string]string{}
	if os.Getenv("GH_TOKEN") == "" {
		if token, err := sh.Output("gh", "auth", "token"); err == nil && token != "" {
			env["GH_TOKEN"] = token
			fmt.Println("→ Using gh CLI authentication for PR tests")
		}
	}

	return sh.RunWithV(env, "gotestsum", "--", "-tags=integration", "-timeout=300s", "./cmd/grove/...")
}

// Coverage runs unit tests with coverage reporting and optional CI validation.
func (Test) Coverage() error {
	fmt.Println("Running unit tests with coverage...")

	if err := os.MkdirAll("coverage", fs.DirGit); err != nil {
		return err
	}

	// Determine if we're in CI environment
	isCI := os.Getenv("CI") != ""

	// Build coverage command
	args := []string{"test", "-v", "-tags=!integration", "-short", "-coverprofile=coverage/coverage.out", "-covermode=atomic"}
	if isCI {
		args = append(args, "-race") // Add race detection for CI
	}
	args = append(args, "./...")

	if err := sh.RunV("go", args...); err != nil {
		return err
	}

	// Get and display coverage percentage
	output, err := sh.Output("go", "tool", "cover", "-func=coverage/coverage.out")
	if err != nil {
		return fmt.Errorf("failed to get coverage: %w", err)
	}

	fmt.Print(output)

	// In CI, validate coverage threshold
	if isCI {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		totalLine := lines[len(lines)-1]

		// Extract percentage from "total: (statements) XX.X%"
		parts := strings.Fields(totalLine)
		if len(parts) == 0 {
			return fmt.Errorf("failed to parse coverage: empty output line")
		}
		percentStr := strings.TrimSuffix(parts[len(parts)-1], "%")
		percentage, err := strconv.ParseFloat(percentStr, 64)
		if err != nil {
			return fmt.Errorf("failed to parse coverage: %w", err)
		}

		if percentage < 70.0 {
			return fmt.Errorf("coverage %.1f%% below 70%% threshold", percentage)
		}
	}

	if isCI {
		fmt.Println("Coverage requirement (70%+) met successfully!")
	}
	return nil
}

// Dev builds the main Grove binary for development.
func (Build) Dev() error {
	fmt.Println("Building Grove...")

	if err := os.MkdirAll("bin", fs.DirGit); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	return sh.RunV("go", "build", "-o", "bin/grove", "./cmd/grove")
}

// Release builds release binaries for common platforms.
func (Build) Release() error {
	fmt.Println("Building release binaries...")

	if err := os.MkdirAll("bin", fs.DirGit); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	platforms := []struct {
		os   string
		arch string
	}{
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"darwin", "amd64"},
		{"darwin", "arm64"},
	}

	for _, platform := range platforms {
		output := fmt.Sprintf("bin/grove-%s-%s", platform.os, platform.arch)

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

// Format formats code and documentation files.
func Format() error {
	fmt.Println("Formatting code...")
	if err := sh.RunV("gofmt", "-w", "-s", "."); err != nil {
		return err
	}

	fmt.Println("Formatting Markdown/JSON/YAML...")
	return sh.RunV("npx", "prettier", "--write", ".")
}

// Deadcode finds unused code in the project.
func Deadcode() error {
	fmt.Println("Checking for dead code...")
	return sh.RunV("go", "run", "golang.org/x/tools/cmd/deadcode@latest", "./...")
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

// Check lists available updates for direct dependencies only.
func (Deps) Check() error {
	fmt.Println("Checking for dependency updates...")

	// Get direct dependencies with updates
	output, err := sh.Output("go", "list", "-m", "-u", "-json", "all")
	if err != nil {
		return err
	}

	// Parse and filter results
	modules, err := parseModuleUpdates(output)
	if err != nil {
		return err
	}

	directDeps, err := getDirectDependencies()
	if err != nil {
		return err
	}

	hasUpdates := false
	for _, mod := range modules {
		if directDeps[mod.Path] && mod.Update != nil {
			if !hasUpdates {
				fmt.Println("\nDirect dependencies with available updates:")
				fmt.Println("==========================================")
				hasUpdates = true
			}
			fmt.Printf("%-40s %s → %s\n", mod.Path, mod.Version, mod.Update.Version)
		}
	}

	if !hasUpdates {
		fmt.Println("✓ All direct dependencies are up to date")
	}

	return nil
}

type moduleInfo struct {
	Path    string `json:"Path"`
	Version string `json:"Version"`
	Update  *struct {
		Version string `json:"Version"`
	} `json:"Update,omitempty"`
}

func parseModuleUpdates(jsonOutput string) ([]moduleInfo, error) {
	var modules []moduleInfo

	// Split by lines and parse each JSON object
	lines := strings.Split(strings.TrimSpace(jsonOutput), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var mod moduleInfo
		if err := json.Unmarshal([]byte(line), &mod); err != nil {
			return nil, fmt.Errorf("failed to parse module: %w", err)
		}
		modules = append(modules, mod)
	}
	return modules, nil
}

func getDirectDependencies() (map[string]bool, error) {
	output, err := sh.Output("go", "mod", "edit", "-json")
	if err != nil {
		return nil, fmt.Errorf("failed to get module info: %w", err)
	}

	var modFile struct {
		Require []struct {
			Path     string `json:"Path"`
			Indirect bool   `json:"Indirect,omitempty"`
		} `json:"Require"`
	}

	if err := json.Unmarshal([]byte(output), &modFile); err != nil {
		return nil, fmt.Errorf("failed to parse module file: %w", err)
	}

	direct := make(map[string]bool)
	for _, req := range modFile.Require {
		if !req.Indirect {
			direct[req.Path] = true
		}
	}
	return direct, nil
}

// Update updates direct dependencies to latest minor/patch versions and runs tests.
func (Deps) Update() error {
	fmt.Println("Updating dependencies...")

	if err := sh.RunV("go", "get", "-u", "-t"); err != nil {
		return fmt.Errorf("failed to update dependencies: %w", err)
	}

	if err := sh.RunV("go", "mod", "tidy"); err != nil {
		return fmt.Errorf("failed to tidy modules: %w", err)
	}

	fmt.Println("Running tests after dependency update...")
	test := Test{}
	if err := test.Unit(); err != nil {
		return fmt.Errorf("unit tests failed after update: %w", err)
	}

	if err := test.Integration(); err != nil {
		return fmt.Errorf("integration tests failed after update: %w", err)
	}

	fmt.Println("Dependencies updated successfully!")
	return nil
}

// Audit scans for security vulnerabilities in dependencies.
func (Deps) Audit() error {
	fmt.Println("Scanning for security vulnerabilities...")
	return sh.RunV("go", "run", "golang.org/x/vuln/cmd/govulncheck@latest", "./...")
}
