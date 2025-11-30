package git

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/sqve/grove/internal/logger"
)

// ErrConfigNotFound is returned when a config key is not found
var ErrConfigNotFound = errors.New("config key not found")

// IsConfigNotFoundError returns true if error indicates config not found
func IsConfigNotFoundError(err error) bool {
	return errors.Is(err, ErrConfigNotFound)
}

// GetConfig gets a single config value
func GetConfig(key string, global bool) (string, error) {
	logger.Debug("Getting git config: %s (global=%v)", key, global)

	args := []string{"config", "--get"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key)

	cmd, cancel := GitCommand("git", args...)
	defer cancel()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
			return "", ErrConfigNotFound
		}
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return "", err
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GetConfigs gets all config values for keys with a given prefix
func GetConfigs(prefix string, global bool) (map[string][]string, error) {
	logger.Debug("Getting git configs with prefix: %s (global=%v)", prefix, global)

	args := []string{"config", "--get-regexp"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, prefix)

	cmd, cancel := GitCommand("git", args...)
	defer cancel()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
			return make(map[string][]string), nil
		}
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}

	configs := make(map[string][]string)
	scanner := bufio.NewScanner(&stdout)
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

	args := []string{"config"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key, value)

	cmd, cancel := GitCommand("git", args...)
	defer cancel()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return err
	}

	return nil
}

// AddConfig adds a value to a multi-value config key
func AddConfig(key, value string, global bool) error {
	logger.Debug("Adding git config: %s=%s (global=%v)", key, value, global)

	args := []string{"config", "--add"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key, value)

	cmd, cancel := GitCommand("git", args...)
	defer cancel()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return err
	}

	return nil
}

// UnsetConfig removes a config key and all its values
func UnsetConfig(key string, global bool) error {
	logger.Debug("Unsetting git config: %s (global=%v)", key, global)

	args := []string{"config", "--unset-all"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key)

	cmd, cancel := GitCommand("git", args...)
	defer cancel()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 5 {
			return ErrConfigNotFound
		}
		if stderr.Len() > 0 {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return err
	}

	return nil
}

// UnsetConfigValue removes a specific value from a config key using pattern matching
func UnsetConfigValue(key, valuePattern string, global bool) error {
	logger.Debug("Unsetting git config value: %s=%s (global=%v)", key, valuePattern, global)

	args := []string{"config", "--unset-all", "--fixed-value"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key, valuePattern)

	cmd, cancel := GitCommand("git", args...)
	defer cancel()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 5 {
			return ErrConfigNotFound
		}
		if stderr.Len() > 0 {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return err
	}

	return nil
}
