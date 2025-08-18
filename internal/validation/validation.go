package validation

import (
	"os"
)

// DirectoryExists checks if a directory exists
func DirectoryExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}

	return info.IsDir()
}

// IsEmptyDir checks if a directory is empty
func IsEmptyDir(path string) (bool, error) {
	f, err := os.Open(path) //nolint:gosec // Controlled path for workspace validation
	if err != nil {
		return false, err
	}
	defer func() {
		_ = f.Close()
	}()

	_, err = f.Readdirnames(1)
	if err == nil {
		return false, nil
	}

	// Check if error is EOF (empty directory)
	if err.Error() == "EOF" {
		return true, nil
	}

	return false, err
}
