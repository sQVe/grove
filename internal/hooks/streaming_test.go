package hooks

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/testutil"
)

type errorWriter struct {
	errAfter int
	written  int
}

func (w *errorWriter) Write(p []byte) (int, error) {
	if w.written >= w.errAfter {
		return 0, errors.New("write error")
	}
	w.written += len(p)
	return len(p), nil
}

func testPrefixWriter(prefix string, target *bytes.Buffer) *prefixWriter {
	return newPrefixWriter(prefix, target, &sync.Mutex{})
}

func Test_prefixWriter(t *testing.T) {
	t.Run("single complete line", func(t *testing.T) {
		var buf bytes.Buffer
		pw := testPrefixWriter("[prefix]", &buf)

		n, err := pw.Write([]byte("line\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 5 {
			t.Errorf("expected n=5, got %d", n)
		}
		if buf.String() != "[prefix] line\n" {
			t.Errorf("expected '[prefix] line\\n', got %q", buf.String())
		}
	})

	t.Run("multiple lines in single write", func(t *testing.T) {
		var buf bytes.Buffer
		pw := testPrefixWriter("[prefix]", &buf)

		_, err := pw.Write([]byte("a\nb\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "[prefix] a\n[prefix] b\n"
		if buf.String() != expected {
			t.Errorf("expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("partial line buffered until flush", func(t *testing.T) {
		var buf bytes.Buffer
		pw := testPrefixWriter("[prefix]", &buf)

		_, err := pw.Write([]byte("partial"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if buf.String() != "" {
			t.Errorf("expected no output before flush, got %q", buf.String())
		}

		err = pw.Flush()
		if err != nil {
			t.Fatalf("flush error: %v", err)
		}

		expected := "[prefix] partial\n"
		if buf.String() != expected {
			t.Errorf("expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("split line across writes", func(t *testing.T) {
		var buf bytes.Buffer
		pw := testPrefixWriter("[prefix]", &buf)

		_, _ = pw.Write([]byte("hel"))
		if buf.String() != "" {
			t.Errorf("expected no output after first write, got %q", buf.String())
		}

		_, _ = pw.Write([]byte("lo\n"))
		expected := "[prefix] hello\n"
		if buf.String() != expected {
			t.Errorf("expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("flush on empty buffer does nothing", func(t *testing.T) {
		var buf bytes.Buffer
		pw := testPrefixWriter("[prefix]", &buf)

		err := pw.Flush()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf.String() != "" {
			t.Errorf("expected empty output, got %q", buf.String())
		}
	})

	t.Run("mixed complete and partial lines", func(t *testing.T) {
		var buf bytes.Buffer
		pw := testPrefixWriter("[prefix]", &buf)

		_, _ = pw.Write([]byte("a\nb"))

		if buf.String() != "[prefix] a\n" {
			t.Errorf("expected '[prefix] a\\n' after write, got %q", buf.String())
		}

		err := pw.Flush()
		if err != nil {
			t.Fatalf("flush error: %v", err)
		}

		expected := "[prefix] a\n[prefix] b\n"
		if buf.String() != expected {
			t.Errorf("expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("empty prefix works", func(t *testing.T) {
		var buf bytes.Buffer
		pw := testPrefixWriter("", &buf)

		_, _ = pw.Write([]byte("line\n"))

		if buf.String() != " line\n" {
			t.Errorf("expected ' line\\n', got %q", buf.String())
		}
	})

	t.Run("multiple flushes are idempotent", func(t *testing.T) {
		var buf bytes.Buffer
		pw := testPrefixWriter("[prefix]", &buf)

		_, _ = pw.Write([]byte("data"))
		_ = pw.Flush()
		_ = pw.Flush()

		expected := "[prefix] data\n"
		if buf.String() != expected {
			t.Errorf("expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("returns error when target writer fails during Write", func(t *testing.T) {
		ew := &errorWriter{errAfter: 0}
		pw := newPrefixWriter("[prefix]", ew, &sync.Mutex{})

		_, err := pw.Write([]byte("line\n"))
		if err == nil {
			t.Fatal("expected error from target writer")
		}
		if err.Error() != "write error" {
			t.Errorf("expected 'write error', got %q", err.Error())
		}
	})

	t.Run("returns error when target writer fails during Flush", func(t *testing.T) {
		ew := &errorWriter{errAfter: 0}
		pw := newPrefixWriter("[prefix]", ew, &sync.Mutex{})

		_, err := pw.Write([]byte("partial"))
		if err != nil {
			t.Fatalf("unexpected error on write: %v", err)
		}

		err = pw.Flush()
		if err == nil {
			t.Fatal("expected error from target writer on flush")
		}
		if err.Error() != "write error" {
			t.Errorf("expected 'write error', got %q", err.Error())
		}
	})
}

func TestRunAddHooksStreaming(t *testing.T) {
	logger.Init(true, false)
	config.SetPlain(true)

	t.Run("streams output with prefix", func(t *testing.T) {
		workDir := testutil.TempDir(t)
		var output bytes.Buffer

		commands := []string{"echo 'line 1'; echo 'line 2'"}
		result := RunAddHooksStreaming(workDir, commands, &output)

		if len(result.Succeeded) != 1 {
			t.Errorf("Expected 1 succeeded, got %d", len(result.Succeeded))
		}

		out := output.String()
		if !strings.Contains(out, "[echo") {
			t.Error("Expected prefix in output")
		}
		if !strings.Contains(out, "line 1") || !strings.Contains(out, "line 2") {
			t.Error("Expected hook output")
		}
	})

	t.Run("handles command without trailing newline", func(t *testing.T) {
		workDir := testutil.TempDir(t)
		var output bytes.Buffer

		commands := []string{"echo -n 'no newline'"}
		result := RunAddHooksStreaming(workDir, commands, &output)

		if len(result.Succeeded) != 1 {
			t.Errorf("Expected success, got %d succeeded", len(result.Succeeded))
		}

		out := output.String()
		if !strings.Contains(out, "no newline") {
			t.Error("Expected output from command without newline")
		}
	})

	t.Run("stops on first failure", func(t *testing.T) {
		workDir := testutil.TempDir(t)
		var output bytes.Buffer

		commands := []string{"echo 'first'", "false", "echo 'third'"}
		result := RunAddHooksStreaming(workDir, commands, &output)

		if len(result.Succeeded) != 1 {
			t.Errorf("Expected 1 succeeded, got %d", len(result.Succeeded))
		}
		if result.Failed == nil {
			t.Fatal("Expected failure")
		}
		if result.Failed.Command != "false" {
			t.Errorf("Expected 'false' to fail, got %q", result.Failed.Command)
		}
	})

	t.Run("returns empty result for empty commands", func(t *testing.T) {
		workDir := testutil.TempDir(t)
		var output bytes.Buffer

		result := RunAddHooksStreaming(workDir, nil, &output)

		if len(result.Succeeded) != 0 || result.Failed != nil {
			t.Error("Expected empty result")
		}
	})

	t.Run("captures exit code on failure", func(t *testing.T) {
		workDir := testutil.TempDir(t)
		var output bytes.Buffer

		commands := []string{"exit 42"}
		result := RunAddHooksStreaming(workDir, commands, &output)

		if result.Failed == nil {
			t.Fatal("Expected failure")
		}
		if result.Failed.ExitCode != 42 {
			t.Errorf("Expected exit code 42, got %d", result.Failed.ExitCode)
		}
	})

	t.Run("multiple commands all succeed", func(t *testing.T) {
		workDir := testutil.TempDir(t)
		var output bytes.Buffer

		commands := []string{"echo 'first'", "echo 'second'", "echo 'third'"}
		result := RunAddHooksStreaming(workDir, commands, &output)

		if len(result.Succeeded) != 3 {
			t.Errorf("Expected 3 succeeded, got %d", len(result.Succeeded))
		}
		if result.Failed != nil {
			t.Errorf("Expected no failure, got %v", result.Failed)
		}
		out := output.String()
		if !strings.Contains(out, "first") || !strings.Contains(out, "second") || !strings.Contains(out, "third") {
			t.Error("Expected all three outputs")
		}
	})

	t.Run("handles cmd.Start failure for invalid command", func(t *testing.T) {
		workDir := "/nonexistent/directory/that/does/not/exist"
		var output bytes.Buffer

		commands := []string{"echo hello"}
		result := RunAddHooksStreaming(workDir, commands, &output)

		if result.Failed == nil {
			t.Fatal("Expected failure for invalid workDir")
		}
		if result.Failed.ExitCode != 1 {
			t.Errorf("Expected exit code 1, got %d", result.Failed.ExitCode)
		}
		if result.Failed.Stderr == "" {
			t.Error("Expected error message in Stderr")
		}
	})
}
