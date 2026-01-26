package logger

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/sqve/grove/internal/config"
)

func TestSpinnerUpdate(t *testing.T) {
	t.Run("Update stores new message", func(t *testing.T) {
		config.SetPlain(true)
		Init(true, false)
		spinner := StartSpinner("initial")
		spinner.Update("updated message")

		got := spinner.message.Load().(string)
		if got != "updated message" {
			t.Errorf("Update() = %q, want %q", got, "updated message")
		}
	})
}

func TestSpinnerStopIdempotent(t *testing.T) {
	t.Run("Stop can be called multiple times without panic", func(t *testing.T) {
		config.SetPlain(true)
		Init(true, false)
		spinner := StartSpinner("test")

		spinner.Stop()
		spinner.Stop()
		spinner.Stop()
	})
}

func TestSpinnerStopWithSuccess(t *testing.T) {
	t.Run("prints checkmark in normal mode", func(t *testing.T) {
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w
		t.Cleanup(func() { os.Stderr = oldStderr })

		config.SetPlain(false)
		Init(false, false)
		t.Setenv("GROVE_TEST_COLORS", "true")
		spinner := StartSpinner("working")
		spinner.StopWithSuccess("done successfully")

		_ = w.Close()

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "done successfully") {
			t.Error("StopWithSuccess should print the message")
		}
		if !strings.Contains(output, "✓") {
			t.Error("StopWithSuccess should print checkmark")
		}
	})
}

func TestSpinnerStopWithError(t *testing.T) {
	t.Run("prints X in normal mode", func(t *testing.T) {
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w
		t.Cleanup(func() { os.Stderr = oldStderr })

		config.SetPlain(false)
		Init(false, false)
		t.Setenv("GROVE_TEST_COLORS", "true")
		spinner := StartSpinner("working")
		spinner.StopWithError("something failed")

		_ = w.Close()

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "something failed") {
			t.Error("StopWithError should print the message")
		}
		if !strings.Contains(output, "✗") {
			t.Error("StopWithError should print X symbol")
		}
	})
}

func TestSpinnerPlainMode(t *testing.T) {
	t.Run("prints message once without animation", func(t *testing.T) {
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w
		t.Cleanup(func() { os.Stderr = oldStderr })

		config.SetPlain(true)
		Init(true, false)
		spinner := StartSpinner("Loading data")
		spinner.Stop()

		_ = w.Close()

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "Loading data") {
			t.Error("Plain mode should print message")
		}
	})

	t.Run("no ANSI codes in output", func(t *testing.T) {
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w
		t.Cleanup(func() { os.Stderr = oldStderr })

		config.SetPlain(true)
		Init(true, false)
		spinner := StartSpinner("test")
		spinner.Update("updated")
		spinner.Stop()

		_ = w.Close()

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		if strings.Contains(output, "\033[") {
			t.Error("Plain mode should not contain ANSI escape codes")
		}
		if strings.Contains(output, "⠋") {
			t.Error("Plain mode should not show spinner frames")
		}
	})
}
