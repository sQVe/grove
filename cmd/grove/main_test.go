package main

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/sqve/grove/internal/logger"
)

func TestHandleCommandError_PrintsRawExitError(t *testing.T) {
	err := exec.Command("sh", "-c", "exit 7").Run()
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		t.Fatalf("expected *exec.ExitError, got %T", err)
	}

	var output bytes.Buffer
	logger.SetOutput(&output)
	defer logger.SetOutput(nil)

	if exitCode := handleCommandError(err); exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(output.String(), "exit status 7") {
		t.Errorf("expected raw exit error to be printed, got %q", output.String())
	}
	if !strings.Contains(output.String(), "Run 'grove --help' for usage.") {
		t.Errorf("expected usage hint to be printed, got %q", output.String())
	}
}
