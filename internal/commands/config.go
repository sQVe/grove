package commands

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/logger"
)

// NewConfigCmd creates the main config command with subcommands.
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage Grove configuration",
		Long: `Manage Grove configuration settings.

Available configuration sections:
  general   - General Grove settings (editor, pager, output format)
  git       - Git-related settings (timeouts, retries, default remote)
  retry     - Retry behavior configuration
  logging   - Logging level and format
  worktree  - Worktree naming and cleanup settings

Examples:
  grove config list                     # Show all configuration
  grove config get general.editor      # Get a specific value
  grove config set git.max_retries 5   # Set a configuration value
  grove config validate                # Validate current configuration
  grove config path                    # Show config file paths`,
	}

	// Add subcommands
	cmd.AddCommand(newConfigGetCmd())
	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigListCmd())
	cmd.AddCommand(newConfigValidateCmd())
	cmd.AddCommand(newConfigResetCmd())
	cmd.AddCommand(newConfigPathCmd())
	cmd.AddCommand(newConfigInitCmd())

	return cmd
}

// newConfigGetCmd creates the config get subcommand.
func newConfigGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long: `Get a configuration value by key.

Examples:
  grove config get general.editor
  grove config get git.fetch_timeout
  grove config get logging.level`,
		Args: cobra.ExactArgs(1),
		RunE: runConfigGet,
	}

	cmd.Flags().Bool("default", false, "Show the default value instead of the current value")

	return cmd
}

// newConfigSetCmd creates the config set subcommand.
func newConfigSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value by key.

Examples:
  grove config set general.editor vim
  grove config set git.fetch_timeout 60s
  grove config set logging.level debug`,
		Args: cobra.ExactArgs(2),
		RunE: runConfigSet,
	}

	return cmd
}

// newConfigListCmd creates the config list subcommand.
func newConfigListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configuration values",
		Long: `List all configuration values.

Examples:
  grove config list                 # Show all configuration in text format
  grove config list --format=json  # Show all configuration in JSON format`,
		RunE: runConfigList,
	}

	cmd.Flags().String("format", "text", "Output format (text, json)")
	cmd.Flags().Bool("defaults", false, "Show default values instead of current values")

	return cmd
}

// newConfigValidateCmd creates the config validate subcommand.
func newConfigValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate the current configuration",
		Long: `Validate the current configuration and report any errors.

This command checks all configuration values against their validation rules
and reports any issues found.`,
		RunE: runConfigValidate,
	}

	return cmd
}

// newConfigResetCmd creates the config reset subcommand.
func newConfigResetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset [key]",
		Short: "Reset configuration to defaults",
		Long: `Reset configuration to default values.

If a key is specified, only that configuration value is reset.
If no key is specified, all configuration is reset to defaults.

Examples:
  grove config reset                    # Reset all configuration
  grove config reset general.editor    # Reset only the editor setting`,
		Args: cobra.MaximumNArgs(1),
		RunE: runConfigReset,
	}

	cmd.Flags().Bool("confirm", false, "Skip confirmation prompt")

	return cmd
}

// newConfigPathCmd creates the config path subcommand.
func newConfigPathCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Show configuration file paths",
		Long: `Show the paths where Grove looks for configuration files.

This includes the currently used config file (if any) and all search paths.`,
		RunE: runConfigPath,
	}

	return cmd
}

// newConfigInitCmd creates the config init subcommand.
func newConfigInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a default configuration file",
		Long: `Create a default configuration file with all default values.

This creates a configuration file in the user's config directory with
all configuration options set to their default values.`,
		RunE: runConfigInit,
	}

	cmd.Flags().Bool("force", false, "Overwrite existing configuration file")

	return cmd
}

// runConfigGet handles the config get command.
func runConfigGet(cmd *cobra.Command, args []string) error {
	log := logger.WithComponent("config_get")
	key := args[0]

	log.Debug("getting configuration value", "key", key)

	// Validate key
	if !config.IsValidKey(key) {
		return fmt.Errorf("invalid configuration key: %s", key)
	}

	showDefault, _ := cmd.Flags().GetBool("default")

	var value interface{}
	if showDefault {
		// Get default value
		defaultConfig := config.DefaultConfig()
		value = getConfigValueByKey(defaultConfig, key)
	} else {
		// Get current value
		if err := config.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize config: %w", err)
		}

		value = getConfigValueFromViper(key)
	}

	if value == nil {
		return fmt.Errorf("configuration key not found: %s", key)
	}

	cmd.Printf("%v\n", value)
	return nil
}

// runConfigSet handles the config set command.
func runConfigSet(cmd *cobra.Command, args []string) error {
	log := logger.WithComponent("config_set")
	key := args[0]
	valueStr := args[1]

	log.Debug("setting configuration value", "key", key, "value", valueStr)

	// Validate key
	if !config.IsValidKey(key) {
		return fmt.Errorf("invalid configuration key: %s", key)
	}

	// Initialize config
	if err := config.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	// Parse value based on key type
	value, err := parseConfigValue(key, valueStr)
	if err != nil {
		return fmt.Errorf("invalid value for key %s: %w", key, err)
	}

	// Validate the value
	if err := config.ValidateKey(key, value); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Set the value
	config.Set(key, value)

	// Write the configuration
	if err := config.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	cmd.Printf("Set %s = %v\n", key, value)
	return nil
}

