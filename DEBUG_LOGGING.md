# Debug Logging

Grove provides structured debug logging to troubleshoot git operations and repository issues.

## Enable Debug Logging

```bash
# Command line flag
grove --debug init https://github.com/user/repo.git

# Environment variable  
GROVE_DEBUG=1 grove init https://github.com/user/repo.git

# JSON format for parsing
grove --log-format=json --debug init
```

## Key Components

- **`init_command`**: Repository initialization and cloning
- **`git_executor`**: All git command execution with timing
- **`default_branch`**: Multi-tier branch detection strategy
- **`git_utils`**: Repository validation and URL parsing
- **`system_utils`**: Git availability checks

## Common Issues

### Git Not Found
```
level=ERROR component=system_utils msg="git not found in PATH"
```
**Solution**: Install git or add to PATH

### Network Timeouts
```
level=DEBUG component=default_branch msg="context deadline exceeded"
```
**Solution**: Check network connectivity or use local repository

### Repository Not Found
```
level=ERROR component=git_executor msg="repository not found"  
```
**Solution**: Verify URL is correct and accessible

### Branch Detection Issues
```
level=WARN component=default_branch msg="using fallback" branch=main method=fallback
```
**Solution**: Explicitly set default branch or check remote configuration

## Debug Log Format

```
time=2024-01-01T12:00:00Z level=DEBUG msg="git command" component=git_executor git_command=clone git_args="[--bare https://github.com/user/repo.git]" duration=2.1s
```

Key attributes:
- `component`: Which part of Grove generated the log
- `duration`: How long operations took
- `error`: Error details when operations fail
- `git_command`/`git_args`: Exact git commands executed

## Performance Analysis

Enable debug logging to identify slow operations:
- Git command timing in `duration` field
- Network vs local operation differentiation
- Multi-tier fallback performance in default branch detection

## Style Guidelines

### Component Naming
- Use snake_case: `git_utils`, `init_command`, `default_branch`
- Keep names descriptive but concise
- Group related functionality: `git_executor`, `git_clone`, `git_file`

### Log Messages
- Use sentence case without periods: `"checking git availability"`
- Be specific and actionable: `"git not found in PATH"` vs `"error"`
- Include context in structured attributes, not message text

### Structured Attributes
- Use consistent naming: `duration`, `component`, `git_command`, `error`
- Include timing for operations: `"duration", time.Since(start)`
- Add relevant context: `"git_args", args`, `"target_dir", path`

### Log Levels
- **Debug**: Detailed operation flow, git commands, validation steps
- **Info**: Major operation completion with summary
- **Warn**: Fallback behaviors, non-critical failures
- **Error**: Operation failures requiring user action

### Example Pattern
```go
log := logger.WithComponent("git_utils")
start := time.Now()

log.DebugOperation("checking if current directory is git repository")

output, err := executor.Execute("rev-parse", "--git-dir")
if err != nil {
    log.ErrorOperation("git repository check failed", err, "duration", time.Since(start))
    return false, err
}

log.Debug("git repository detected", "git_dir", strings.TrimSpace(output), "duration", time.Since(start))
return true, nil
```

## Security Note

Debug logs contain repository URLs and paths but no authentication credentials.