package logger

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/sqve/grove/internal/config"
)

func TestDebugLogging(t *testing.T) {
	t.Run("does not output when debug disabled", func(t *testing.T) {
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		config.Global.Debug = false
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

		config.Global.Debug = true
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
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		config.Global.Plain = true
		Success("test message")

		_ = w.Close()
		os.Stdout = oldStdout

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
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		config.Global.Plain = false
		t.Setenv("GROVE_TEST_COLORS", "true")
		Success("test message with colors")

		_ = w.Close()
		os.Stdout = oldStdout

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

func TestInfoAndWarning(t *testing.T) {
	t.Run("info message contains correct symbol and colors", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		config.Global.Plain = false
		t.Setenv("GROVE_TEST_COLORS", "true")
		Info("info message")

		_ = w.Close()
		os.Stdout = oldStdout

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

	t.Run("warning message contains correct symbol and colors", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		config.Global.Plain = false
		t.Setenv("GROVE_TEST_COLORS", "true")
		Warning("warning message")

		_ = w.Close()
		os.Stdout = oldStdout

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

func TestWorktreeListItem(t *testing.T) {
	t.Run("formats current worktree with bullet in plain mode", func(t *testing.T) {
		config.Global.Plain = true
		defer func() { config.Global.Plain = false }()

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		WorktreeListItem("main", true, "[clean]", "=", 10)

		_ = w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "* main") {
			t.Errorf("expected '* main' in output, got: %s", output)
		}
		if !strings.Contains(output, "[clean]") {
			t.Errorf("expected '[clean]' in output, got: %s", output)
		}
	})

	t.Run("formats non-current worktree without bullet in plain mode", func(t *testing.T) {
		config.Global.Plain = true
		defer func() { config.Global.Plain = false }()

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		WorktreeListItem("feature", false, "[dirty]", "↑2", 10)

		_ = w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		if strings.Contains(output, "*") {
			t.Errorf("non-current worktree should not have '*', got: %s", output)
		}
		if !strings.Contains(output, "feature") {
			t.Errorf("expected 'feature' in output, got: %s", output)
		}
	})

	t.Run("formats current worktree with unicode bullet in colored mode", func(t *testing.T) {
		config.Global.Plain = false
		t.Setenv("GROVE_TEST_COLORS", "true")

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		WorktreeListItem("main", true, "[clean]", "=", 10)

		_ = w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "●") {
			t.Errorf("expected '●' bullet in colored output, got: %s", output)
		}
	})
}