// runConfigList handles the config list command.
func runConfigList(cmd *cobra.Command, args []string) error {
	log := logger.WithComponent("config_list")

	format, _ := cmd.Flags().GetString("format")
	showDefaults, _ := cmd.Flags().GetBool("defaults")

	log.Debug("listing configuration", "format", format, "show_defaults", showDefaults)

	var configData map[string]interface{}

	if showDefaults {
		// Get default configuration
		defaultConfig := config.DefaultConfig()
		configData = structToMap(defaultConfig)
	} else {
		// Initialize and get current configuration
		if err := config.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize config: %w", err)
		}
		configData = config.AllSettings()
	}

	switch format {
	case "json":
		data, err := json.MarshalIndent(configData, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config to JSON: %w", err)
		}
		cmd.Println(string(data))
	case "text":
		printConfigText(cmd, configData)
	default:
		return fmt.Errorf("unsupported format: %s (supported: text, json)", format)
	}

	return nil
}

// runConfigValidate handles the config validate command.
func runConfigValidate(cmd *cobra.Command, args []string) error {
	log := logger.WithComponent("config_validate")

	log.Debug("validating configuration")

	if err := config.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	if err := config.Validate(); err != nil {
		cmd.Printf("Configuration validation failed:\n%v\n", err)
		return fmt.Errorf("configuration is invalid")
	}

	cmd.Println("Configuration is valid")
	return nil
}

// runConfigReset handles the config reset command.
func runConfigReset(cmd *cobra.Command, args []string) error {
	log := logger.WithComponent("config_reset")

	confirm, _ := cmd.Flags().GetBool("confirm")

	if len(args) == 0 {
		// Reset all configuration
		log.Debug("resetting all configuration")

		if !confirm {
			cmd.Print("This will reset all configuration to defaults. Continue? (y/N): ")
			var response string
			if _, err := fmt.Scanln(&response); err != nil {
				// User cancelled or input error, treat as "no"
				cmd.Println("Cancelled")
				return nil
			}
			if !strings.EqualFold(response, "y") && !strings.EqualFold(response, "yes") {
				cmd.Println("Cancelled")
				return nil
			}
		}

		// Initialize with defaults
		config.SetDefaults()
		if err := config.WriteConfig(); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		cmd.Println("All configuration reset to defaults")
	} else {
		// Reset specific key
		key := args[0]
		log.Debug("resetting configuration key", "key", key)

		if !config.IsValidKey(key) {
			return fmt.Errorf("invalid configuration key: %s", key)
		}

		// Initialize config
		if err := config.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize config: %w", err)
		}

		// Get default value
		defaultConfig := config.DefaultConfig()
		defaultValue := getConfigValueByKey(defaultConfig, key)

		// Set to default
		config.Set(key, defaultValue)

		// Write the configuration
		if err := config.WriteConfig(); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		cmd.Printf("Reset %s to default value: %v\n", key, defaultValue)
	}

	return nil
}

// runConfigPath handles the config path command.
func runConfigPath(cmd *cobra.Command, args []string) error {
	log := logger.WithComponent("config_path")

	log.Debug("showing configuration paths")

	// Get config paths
	paths := config.GetConfigPaths()

	cmd.Println("Configuration file search paths:")
	for i, path := range paths {
		cmd.Printf("  %d. %s\n", i+1, path)
	}

	// Show currently used config file
	if err := config.Initialize(); err != nil {
		cmd.Printf("\nWarning: failed to initialize config: %v\n", err)
		return nil
	}

	usedFile := config.ConfigFileUsed()
	if usedFile != "" {
		cmd.Printf("\nCurrently used config file:\n  %s\n", usedFile)
	} else {
		cmd.Printf("\nNo config file found, using defaults\n")
	}

	return nil
}

// runConfigInit handles the config init command.
func runConfigInit(cmd *cobra.Command, args []string) error {
	log := logger.WithComponent("config_init")

	force, _ := cmd.Flags().GetBool("force")

	log.Debug("initializing configuration file", "force", force)

	// Initialize config system
	if err := config.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	// Check if config file already exists
	usedFile := config.ConfigFileUsed()
	if usedFile != "" && !force {
		return fmt.Errorf("configuration file already exists at %s (use --force to overwrite)", usedFile)
	}

	// Set all defaults
	config.SetDefaults()

	// Write configuration file
	if err := config.SafeWriteConfig(); err != nil {
		if err := config.WriteConfig(); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}
	}

	// Show the created file path
	newFile := config.ConfigFileUsed()
	cmd.Printf("Created configuration file: %s\n", newFile)

	return nil
}

// Helper functions.

