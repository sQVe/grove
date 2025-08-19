package styles

import (
	"strings"
	"testing"

	"github.com/sqve/grove/internal/config"
)

func TestRenderWithPlainMode(t *testing.T) {
	config.Global.Plain = true
	result := Render(&Success, "test message")

	if result != "test message" {
		t.Errorf("Expected plain text, got %q", result)
	}
	if strings.Contains(result, "\033[") {
		t.Error("Plain mode should not contain ANSI escape codes")
	}
}

func TestRenderWithColors(t *testing.T) {
	config.Global.Plain = false
	t.Setenv("GROVE_TEST_COLORS", "true")

	result := Render(&Success, "test message")

	if result == "test message" {
		t.Error("Expected styled text, got plain text")
	}
	if !strings.Contains(result, "\033[") {
		t.Error("Colored mode should contain ANSI escape codes")
	}
}
