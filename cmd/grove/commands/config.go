package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/workspace"
)

const (
	configKeyPlain    = "grove.plain"
	configKeyDebug    = "grove.debug"
	configKeyPreserve = "grove.preserve"
	configKeyHooksAdd = "hooks.add"
	tomlKeyPlain      = "plain"
	tomlKeyDebug      = "debug"
	tomlKeyPreserve   = "preserve.patterns"
)

var (
	allConfigKeys     = []string{configKeyPlain, configKeyDebug, configKeyPreserve}
	booleanConfigKeys = []string{configKeyPlain, configKeyDebug}
	multiValueKeys    = []string{configKeyPreserve}
)

// isValidConfigKey validates that key is in grove.* namespace
func isValidConfigKey(key string) bool {
	if key == "" {
		return false
	}
	return strings.HasPrefix(strings.ToLower(key), "grove.")
}

// isValidBooleanValue validates boolean configuration values
func isValidBooleanValue(value string) bool {
	if value == "" {
		return false
	}
	lower := strings.ToLower(value)
	validValues := []string{"true", "false", "yes", "no", "on", "off", "1", "0"}
	return slices.Contains(validValues, lower)
}

// isMultiValueKey returns true if the key supports multiple values
func isMultiValueKey(key string) bool {
	return slices.ContainsFunc(multiValueKeys, func(k string) bool {
		return strings.EqualFold(key, k)
	})
}

// getConfigCompletions returns completion suggestions for config keys
func getConfigCompletions(toComplete string) []string {
	var completions []string
	for _, key := range allConfigKeys {
		if strings.HasPrefix(key, toComplete) {
			completions = append(completions, key)
		}
	}
	return completions
}

// getExistingConfigCompletions returns completion suggestions for existing config keys
func getExistingConfigCompletions(toComplete string, global bool) []string {
	configs, err := git.GetConfigs("grove.", global)
	if err != nil {
		return getConfigCompletions(toComplete)
	}

	var completions []string
	for key := range configs {
		if strings.HasPrefix(key, toComplete) {
			completions = append(completions, key)
		}
	}
	return completions
}

// getExistingConfigValues returns existing values for a config key
func getExistingConfigValues(key, toComplete string, global bool) []string {
	configs, err := git.GetConfigs("grove.", global)
	if err != nil {
		return nil
	}

	values, exists := configs[key]
	if !exists {
		return nil
	}

	var completions []string
	for _, value := range values {
		if strings.HasPrefix(value, toComplete) {
			completions = append(completions, value)
		}
	}
	return completions
}

// getBooleanCompletions returns completion suggestions for boolean values
func getBooleanCompletions(toComplete string) []string {
	booleans := []string{"true", "false"}
	var completions []string
	for _, b := range booleans {
		if strings.HasPrefix(b, toComplete) {
			completions = append(completions, b)
		}
	}
	return completions
}

// isBooleanKey returns true if the key expects boolean values
func isBooleanKey(key string) bool {
	return slices.ContainsFunc(booleanConfigKeys, func(k string) bool {
		return strings.EqualFold(key, k)
	})
}

// findWorktreeDir finds the current worktree directory or returns empty string
func findWorktreeDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	bareDir, err := workspace.FindBareDir(cwd)
	if err != nil {
		return ""
	}

	// Check if we're in a worktree (not workspace root)
	if git.IsWorktree(cwd) {
		return cwd
	}

	// Walk up to find worktree root
	workspaceRoot := bareDir[:len(bareDir)-5] // Remove "/.bare"
	dir := cwd
	for dir != workspaceRoot && dir != "/" {
		if git.IsWorktree(dir) {
			return dir
		}
		dir = filepath.Dir(dir)
	}

	return ""
}

