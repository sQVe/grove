# Technical Standards

## Technology Stack

### Core Technologies

- **Language**: Go 1.24.5+ (no specific compatibility constraints)
- **CLI Framework**: Cobra for command structure + Viper for configuration
- **UI/Styling**: Charm Bracelet's Lipgloss for terminal styling
- **Configuration**: TOML/JSON with environment variable overrides (`GROVE_*`)
- **Testing**: testify framework with comprehensive coverage
- **Build System**: Mage for development automation

### Dependencies Philosophy

- Minimal external dependencies
- Prefer standard library when possible
- Choose mature, well-maintained libraries
- No specific version constraints beyond current stack

## Performance Requirements

### Large Repository Support

- Must handle repositories with 100+ worktrees efficiently
- Operations should remain responsive regardless of repository size
- Memory usage should scale reasonably with repository complexity
- Parallel operations where safe and beneficial

### Speed Requirements

- All operations must be "snappy and quick"
- No waiting for common operations (list, switch, status)
- Progress indicators for longer operations (clone, large cleanups)
- Optimize for developer productivity over absolute performance

## Development Standards

### Code Quality

- Follow Go best practices and idioms
- Use golangci-lint with strict standards (current configuration)
- Maintain 90%+ test coverage (currently 96.4%)
- Comprehensive error handling with standardized error codes

### Architecture Patterns

- Standard Go project layout (`/cmd`, `/internal`, `/docs`)
- Package separation: commands, config, git, utils, testutils
- Configuration-driven design with defaults and validation
- Retry mechanisms with exponential backoff for network operations

### Debugging & Observability

- Structured logging for easy debugging
- Clear error messages with actionable context
- Configurable log levels and output formats
- Filesystem-safe operations with proper error recovery

## Platform Support

- Cross-platform compatibility: Windows, macOS, Linux
- Platform-specific optimizations where beneficial
- Consistent behavior across all supported platforms
