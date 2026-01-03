package logger

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestDebugLogging(t *testing.T) {
	t.Run("does not output when debug disabled", func(t *testing.T) {
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		Init(false, false) // plain=false, debug=false
		Debug("This should not appear")

		_ = w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		if strings.Contains(output, "This should not appear") {
			t.Error("Debug message appeared when debug mode was disabled")
		}
	})

	t.Run("outputs with prefix when debug enabled", func(t *testing.T) {
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		Init(false, true) // plain=false, debug=true
		Debug("This should appear")

		_ = w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "This should appear") {
			t.Error("Debug message did not appear when debug mode was enabled")
		}
		if !strings.Contains(output, "[DEBUG]") {
			t.Error("Debug prefix not found in output")
		}
	})
}

func TestPlainOutput(t *testing.T) {
	t.Run("plain mode without emoji or colors", func(t *testing.T) {
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		Init(true, false) // plain=true, debug=false
		Success("test message")

		_ = w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		if strings.Contains(output, "✓") {
			t.Error("Plain mode output should not contain emoji")
		}
		if strings.Contains(output, "\033[") {
			t.Error("Plain mode output should not contain ANSI escape codes")
		}
	})

	t.Run("colored mode with emoji and colors", func(t *testing.T) {
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		Init(false, false) // plain=false, debug=false
		t.Setenv("GROVE_TEST_COLORS", "true")
		Success("test message with colors")

		_ = w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "✓") {
			t.Error("Colored mode output should contain emoji")
		}
		if !strings.Contains(output, "\033[") {
			t.Error("Colored mode output should contain ANSI escape codes")
		}
	})
}

func TestInfo(t *testing.T) {
	t.Run("plain mode outputs message without symbols", func(t *testing.T) {
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		Init(true, false) // plain=true, debug=false
		Info("info message")

		_ = w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "info message") {
			t.Error("Info output should contain the message")
		}
		if strings.Contains(output, "\033[") {
			t.Error("Plain mode should not contain ANSI escape codes")
		}
	})

	t.Run("colored mode outputs with symbol and colors", func(t *testing.T) {
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		Init(false, false) // plain=false, debug=false
		t.Setenv("GROVE_TEST_COLORS", "true")
		Info("info message")

		_ = w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "→") {
			t.Error("Info output should contain info symbol")
		}
		if !strings.Contains(output, "\033[") {
			t.Error("Info output should contain ANSI escape codes")
		}
	})
}

func TestWarning(t *testing.T) {
	t.Run("plain mode outputs message without symbols", func(t *testing.T) {
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		Init(true, false) // plain=true, debug=false
		Warning("warning message")

		_ = w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "warning message") {
			t.Error("Warning output should contain the message")
		}
		if strings.Contains(output, "\033[") {
			t.Error("Plain mode should not contain ANSI escape codes")
		}
	})

	t.Run("colored mode outputs with symbol and colors", func(t *testing.T) {
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		Init(false, false) // plain=false, debug=false
		t.Setenv("GROVE_TEST_COLORS", "true")
		Warning("warning message")

		_ = w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "⚠") {
			t.Error("Warning output should contain warning symbol")
		}
		if !strings.Contains(output, "\033[") {
			t.Error("Warning output should contain ANSI escape codes")
		}
	})
}