// NewConfigCmd creates the config command with all subcommands
func NewConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Get and set grove configuration options",
		Long: `Manage grove configuration settings.

Configuration sources (in order of precedence for reading):
  CLI flags and environment variables (always highest)
  For team settings (preserve, hooks): .grove.toml > global git config
  For personal settings (plain, debug): global git config > .grove.toml

Use --shared to read/write .grove.toml (team settings).
Use --global to read/write global git config (personal settings).`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List configuration settings",
		Long: `List grove configuration settings.

Without flags: shows effective (merged) configuration.
With --shared: shows only .grove.toml settings.
With --global: shows only global git config settings.`,
		Args: cobra.NoArgs,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			shared, _ := cmd.Flags().GetBool("shared")
			global, _ := cmd.Flags().GetBool("global")
			return runConfigList(shared, global)
		},
	}
	listCmd.Flags().Bool("shared", false, "List only .grove.toml settings")
	listCmd.Flags().Bool("global", false, "List only global git config settings")

	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long: `Get a configuration value.

Without flags: shows effective (merged) value.
With --shared: shows value from .grove.toml only.
With --global: shows value from global git config only.`,
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return getConfigCompletions(toComplete), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			shared, _ := cmd.Flags().GetBool("shared")
			global, _ := cmd.Flags().GetBool("global")
			return runConfigGet(args[0], shared, global)
		},
	}
	getCmd.Flags().Bool("shared", false, "Get from .grove.toml")
	getCmd.Flags().Bool("global", false, "Get from global git config")

	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value.

Use --shared to write to .grove.toml (team settings).
Use --global to write to global git config (personal settings).

One of --shared or --global must be specified.

Note: Array values (preserve.patterns, hooks.add) cannot be set via this command.
Edit .grove.toml directly for array values.`,
		Args: cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return getConfigCompletions(toComplete), cobra.ShellCompDirectiveNoFileComp
			}
			if len(args) == 1 && isBooleanKey(args[0]) {
				return getBooleanCompletions(toComplete), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			shared, _ := cmd.Flags().GetBool("shared")
			global, _ := cmd.Flags().GetBool("global")
			return runConfigSet(args[0], args[1], shared, global)
		},
	}
	setCmd.Flags().Bool("shared", false, "Write to .grove.toml")
	setCmd.Flags().Bool("global", false, "Write to global git config")

	unsetCmd := &cobra.Command{
		Use:   "unset <key> [<value>]",
		Short: "Remove a configuration setting",
		Long: `Remove a configuration setting.

Use --shared to remove from .grove.toml.
Use --global to remove from global git config.

One of --shared or --global must be specified.

For multi-value keys, optionally specify a value to remove only that value.`,
		Args: cobra.RangeArgs(1, 2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			global, _ := cmd.Flags().GetBool("global")
			if len(args) == 0 {
				return getExistingConfigCompletions(toComplete, global), cobra.ShellCompDirectiveNoFileComp
			}
			if len(args) == 1 {
				return getExistingConfigValues(args[0], toComplete, global), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			shared, _ := cmd.Flags().GetBool("shared")
			global, _ := cmd.Flags().GetBool("global")
			var value string
			if len(args) > 1 {
				value = args[1]
			}
			return runConfigUnset(args[0], value, shared, global)
		},
	}
	unsetCmd.Flags().Bool("shared", false, "Remove from .grove.toml")
	unsetCmd.Flags().Bool("global", false, "Remove from global git config")

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Create a .grove.toml template",
		Long: `Create a .grove.toml configuration file with default values.

Must be run from inside a worktree (not workspace root).
Skips if .grove.toml already exists (use --force to overwrite).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")
			return runConfigInit(force)
		},
	}
	initCmd.Flags().Bool("force", false, "Overwrite existing .grove.toml")

	configCmd.AddCommand(listCmd, getCmd, setCmd, unsetCmd, initCmd)
	return configCmd
}

func runConfigList(shared, global bool) error {
	if shared && global {
		return errors.New("cannot use both --shared and --global")
	}

	if shared {
		return runConfigListShared()
	}

	if global {
		return runConfigListGlobal()
	}

	// Default: show effective config
	return runConfigListEffective()
}

func runConfigListShared() error {
	worktreeDir := findWorktreeDir()
	if worktreeDir == "" {
		return errors.New("not in a worktree (run from inside a worktree to access .grove.toml)")
	}

	cfg, err := config.LoadFromFile(worktreeDir)
	if err != nil {
		return err
	}

	// Print TOML values
	if cfg.Plain {
		fmt.Println("plain=true")
	}
	if cfg.Debug {
		fmt.Println("debug=true")
	}
	for _, p := range cfg.Preserve.Patterns {
		fmt.Printf("preserve.patterns=%s\n", p)
	}
	for _, h := range cfg.Hooks.Add {
		fmt.Printf("hooks.add=%s\n", h)
	}

	return nil
}

