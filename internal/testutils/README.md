# Robust Testing Infrastructure

This package provides utilities to make tests more reliable and less brittle by addressing common testing issues.

## Common Testing Problems Solved

### 1. Working Directory Dependencies

- **Problem**: Tests fail when run from different directories
- **Solution**: Use `IntegrationTestHelper` which finds project root automatically

### 2. Filesystem State Pollution

- **Problem**: Previous test runs leave artifacts that interfere with new tests
- **Solution**: Use `WithCleanFilesystem()` to clean up before tests

### 3. Environment Isolation

- **Problem**: Tests affected by environment variables or global state
- **Solution**: Use `TestRunner.WithCleanEnvironment()` for isolation

### 4. Path Conflicts

- **Problem**: Multiple tests use the same paths causing conflicts
- **Solution**: Use `UnitTestHelper.GetUniqueTestPath()` for unique paths

## Usage Examples

### Integration Tests

```go
func TestMyCommand_Integration(t *testing.T) {
    // Create robust integration test helper
    helper := testutils.NewIntegrationTestHelper(t).WithCleanFilesystem()

    // Execute command with isolation
    stdout, stderr, err := helper.ExecGrove("--version")
    require.NoError(t, err)

    assert.Equal(t, "grove version v0.1.0\n", stdout)
    assert.Empty(t, stderr)
}
```

### Unit Tests with Filesystem Operations

```go
func TestPathValidation_Unit(t *testing.T) {
    // Create robust unit test helper
    helper := testutils.NewUnitTestHelper(t).
        WithCleanFilesystem().
        WithIsolatedPath()

    // Use unique paths to avoid conflicts
    testPath := helper.GetUniqueTestPath("test-directory")

    // Your test logic here
    err := validatePath(testPath)
    assert.NoError(t, err)
}
```

### Environment Isolation

```go
func TestWithCleanEnvironment(t *testing.T) {
    runner := testutils.NewTestRunner(t).
        WithCleanEnvironment().
        WithCleanFilesystem().
        WithIsolatedWorkingDir()

    runner.Run(func() {
        // Test runs in isolated environment
        // with clean filesystem and working directory
    })
}
```

## Best Practices

### 1. Always Use Helpers for Integration Tests

```go
// ❌ BAD: Direct binary execution
cmd := exec.Command("./grove", "--version")

// ✅ GOOD: Use helper
helper := testutils.NewIntegrationTestHelper(t)
stdout, stderr, err := helper.ExecGrove("--version")
```

### 2. Clean Filesystem Before Tests

```go
// ✅ GOOD: Clean potential conflicts
helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
```

### 3. Use Unique Paths for Unit Tests

```go
// ❌ BAD: Hardcoded paths that may conflict
testPath := "/tmp/grove-test/my-test"

// ✅ GOOD: Unique paths per test
testPath := helper.GetUniqueTestPath("my-test")
```

### 4. Isolate Working Directory Changes

```go
// ❌ BAD: Direct directory changes
os.Chdir("/tmp")

// ✅ GOOD: Use runner with isolation
runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()
```

## API Reference

### IntegrationTestHelper

- `NewIntegrationTestHelper(t)` - Creates new helper
- `WithCleanFilesystem(patterns...)` - Cleans filesystem before test
- `GetBinary()` - Builds and returns binary path (cached)
- `ExecGrove(args...)` - Executes grove with arguments
- `ExecGroveInDir(dir, args...)` - Executes grove in specific directory
- `GetTempDir()` - Returns isolated temp directory

### UnitTestHelper

- `NewUnitTestHelper(t)` - Creates new helper
- `WithCleanFilesystem(patterns...)` - Cleans filesystem
- `WithIsolatedPath()` - Ensures path isolation
- `GetTempDir()` - Returns test temp directory
- `CreateTempFile(name, content)` - Creates temporary file
- `CreateTempDir(path)` - Creates temporary directory
- `GetUniqueTestPath(suffix)` - Returns unique test path
- `AssertFileExists(path)` - Asserts file exists
- `AssertNoFileExists(path)` - Asserts file doesn't exist

### TestRunner

- `NewTestRunner(t)` - Creates new runner
- `WithCleanEnvironment()` - Isolates environment variables
- `WithCleanFilesystem(patterns...)` - Cleans filesystem
- `WithIsolatedWorkingDir()` - Isolates working directory
- `Run(testFn)` - Executes test function with isolation

## Migration Guide

### From Old Integration Tests

```go
// OLD: Manual binary building and execution
func execGrove(t *testing.T, args ...string) (string, string, error) {
    binaryPath := buildGroveBinary(t)
    cmd := exec.Command(binaryPath, args...)
    // ... complex setup
}

// NEW: Use helper
func TestMyFeature(t *testing.T) {
    helper := testutils.NewIntegrationTestHelper(t).WithCleanFilesystem()
    stdout, stderr, err := helper.ExecGrove("--version")
    // ... test logic
}
```

### From Old Unit Tests

```go
// OLD: Hardcoded paths and manual cleanup
func TestPathValidation(t *testing.T) {
    testPath := "/tmp/grove-test"
    defer os.RemoveAll(testPath)
    // ... test logic
}

// NEW: Use helper with unique paths
func TestPathValidation(t *testing.T) {
    helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
    testPath := helper.GetUniqueTestPath("validation-test")
    // ... test logic (automatic cleanup)
}
```

## Debugging Test Issues

### Check Build Process

```go
// Enable verbose build output
helper := testutils.NewIntegrationTestHelper(t)
binary := helper.GetBinary() // Will show build errors if any
```

### Check Environment

```go
// Test in clean environment
runner := testutils.NewTestRunner(t).WithCleanEnvironment()
runner.Run(func() {
    // Test logic here
})
```

### Check Filesystem State

```go
// Clean all potential conflicts
helper.WithCleanFilesystem(
    "/tmp/my-app-*",
    "/tmp/test-*",
    "/var/tmp/my-app-*",
)
```
