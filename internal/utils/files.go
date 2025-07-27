package utils

import (
	"github.com/sqve/grove/internal/logger"
)

// IsHidden reports whether filename starts with a dot (hidden file on Unix systems).
func IsHidden(filename string) bool {
	if filename == "" {
		return false
	}

	isHidden := filename[0] == '.'

	// Only log when debug level is explicitly enabled to avoid overhead.
	if log := logger.GetGlobalLogger(); log != nil {
		log.Debug("file hidden status check", "filename", filename, "is_hidden", isHidden)
	}

	return isHidden
}
