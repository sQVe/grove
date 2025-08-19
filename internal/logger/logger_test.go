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
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	config.Global.Debug = false
	Debug("This should not appear")

	config.Global.Debug = true
	Debug("This should appear")

	// Restore stderr and read output
	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if strings.Contains(output, "This should not appear") {
		t.Error("Debug message appeared when debug mode was disabled")
	}
	if !strings.Contains(output, "This should appear") {
		t.Error("Debug message did not appear when debug mode was enabled")
	}
	if !strings.Contains(output, "[DEBUG]") {
		t.Error("Debug prefix not found in output")
	}
}

func TestPlainOutput(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config.Global.Plain = true
	Success("test message")

	config.Global.Plain = false
	t.Setenv("GROVE_TEST_COLORS", "true")

	Success("test message with colors")

	// Restore stdout and read output
	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines of output, got %d", len(lines))
	}

	plainLine := lines[0]
	colorLine := lines[1]

	if strings.Contains(plainLine, "✓") {
		t.Error("Plain mode output should not contain emoji")
	}
	if strings.Contains(plainLine, "\033[") {
		t.Error("Plain mode output should not contain ANSI escape codes")
	}
	if !strings.Contains(colorLine, "✓") {
		t.Error("Colored mode output should contain emoji")
	}
	if !strings.Contains(colorLine, "\033[") {
		t.Error("Colored mode output should contain ANSI escape codes")
	}
}

func TestInfoAndWarning(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config.Global.Plain = false
	t.Setenv("GROVE_TEST_COLORS", "true")

	Info("info message")
	Warning("warning message")

	// Restore stdout and read output
	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "→") {
		t.Error("Info output should contain info symbol")
	}
	if !strings.Contains(output, "⚠") {
		t.Error("Warning output should contain warning symbol")
	}
	if !strings.Contains(output, "\033[") {
		t.Error("Colored output should contain ANSI escape codes")
	}
}
