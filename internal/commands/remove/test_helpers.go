package remove

import (
	"bytes"

	"github.com/sqve/grove/internal/logger"
)

// newTestLogger creates a logger suitable for testing.
func newTestLogger() *logger.Logger {
	buf := &bytes.Buffer{}
	config := logger.Config{
		Level:  "debug",
		Format: "text",
		Output: buf,
	}
	return logger.New(config)
}
