package git

import (
	"bufio"
	"errors"
	"regexp"
	"strings"

	"github.com/sqve/grove/internal/logger"
)

// ErrConfigNotFound is returned when a config key is not found
var ErrConfigNotFound = errors.New("config key not found")

const configCommand = "config"

// IsConfigNotFoundError returns true if error indicates config not found
func IsConfigNotFoundError(err error) bool {
	return errors.Is(err, ErrConfigNotFound)
}

// GetConfig gets a single config value
func GetConfig(key string, global bool) (string, error) {
	logger.Debug("Getting git config: %s (global=%v)", key, global)

	args := []string{configCommand, "--get"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key)

	cmd, cancel := GitCommand("git", args...)
	defer cancel()
	output, err := executeWithOutput(cmd)
	if err != nil {
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
			return "", ErrConfigNotFound
		}
		return "", err
	}

	return output, nil
}

// GetConfigs gets all config values for keys with a given prefix
func GetConfigs(prefix string, global bool) (map[string][]string, error) {
	logger.Debug("Getting git configs with prefix: %s (global=%v)", prefix, global)

	args := []string{configCommand, "--get-regexp"}
	if global {
		args = append(args, "--global")
	}
	// Escape regex metacharacters and anchor to start for exact prefix matching
	args = append(args, "^"+regexp.QuoteMeta(prefix))

	cmd, cancel := GitCommand("git", args...)
	defer cancel()
	output, err := executeWithOutput(cmd)
	if err != nil {
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
			return make(map[string][]string), nil
		}
		return nil, err
	}

	configs := make(map[string][]string)
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			key, value := parts[0], parts[1]
			configs[key] = append(configs[key], value)
		}
	}

	return configs, scanner.Err()
}

// SetConfig sets a config value, replacing any existing value
func SetConfig(key, value string, global bool) error {
	logger.Debug("Setting git config: %s=%s (global=%v)", key, value, global)

	args := []string{configCommand}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key, value)

	cmd, cancel := GitCommand("git", args...)
	defer cancel()
	_, err := executeWithOutput(cmd)
	return err
}

// SetBranchConfig sets a branch value in the repository's local config.
func SetBranchConfig(bareDir, branch, key, value string) error {
	logger.Debug("Setting git config: branch.%s.%s=%s (repo=%s)", branch, key, value, bareDir)

	if bareDir == "" || branch == "" || key == "" {
		return errors.New("repository path, branch and config key cannot be empty")
	}

	cmd, cancel := GitCommand("git", configCommand, "--local", "branch."+branch+"."+key, value)
	defer cancel()
	cmd.Dir = bareDir

	_, err := executeWithOutput(cmd)
	return err
}

// GetBranchConfigs reads a config value for all branches in one command.
func GetBranchConfigs(bareDir, key string) (map[string]string, error) {
	logger.Debug("Getting git configs with branch key: %s (repo=%s)", key, bareDir)

	if bareDir == "" || key == "" {
		return nil, errors.New("repository path and config key cannot be empty")
	}

	suffix := "." + strings.ToLower(key)
	cmd, cancel := GitCommand("git", configCommand, "--local", "--get-regexp", "^branch\\..+"+regexp.QuoteMeta(suffix)+"$")
	defer cancel()
	cmd.Dir = bareDir
	output, err := executeWithOutput(cmd)
	configs := make(map[string]string)
	if err != nil {
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
			return configs, nil
		}
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		name, value, found := strings.Cut(scanner.Text(), " ")
		if found {
			branch := strings.TrimSuffix(strings.TrimPrefix(name, "branch."), suffix)
			configs[branch] = value
		}
	}

	return configs, scanner.Err()
}

// UnsetConfig removes a config key and all its values
func UnsetConfig(key string, global bool) error {
	logger.Debug("Unsetting git config: %s (global=%v)", key, global)

	args := []string{configCommand, "--unset-all"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key)

	cmd, cancel := GitCommand("git", args...)
	defer cancel()
	_, err := executeWithOutput(cmd)
	if err != nil {
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 5 {
			return ErrConfigNotFound
		}
		return err
	}

	return nil
}

// UnsetConfigValue removes a specific value from a config key using pattern matching
func UnsetConfigValue(key, valuePattern string, global bool) error {
	logger.Debug("Unsetting git config value: %s=%s (global=%v)", key, valuePattern, global)

	args := []string{configCommand, "--unset-all", "--fixed-value"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key, valuePattern)

	cmd, cancel := GitCommand("git", args...)
	defer cancel()
	_, err := executeWithOutput(cmd)
	if err != nil {
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 5 {
			return ErrConfigNotFound
		}
		return err
	}

	return nil
}
