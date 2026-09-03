package logger

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

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
		stderrTerminal.Store(true)
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
		stderrTerminal.Store(true)
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

func TestSpinnerNonTerminal(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = oldStderr })

	config.SetPlain(false)
	Init(false, false)
	stderrTerminal.Store(false)
	spinner := StartSpinner("Gathering worktree status...")
	spinner.Stop()

	_ = w.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if got, want := buf.String(), "→ Gathering worktree status...\n"; got != want {
		t.Errorf("spinner output = %q, want %q", got, want)
	}
}

func TestSpinnerTerminal(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = oldStderr })

	config.SetPlain(false)
	Init(false, false)
	stderrTerminal.Store(true)
	spinner := StartSpinner("Gathering worktree status...")

	var buf bytes.Buffer
	firstFrame := make(chan struct{})
	drained := make(chan struct{})
	go func() {
		chunk := make([]byte, 64)
		n, _ := r.Read(chunk)
		buf.Write(chunk[:n])
		close(firstFrame)
		_, _ = io.Copy(&buf, r)
		close(drained)
	}()

	select {
	case <-firstFrame:
	case <-time.After(5 * time.Second):
		t.Fatal("no spinner frame written within 5s")
	}
	spinner.Stop()
	_ = w.Close()
	<-drained

	output := buf.String()
	if !strings.Contains(output, "\r") || !strings.ContainsAny(output, "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏") {
		t.Errorf("spinner output = %q, want animation", output)
	}
}