func runConfigListGlobal() error {
	configs, err := git.GetConfigs("grove.", true)
	if err != nil {
		return err
	}

	if len(configs) == 0 {
		return nil
	}

	keys := make([]string, 0, len(configs))
	for key := range configs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		values := configs[key]
		for _, value := range values {
			fmt.Printf("%s=%s\n", key, value)
		}
	}

	return nil
}

func runConfigListEffective() error {
	worktreeDir := findWorktreeDir()

	// Show plain
	plain := config.GetMergedPlain(worktreeDir)
	if plain {
		fmt.Println("grove.plain=true")
	}

	// Show debug
	debug := config.GetMergedDebug(worktreeDir)
	if debug {
		fmt.Println("grove.debug=true")
	}

	// Show preserve patterns
	patterns := config.GetMergedPreservePatterns(worktreeDir)
	for _, p := range patterns {
		fmt.Printf("grove.preserve=%s\n", p)
	}

	// Show hooks (from TOML only)
	if worktreeDir != "" {
		cfg, err := config.LoadFromFile(worktreeDir)
		if err == nil {
			for _, h := range cfg.Hooks.Add {
				fmt.Printf("hooks.add=%s\n", h)
			}
		}
	}

	return nil
}

func runConfigGet(key string, shared, global bool) error {
	if !isValidConfigKey(key) && !strings.HasPrefix(key, "preserve.") && !strings.HasPrefix(key, "hooks.") {
		return errors.New("only grove.* settings are supported")
	}

	if shared && global {
		return errors.New("cannot use both --shared and --global")
	}

	if shared {
		return runConfigGetShared(key)
	}

	if global {
		return runConfigGetGlobal(key)
	}

	// Default: get effective value
	return runConfigGetEffective(key)
}

func runConfigGetShared(key string) error {
	worktreeDir := findWorktreeDir()
	if worktreeDir == "" {
		return errors.New("not in a worktree (run from inside a worktree to access .grove.toml)")
	}

	cfg, err := config.LoadFromFile(worktreeDir)
	if err != nil {
		return err
	}

	switch strings.ToLower(key) {
	case configKeyPlain, tomlKeyPlain:
		fmt.Println(cfg.Plain)
	case configKeyDebug, tomlKeyDebug:
		fmt.Println(cfg.Debug)
	case configKeyPreserve, tomlKeyPreserve:
		for _, p := range cfg.Preserve.Patterns {
			fmt.Println(p)
		}
	case configKeyHooksAdd:
		for _, h := range cfg.Hooks.Add {
			fmt.Println(h)
		}
	default:
		return fmt.Errorf("unknown key: %s", key)
	}

	return nil
}

func runConfigGetGlobal(key string) error {
	if !isValidConfigKey(key) {
		return errors.New("only grove.* settings are supported in git config")
	}

	value, err := git.GetConfig(key, true)
	if err != nil {
		if git.IsConfigNotFoundError(err) {
			return errors.New("config key not found")
		}
		return err
	}

	fmt.Println(value)
	return nil
}

func runConfigGetEffective(key string) error {
	worktreeDir := findWorktreeDir()

	switch strings.ToLower(key) {
	case configKeyPlain, tomlKeyPlain:
		fmt.Println(config.GetMergedPlain(worktreeDir))
	case configKeyDebug, tomlKeyDebug:
		fmt.Println(config.GetMergedDebug(worktreeDir))
	case configKeyPreserve, tomlKeyPreserve:
		for _, p := range config.GetMergedPreservePatterns(worktreeDir) {
			fmt.Println(p)
		}
	case configKeyHooksAdd:
		if worktreeDir != "" {
			cfg, err := config.LoadFromFile(worktreeDir)
			if err == nil {
				for _, h := range cfg.Hooks.Add {
					fmt.Println(h)
				}
			}
		}
	default:
		// Try git config for unknown keys
		value, err := git.GetConfig(key, true)
		if err != nil {
			if git.IsConfigNotFoundError(err) {
				return errors.New("config key not found")
			}
			return err
		}
		fmt.Println(value)
	}

	return nil
}