// parseConfigValue parses a string value into the appropriate type for the given config key.
func parseConfigValue(key, valueStr string) (interface{}, error) {
	// Get the expected type from the default config
	defaultConfig := config.DefaultConfig()
	defaultValue := getConfigValueByKey(defaultConfig, key)

	if defaultValue == nil {
		return nil, fmt.Errorf("unknown configuration key: %s", key)
	}

	switch defaultValue.(type) {
	case string:
		return valueStr, nil
	case int:
		return strconv.Atoi(valueStr)
	case bool:
		return strconv.ParseBool(valueStr)
	case time.Duration:
		return time.ParseDuration(valueStr)
	default:
		return nil, fmt.Errorf("unsupported type for key %s", key)
	}
}

// getConfigValueByKey gets a value from a config struct using dot notation with comprehensive safety checks.
func getConfigValueByKey(cfg *config.Config, key string) interface{} {
	// Add panic recovery for reflection operations
	defer func() {
		if r := recover(); r != nil {
			// Log the panic but don't crash the application
			fmt.Printf("Warning: reflection panic in config access: %v\n", r)
		}
	}()

	// Validate input parameters
	if cfg == nil {
		return nil
	}
	if key == "" {
		return nil
	}

	parts := strings.Split(key, ".")
	if len(parts) != 2 {
		return nil
	}

	section, field := parts[0], parts[1]
	if section == "" || field == "" {
		return nil
	}

	// Safe reflection with comprehensive checks
	v := reflect.ValueOf(cfg)
	if !v.IsValid() {
		return nil
	}

	// Check if it's a pointer and is not nil
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return nil
	}

	// Get the element safely
	v = v.Elem()
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return nil
	}

	// Convert section name to struct field name
	sectionName := capitalizeFirst(section)
	sectionValue := v.FieldByName(sectionName)
	if !sectionValue.IsValid() || !sectionValue.CanInterface() {
		return nil
	}

	// Ensure section is a struct
	if sectionValue.Kind() != reflect.Struct {
		return nil
	}

	// Convert field name to struct field name
	fieldName := convertFieldName(field)
	fieldValue := sectionValue.FieldByName(fieldName)
	if !fieldValue.IsValid() || !fieldValue.CanInterface() {
		return nil
	}

	return fieldValue.Interface()
}

// convertFieldName converts config field names to Go struct field names.
func convertFieldName(field string) string {
	switch field {
	case "editor":
		return "Editor"
	case "pager":
		return "Pager"
	case "output_format":
		return "OutputFormat"
	case "default_remote":
		return "DefaultRemote"
	case "fetch_timeout":
		return "FetchTimeout"
	case "max_retries":
		return "MaxRetries"
	case "max_attempts":
		return "MaxAttempts"
	case "base_delay":
		return "BaseDelay"
	case "max_delay":
		return "MaxDelay"
	case "jitter_enabled":
		return "Jitter"
	case "level":
		return "Level"
	case "format":
		return "Format"
	case "naming_pattern":
		return "NamingPattern"
	case "cleanup_threshold":
		return "CleanupThreshold"
	default:
		// Fallback: convert underscore to title case
		parts := strings.Split(field, "_")
		for i, part := range parts {
			parts[i] = capitalizeFirst(part)
		}
		return strings.Join(parts, "")
	}
}

// capitalizeFirst capitalizes the first letter of a string.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// getConfigValueFromViper gets a value from viper using the key.
func getConfigValueFromViper(key string) interface{} {
	if !config.IsSet(key) {
		return nil
	}

	// Get the expected type from the default config to know what type to return
	defaultConfig := config.DefaultConfig()
	defaultValue := getConfigValueByKey(defaultConfig, key)

	if defaultValue == nil {
		return nil
	}

	// Return the value in the same type as the default
	switch defaultValue.(type) {
	case string:
		return config.GetString(key)
	case int:
		return config.GetInt(key)
	case bool:
		return config.GetBool(key)
	case time.Duration:
		return config.GetDuration(key)
	default:
		return config.GetString(key) // fallback to string
	}
}

// structToMap converts a struct to a map using reflection.
func structToMap(obj interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Get the mapstructure tag or use the field name
		tag := fieldType.Tag.Get("mapstructure")
		if tag == "" {
			tag = strings.ToLower(fieldType.Name)
		}

		if field.Kind() == reflect.Struct {
			// Recursively handle nested structs
			nested := structToMap(field.Interface())
			result[tag] = nested
		} else {
			result[tag] = field.Interface()
		}
	}

	return result
}

// printConfigText prints configuration in a readable text format.
func printConfigText(cmd *cobra.Command, data map[string]interface{}) {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, section := range keys {
		cmd.Printf("[%s]\n", section)
		if sectionData, ok := data[section].(map[string]interface{}); ok {
			sectionKeys := make([]string, 0, len(sectionData))
			for k := range sectionData {
				sectionKeys = append(sectionKeys, k)
			}
			sort.Strings(sectionKeys)

			for _, key := range sectionKeys {
				value := sectionData[key]
				cmd.Printf("  %s = %v\n", key, value)
			}
		}
		cmd.Println()
	}
}
