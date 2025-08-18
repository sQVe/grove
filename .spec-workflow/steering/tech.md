# Grove Technical Architecture & Standards

## Core Technology Stack

### Language & Runtime
- **Go 1.24.5**: Primary language for cross-platform CLI development
- **Minimum Go version**: 1.21+ for compatibility with CI/CD environments
- **Standard library first**: Prefer Go stdlib over external dependencies where possible

### CLI Framework & Dependencies
- **spf13/cobra**: CLI framework for command structure and argument parsing
- **spf13/viper**: Configuration management with TOML/JSON support and environment variables
- **pelletier/go-toml/v2**: TOML configuration file parsing
- **charmbracelet/lipgloss**: Terminal UI styling for consistent visual presentation
- **stretchr/testify**: Testing framework with rich assertions and mocking
- **magefile/mage**: Cross-platform build automation

### Development Tools
- **golangci-lint 1.50+**: Comprehensive linting with strict quality standards
- **gofumpt + goimports**: Automatic code formatting and import organization
- **pre-commit hooks**: Optional automated quality checks before commits