func runConfigSet(key, value string, shared, global bool) error {
	if !shared && !global {
		return errors.New("must specify --shared or --global")
	}

	if shared && global {
		return errors.New("cannot use both --shared and --global")
	}

	if shared {
		return runConfigSetShared(key, value)
	}

	return runConfigSetGlobal(key, value)
}

func runConfigSetShared(key, value string) error {
	worktreeDir := findWorktreeDir()
	if worktreeDir == "" {
		return errors.New("not in a worktree (run from inside a worktree to access .grove.toml)")
	}

	normalizedKey := strings.ToLower(key)
	if normalizedKey != configKeyPlain && normalizedKey != tomlKeyPlain &&
		normalizedKey != configKeyDebug && normalizedKey != tomlKeyDebug {
		return fmt.Errorf("setting %s requires editing .grove.toml directly (arrays/tables not supported via set)", key)
	}

	// Validate boolean value
	if !isValidBooleanValue(value) {
		return fmt.Errorf("invalid boolean value '%s'", value)
	}

	boolValue := strings.EqualFold(value, "true") || value == "1" || strings.EqualFold(value, "yes") || strings.EqualFold(value, "on")

	cfg, err := config.LoadFromFile(worktreeDir)
	if err != nil {
		return err
	}

	switch normalizedKey {
	case "grove.plain", "plain":
		cfg.Plain = boolValue
	case "grove.debug", "debug":
		cfg.Debug = boolValue
	}

	return config.WriteToFile(worktreeDir, cfg)
}

func runConfigSetGlobal(key, value string) error {
	if !isValidConfigKey(key) {
		return errors.New("only grove.* settings are supported")
	}

	if isBooleanKey(key) && !isValidBooleanValue(value) {
		return fmt.Errorf("invalid boolean value '%s' for key '%s'", value, key)
	}

	return git.SetConfig(key, value, true)
}

func runConfigUnset(key, value string, shared, global bool) error {
	if !shared && !global {
		return errors.New("must specify --shared or --global")
	}

	if shared && global {
		return errors.New("cannot use both --shared and --global")
	}

	if shared {
		return runConfigUnsetShared(key)
	}

	return runConfigUnsetGlobal(key, value)
}

func runConfigUnsetShared(key string) error {
	worktreeDir := findWorktreeDir()
	if worktreeDir == "" {
		return errors.New("not in a worktree (run from inside a worktree to access .grove.toml)")
	}

	cfg, err := config.LoadFromFile(worktreeDir)
	if err != nil {
		return err
	}

	normalizedKey := strings.ToLower(key)
	switch normalizedKey {
	case "grove.plain", "plain":
		cfg.Plain = false
	case "grove.debug", "debug":
		cfg.Debug = false
	case "grove.preserve", "preserve.patterns":
		cfg.Preserve.Patterns = nil
	case "hooks.add":
		cfg.Hooks.Add = nil
	default:
		return fmt.Errorf("unknown key: %s", key)
	}

	return config.WriteToFile(worktreeDir, cfg)
}

func runConfigUnsetGlobal(key, value string) error {
	if !isValidConfigKey(key) {
		return errors.New("only grove.* settings are supported")
	}

	if value != "" {
		err := git.UnsetConfigValue(key, value, true)
		if git.IsConfigNotFoundError(err) {
			return nil // Unsetting nonexistent value is a no-op
		}
		return err
	}

	err := git.UnsetConfig(key, true)
	if git.IsConfigNotFoundError(err) {
		return nil // Unsetting nonexistent key is a no-op
	}
	return err
}

func runConfigInit(force bool) error {
	worktreeDir := findWorktreeDir()
	if worktreeDir == "" {
		return errors.New("not in a worktree (run from inside a worktree)")
	}

	if config.FileConfigExists(worktreeDir) && !force {
		logger.Info(".grove.toml already exists (use --force to overwrite)")
		return nil
	}

	// Create template config
	cfg := config.FileConfig{}
	cfg.Preserve.Patterns = config.DefaultConfig.PreservePatterns

	if err := config.WriteToFile(worktreeDir, cfg); err != nil {
		return fmt.Errorf("failed to write .grove.toml: %w", err)
	}

	logger.Success("Created .grove.toml")
	return nil
}
