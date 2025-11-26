package commands

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/git"
)

const (
	configKeyPlain    = "grove.plain"
	configKeyDebug    = "grove.debug"
	configKeyPreserve = "grove.convert.preserve"
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

// NewConfigCmd creates the config command with all subcommands
func NewConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Get and set grove configuration options",
		Long:  `Manage grove configuration settings using git config system.`,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all grove.* configuration settings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			global, _ := cmd.Flags().GetBool("global")
			return runConfigList(global)
		},
	}
	listCmd.Flags().Bool("global", false, "List only global settings")

	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a specific configuration value",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return getConfigCompletions(toComplete), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			global, _ := cmd.Flags().GetBool("global")
			return runConfigGet(args[0], global)
		},
	}
	getCmd.Flags().Bool("global", false, "Get global setting")

	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
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
			global, _ := cmd.Flags().GetBool("global")
			return runConfigSet(args[0], args[1], global)
		},
	}
	setCmd.Flags().Bool("global", false, "Set global setting")

	addCmd := &cobra.Command{
		Use:   "add <key> <value>",
		Short: "Add to a multi-value configuration key",
		Args:  cobra.ExactArgs(2),
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
			global, _ := cmd.Flags().GetBool("global")
			return runConfigAdd(args[0], args[1], global)
		},
	}
	addCmd.Flags().Bool("global", false, "Add to global setting")

	unsetCmd := &cobra.Command{
		Use:   "unset <key> [<value>]",
		Short: "Remove a configuration setting (optionally by value pattern)",
		Args:  cobra.RangeArgs(1, 2),
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
			global, _ := cmd.Flags().GetBool("global")
			var value string
			if len(args) > 1 {
				value = args[1]
			}
			return runConfigUnset(args[0], value, global)
		},
	}
	unsetCmd.Flags().Bool("global", false, "Remove global setting")

	configCmd.AddCommand(listCmd, getCmd, setCmd, addCmd, unsetCmd)
	return configCmd
}

func runConfigList(global bool) error {
	configs, err := git.GetConfigs("grove.", global)
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

func runConfigGet(key string, global bool) error {
	if !isValidConfigKey(key) {
		return errors.New("only grove.* settings are supported")
	}

	value, err := git.GetConfig(key, global)
	if err != nil {
		if git.IsConfigNotFoundError(err) {
			return errors.New("config key not found")
		}
		return err
	}

	fmt.Println(value)
	return nil
}

func runConfigSet(key, value string, global bool) error {
	if !isValidConfigKey(key) {
		return errors.New("only grove.* settings are supported")
	}

	if isBooleanKey(key) && !isValidBooleanValue(value) {
		return fmt.Errorf("invalid boolean value '%s' for key '%s'", value, key)
	}

	return git.SetConfig(key, value, global)
}

func runConfigAdd(key, value string, global bool) error {
	if !isValidConfigKey(key) {
		return errors.New("only grove.* settings are supported")
	}

	return git.AddConfig(key, value, global)
}

func runConfigUnset(key, value string, global bool) error {
	if !isValidConfigKey(key) {
		return errors.New("only grove.* settings are supported")
	}

	if value != "" {
		err := git.UnsetConfigValue(key, value, global)
		if git.IsConfigNotFoundError(err) {
			return nil // Unsetting nonexistent value is a no-op
		}
		return err
	}

	err := git.UnsetConfig(key, global)
	if git.IsConfigNotFoundError(err) {
		return nil // Unsetting nonexistent key is a no-op
	}
	return err
}
