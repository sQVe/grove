package config

import (
	"os"
	"testing"
)

func TestGlobalConfig(t *testing.T) {
	Global = struct {
		Plain bool
		Debug bool
	}{}

	if IsPlain() {
		t.Error("Expected plain mode to be false initially")
	}
	if IsDebug() {
		t.Error("Expected debug mode to be false initially")
	}

	Global.Plain = true
	if !IsPlain() {
		t.Error("Expected plain mode to be true after setting Global.Plain = true")
	}

	Global.Debug = true
	if !IsDebug() {
		t.Error("Expected debug mode to be true after setting Global.Debug = true")
	}
}

func TestLoadFromEnv(t *testing.T) {
	Global = struct {
		Plain bool
		Debug bool
	}{}
	_ = os.Unsetenv("GROVE_PLAIN")
	_ = os.Unsetenv("GROVE_DEBUG")

	LoadFromEnv()
	if IsPlain() || IsDebug() {
		t.Error("Expected both modes to be false with no env vars")
	}

	// Test GROVE_PLAIN
	_ = os.Setenv("GROVE_PLAIN", "1")
	LoadFromEnv()
	if !IsPlain() {
		t.Error("Expected plain mode to be true with GROVE_PLAIN=1")
	}

	// Reset and test GROVE_DEBUG
	Global = struct {
		Plain bool
		Debug bool
	}{}
	_ = os.Unsetenv("GROVE_PLAIN")
	_ = os.Setenv("GROVE_DEBUG", "1")
	LoadFromEnv()
	if !IsDebug() {
		t.Error("Expected debug mode to be true with GROVE_DEBUG=1")
	}

	// Test non-"1" values don't enable flags
	Global = struct {
		Plain bool
		Debug bool
	}{}
	_ = os.Setenv("GROVE_PLAIN", "true")
	_ = os.Setenv("GROVE_DEBUG", "yes")
	LoadFromEnv()
	if IsPlain() || IsDebug() {
		t.Error("Expected both modes to be false with non-'1' env values")
	}

	// Cleanup
	_ = os.Unsetenv("GROVE_PLAIN")
	_ = os.Unsetenv("GROVE_DEBUG")
}
